package api

import (
	"bytes"
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
	info := struct {
		Containers      int
		DriverStatus    [][2]string
		NEventsListener int
		Debug           bool
	}{
		len(c.cluster.Containers()),
		c.cluster.Info(),
		c.eventsHandler.Size(),
		c.debug,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// GET /version
func getVersion(c *context, w http.ResponseWriter, r *http.Request) {
	version := struct {
		Version    string
		APIVersion string
		GoVersion  string
		GitCommit  string
		Os         string
		Arch       string
	}{
		Version:    "swarm/" + version.VERSION,
		APIVersion: APIVERSION,
		GoVersion:  runtime.Version(),
		GitCommit:  version.GITCOMMIT,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
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

	all := r.Form.Get("all") == "1"
	limit, _ := strconv.Atoi(r.Form.Get("limit"))

	out := []*dockerclient.Container{}
	for _, container := range c.cluster.Containers() {
		tmp := (*container).Container
		// Skip stopped containers unless -a was specified.
		if !strings.Contains(tmp.Status, "Up") && !all && limit <= 0 {
			continue
		}
		// Skip swarm containers unless -a was specified.
		if strings.Split(tmp.Image, ":")[0] == "swarm" && !all {
			continue
		}
		if !container.Engine.IsHealthy() {
			tmp.Status = "Pending"
		}
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

	sort.Sort(sort.Reverse(ContainerSorter(out)))

	w.Header().Set("Content-Type", "application/json")
	if limit > 0 && limit < len(out) {
		json.NewEncoder(w).Encode(out[:limit])
	} else {
		json.NewEncoder(w).Encode(out)
	}
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

	if container := c.cluster.Container(name); container != nil {
		httpError(w, fmt.Sprintf("Conflict, The name %s is already assigned to %s. You have to delete (or rename) that container to be able to assign %s to a container again.", name, container.Id, name), http.StatusConflict)
		return
	}

	container, err := c.cluster.CreateContainer(&config, name)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
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
	force := r.Form.Get("force") == "1"
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

	if image := r.Form.Get("fromImage"); image != "" { //pull
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

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
		c.cluster.Pull(image, callback)
	} else { //import
		httpError(w, "Not supported in clustering mode.", http.StatusNotImplemented)
	}
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

	matchedImages := []*cluster.Image{}
	for _, image := range c.cluster.Images() {
		if image.Match(name) {
			matchedImages = append(matchedImages, image)
		}
	}

	if len(matchedImages) == 0 {
		httpError(w, fmt.Sprintf("No such image %s", name), http.StatusNotFound)
		return
	}

	out := []*dockerclient.ImageDelete{}
	errs := []string{}
	for _, image := range matchedImages {
		content, err := c.cluster.RemoveImage(image)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", image.Engine.Name, err.Error()))
			continue
		}
		out = append(out, content...)
	}

	if len(errs) != 0 {
		httpError(w, strings.Join(errs, ""), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(NewWriteFlusher(w)).Encode(out)
	}

}

// GET /_ping
func ping(c *context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// Proxy a request to the right node
func proxyContainer(c *context, w http.ResponseWriter, r *http.Request) {
	container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := proxy(c.tlsConfig, container.Engine.Addr, w, r); err != nil {
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
	container, err := getContainerFromVars(c, vars)
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
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
	container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newName := r.Form.Get("name")

	// call cluster rename container
	err = c.cluster.RenameContainer(container, newName)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

}

// Proxy a hijack request to the right node
func proxyHijack(c *context, w http.ResponseWriter, r *http.Request) {
	container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
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
