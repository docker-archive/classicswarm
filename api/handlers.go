package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"

	dockerfilters "github.com/docker/docker/pkg/parsers/filters"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/version"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

// The Client API version
const APIVERSION = "1.16"

// GET /info
func getInfo(c *context, w http.ResponseWriter, r *http.Request) {
	info := dockerclient.Info{
		Containers:      int64(len(c.cluster.Containers())),
		Images:          int64(len(c.cluster.Images())),
		DriverStatus:    c.statusHandler.Status(),
		NEventsListener: int64(c.eventsHandler.Size()),
		Debug:           c.debug,
		MemoryLimit:     true,
		SwapLimit:       true,
		IPv4Forwarding:  true,
		NCPU:            c.cluster.TotalCpus(),
		MemTotal:        c.cluster.TotalMemory(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// GET /version
func getVersion(c *context, w http.ResponseWriter, r *http.Request) {
	version := dockerclient.Version{
		Version:    "swarm/" + version.VERSION,
		ApiVersion: APIVERSION,
		GoVersion:  runtime.Version(),
		GitCommit:  version.GITCOMMIT,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// GET /images/{name:.*}/get
func getImage(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	for _, image := range c.cluster.Images() {
		if len(strings.SplitN(name, ":", 2)) == 2 && image.Match(name, true) ||
			len(strings.SplitN(name, ":", 2)) == 1 && image.Match(name, false) {
			proxy(c.tlsConfig, image.Engine.Addr, w, r)
			return
		}
	}
	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// GET /images/get
func getImages(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names := r.Form["names"]

	// Create a map of engine address to the list of images it holds.
	engineImages := make(map[string][]*cluster.Image)
	for _, image := range c.cluster.Images() {
		engineImages[image.Engine.Addr] = append(engineImages[image.Engine.Addr], image)
	}

	// Look for an engine that has all the images we need.
	for engine, images := range engineImages {
		matchedImages := 0

		// Count how many images we need it has.
		for _, name := range names {
			for _, image := range images {
				if len(strings.SplitN(name, ":", 2)) == 2 && image.Match(name, true) ||
					len(strings.SplitN(name, ":", 2)) == 1 && image.Match(name, false) {
					matchedImages = matchedImages + 1
					break
				}
			}
		}

		// If the engine has all images, stop our search here.
		if matchedImages == len(names) {
			proxy(c.tlsConfig, engine, w, r)
			return
		}
	}

	httpError(w, fmt.Sprintf("Unable to find an engine containing all images: %s", names), http.StatusNotFound)
}

// GET /images/json
func getImagesJSON(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filters, err := dockerfilters.FromParam(r.Form.Get("filters"))
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accepteds, _ := filters["node"]
	images := []*cluster.Image{}

	for _, image := range c.cluster.Images() {
		if len(accepteds) != 0 {
			found := false
			for _, accepted := range accepteds {
				if accepted == image.Engine.Name || accepted == image.Engine.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		images = append(images, image)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

// GET /containers/ps
// GET /containers/json
func getContainersJSON(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse flags.
	all := boolValue(r, "all")
	limit := intValueOrZero(r, "limit")

	// Parse filters.
	filters, err := dockerfilters.FromParam(r.Form.Get("filters"))
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filtExited := []int{}
	if i, ok := filters["exited"]; ok {
		for _, value := range i {
			code, err := strconv.Atoi(value)
			if err != nil {
				httpError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			filtExited = append(filtExited, code)
		}
	}
	if i, ok := filters["status"]; ok {
		for _, value := range i {
			if value == "exited" {
				all = true
			}
		}
	}

	// Filtering: select the containers we want to return.
	candidates := []*cluster.Container{}
	for _, container := range c.cluster.Containers() {
		// Skip stopped containers unless -a was specified.
		if !container.Info.State.Running && !all && limit <= 0 {
			continue
		}

		// Skip swarm containers unless -a was specified.
		if strings.Split(container.Image, ":")[0] == "swarm" && !all {
			continue
		}

		// Apply filters.
		if !filters.Match("name", strings.TrimPrefix(container.Names[0], "/")) {
			continue
		}
		if !filters.Match("id", container.Id) {
			continue
		}
		if !filters.MatchKVList("label", container.Config.Labels) {
			continue
		}
		if !filters.Match("status", container.Info.State.StateString()) {
			continue
		}

		if len(filtExited) > 0 {
			shouldSkip := true
			for _, code := range filtExited {
				if code == container.Info.State.ExitCode && !container.Info.State.Running {
					shouldSkip = false
					break
				}
			}
			if shouldSkip {
				continue
			}
		}

		candidates = append(candidates, container)
	}

	// Sort the candidates and apply limits.
	sort.Sort(sort.Reverse(ContainerSorter(candidates)))
	if limit > 0 && limit < len(candidates) {
		candidates = candidates[:limit]
	}

	// Convert cluster.Container back into dockerclient.Container.
	out := []*dockerclient.Container{}
	for _, container := range candidates {
		// Create a copy of the underlying dockerclient.Container so we can
		// make changes without messing with cluster.Container.
		tmp := (*container).Container

		// Update the Status. The one we have is stale from the last `docker ps` the engine sent.
		// `Status()` will generate a new one
		tmp.Status = container.Info.State.String()
		if !container.Engine.IsHealthy() {
			tmp.Status = "Host Down"
		}

		// Overwrite labels with the ones we have in the config.
		// This ensures that we can freely manipulate them in the codebase and
		// they will be properly exported back (for instance Swarm IDs).
		tmp.Labels = container.Config.Labels

		// TODO remove the Node Name in the name when we have a good solution
		tmp.Names = make([]string, len(container.Names))
		for i, name := range container.Names {
			tmp.Names[i] = "/" + container.Engine.Name + name
		}

		// insert node IP
		tmp.Ports = make([]dockerclient.Port, len(container.Ports))
		for i, port := range container.Ports {
			tmp.Ports[i] = port
			if port.IP == "0.0.0.0" {
				tmp.Ports[i].IP = container.Engine.IP
			}
		}
		out = append(out, &tmp)
	}

	// Finally, send them back to the CLI.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// GET /containers/{name:.*}/json
func getContainerJSON(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container %s", name), http.StatusNotFound)
		return
	}
	client, scheme := newClientAndScheme(c.tlsConfig)

	resp, err := client.Get(scheme + "://" + container.Engine.Addr + "/containers/" + container.Id + "/json")
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// cleanup
	defer resp.Body.Close()
	defer closeIdleConnections(client)

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	n, err := json.Marshal(container.Engine)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// insert Node field
	data = bytes.Replace(data, []byte("\"Name\":\"/"), []byte(fmt.Sprintf("\"Node\":%s,\"Name\":\"/", n)), -1)

	// insert node IP
	data = bytes.Replace(data, []byte("\"HostIp\":\"0.0.0.0\""), []byte(fmt.Sprintf("\"HostIp\":%q", container.Engine.IP)), -1)

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// POST /containers/create
func postContainersCreate(c *context, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var (
		config dockerclient.ContainerConfig
		name   = r.Form.Get("name")
	)

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	container, err := c.cluster.CreateContainer(cluster.BuildContainerConfig(config), name)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Conflict") {
			httpError(w, err.Error(), http.StatusConflict)
		} else {
			httpError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
	return
}

// DELETE /containers/{name:.*}
func deleteContainers(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := mux.Vars(r)["name"]
	force := boolValue(r, "force")
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := c.cluster.RemoveContainer(container, force); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST  /images/create
func postImagesCreate(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wf := NewWriteFlusher(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if image := r.Form.Get("fromImage"); image != "" { //pull
		authConfig := dockerclient.AuthConfig{}
		buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
		if err == nil {
			json.Unmarshal(buf, &authConfig)
		}

		if tag := r.Form.Get("tag"); tag != "" {
			image += ":" + tag
		}
		callback := func(what, status string) {
			if status == "" {
				fmt.Fprintf(wf, "{%q:%q,%q:\"Pulling %s...\",%q:{}}", "id", what, "status", image, "progressDetail")
			} else {
				fmt.Fprintf(wf, "{%q:%q,%q:\"Pulling %s... : %s\",%q:{}}", "id", what, "status", image, status, "progressDetail")
			}
		}
		c.cluster.Pull(image, &authConfig, callback)
	} else { //import
		source := r.Form.Get("fromSrc")
		repo := r.Form.Get("repo")
		tag := r.Form.Get("tag")

		callback := func(what, status string) {
			fmt.Fprintf(wf, "{%q:%q,%q:\"%s\"}", "id", what, "status", status)
		}
		c.cluster.Import(source, repo, tag, r.Body, callback)
	}
}

// POST /images/load
func postImagesLoad(c *context, w http.ResponseWriter, r *http.Request) {

	// call cluster to load image on every node
	wf := NewWriteFlusher(w)
	callback := func(what, status string) {
		if status == "" {
			fmt.Fprintf(wf, "%s:Loading Image...\n", what)
		} else {
			fmt.Fprintf(wf, "%s:Loading Image... %s\n", what, status)
		}
	}
	c.cluster.Load(r.Body, callback)
}

// GET /events
func getEvents(c *context, w http.ResponseWriter, r *http.Request) {
	c.eventsHandler.Add(r.RemoteAddr, w)

	w.Header().Set("Content-Type", "application/json")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	c.eventsHandler.Wait(r.RemoteAddr)
}

// POST /containers/{name:.*}/exec
func postContainersExec(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container %s", name), http.StatusNotFound)
		return
	}

	client, scheme := newClientAndScheme(c.tlsConfig)

	resp, err := client.Post(scheme+"://"+container.Engine.Addr+"/containers/"+container.Id+"/exec", "application/json", r.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// cleanup
	defer resp.Body.Close()
	defer closeIdleConnections(client)

	// check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httpError(w, string(body), http.StatusInternalServerError)
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := struct{ ID string }{}

	if err := json.Unmarshal(data, &id); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add execID to the container, so the later exec/start will work
	container.Info.ExecIDs = append(container.Info.ExecIDs, id.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// DELETE /images/{name:.*}
func deleteImages(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var name = mux.Vars(r)["name"]

	out, err := c.cluster.RemoveImages(name)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(out) == 0 {
		httpError(w, fmt.Sprintf("No such image %s", name), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(NewWriteFlusher(w)).Encode(out)
}

// GET /_ping
func ping(c *context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// Proxy a request to the right node
func proxyContainer(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.Id, 1)
	}

	if err := proxy(c.tlsConfig, container.Engine.Addr, w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a request to the right node and force refresh container
func proxyContainerAndForceRefresh(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.Id, 1)
	}

	cb := func(resp *http.Response) {
		// force fresh container
		container.Refresh()
	}

	if err := proxyAsync(c.tlsConfig, container.Engine.Addr, w, r, cb); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a request to the right node
func proxyImage(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if image := c.cluster.Image(name); image != nil {
		proxy(c.tlsConfig, image.Engine.Addr, w, r)
		return
	}
	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// Proxy a request to the right node and force refresh
func proxyImageAndForceRefresh(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	// get image by name
	image := c.cluster.Image(name)

	if image == nil {
		httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
		return
	}

	cb := func(resp *http.Response) {
		if resp.StatusCode == http.StatusCreated {
			image.Engine.RefreshImages()
		}
	}

	if err := proxyAsync(c.tlsConfig, image.Engine.Addr, w, r, cb); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

}

// Proxy a request to a random node
func proxyRandom(c *context, w http.ResponseWriter, r *http.Request) {
	engine, err := c.cluster.RANDOMENGINE()
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if engine == nil {
		httpError(w, "no node available in the cluster", http.StatusInternalServerError)
		return
	}

	if err := proxy(c.tlsConfig, engine.Addr, w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

}

// Proxy a request to a random node and force refresh
func proxyRandomAndForceRefresh(c *context, w http.ResponseWriter, r *http.Request) {
	engine, err := c.cluster.RANDOMENGINE()
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if engine == nil {
		httpError(w, "no node available in the cluster", http.StatusInternalServerError)
		return
	}

	cb := func(resp *http.Response) {
		if resp.StatusCode == http.StatusOK {
			engine.RefreshImages()
		}
	}

	if err := proxyAsync(c.tlsConfig, engine.Addr, w, r, cb); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

}

// POST  /commit
func postCommit(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars := make(map[string]string)
	vars["name"] = r.Form.Get("container")

	// get container
	name, container, err := getContainerFromVars(c, vars)
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}
	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.RawQuery = strings.Replace(r.URL.RawQuery, name, container.Id, 1)
	}

	cb := func(resp *http.Response) {
		if resp.StatusCode == http.StatusCreated {
			container.Engine.RefreshImages()
		}
	}

	// proxy commit request to the right node
	if err := proxyAsync(c.tlsConfig, container.Engine.Addr, w, r, cb); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /containers/{name:.*}/rename
func postRenameContainer(c *context, w http.ResponseWriter, r *http.Request) {
	_, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = c.cluster.RenameContainer(container, r.Form.Get("name")); err != nil {
		if strings.HasPrefix(err.Error(), "Conflict") {
			httpError(w, err.Error(), http.StatusConflict)
		} else {
			httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}

}

// Proxy a hijack request to the right node
func proxyHijack(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}
	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.Id, 1)
	}

	if err := hijack(c.tlsConfig, container.Engine.Addr, w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Default handler for methods not supported by clustering.
func notImplementedHandler(c *context, w http.ResponseWriter, r *http.Request) {
	httpError(w, "Not supported in clustering mode.", http.StatusNotImplemented)
}

func optionsHandler(c *context, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
