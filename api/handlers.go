package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/parsers/kernel"
	versionpkg "github.com/docker/docker/pkg/version"
	apitypes "github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	dockerfilters "github.com/docker/engine-api/types/filters"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/experimental"
	"github.com/docker/swarm/version"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

// APIVERSION is the API version supported by swarm manager
const APIVERSION = "1.22"

// GET /info
func getInfo(c *context, w http.ResponseWriter, r *http.Request) {
	info := apitypes.Info{
		Images:            len(c.cluster.Images().Filter(cluster.ImageFilterOptions{})),
		NEventsListener:   c.eventsHandler.Size(),
		Debug:             c.debug,
		MemoryLimit:       true,
		SwapLimit:         true,
		CPUCfsPeriod:      true,
		CPUCfsQuota:       true,
		CPUShares:         true,
		CPUSet:            true,
		IPv4Forwarding:    true,
		BridgeNfIptables:  true,
		BridgeNfIP6tables: true,
		OomKillDisable:    true,
		ServerVersion:     "swarm/" + version.VERSION,
		OperatingSystem:   runtime.GOOS,
		Architecture:      runtime.GOARCH,
		NCPU:              c.cluster.TotalCpus(),
		MemTotal:          c.cluster.TotalMemory(),
		HTTPProxy:         os.Getenv("http_proxy"),
		HTTPSProxy:        os.Getenv("https_proxy"),
		NoProxy:           os.Getenv("no_proxy"),
		SystemTime:        time.Now().Format(time.RFC3339Nano),
		ExperimentalBuild: experimental.ENABLED,
	}

	// API versions older than 1.22 use DriverStatus and return \b characters in the output
	status := c.statusHandler.Status()
	if c.apiVersion != "" && versionpkg.Version(c.apiVersion).LessThan("1.22") {
		for i := range status {
			if status[i][0][:1] == " " {
				status[i][0] = status[i][0][1:]
			} else {
				status[i][0] = "\b" + status[i][0]
			}
		}
		info.DriverStatus = status
	} else {
		info.SystemStatus = status
	}

	kernelVersion := "<unknown>"
	if kv, err := kernel.GetKernelVersion(); err != nil {
		log.Warnf("Could not get kernel version: %v", err)
	} else {
		kernelVersion = kv.String()
	}
	info.KernelVersion = kernelVersion

	for _, c := range c.cluster.Containers() {
		info.Containers++
		if c.Info.State.Paused {
			info.ContainersPaused++
		} else if c.Info.State.Running {
			info.ContainersRunning++
		} else {
			info.ContainersStopped++
		}
	}

	hostname := "<unknown>"
	if hn, err := os.Hostname(); err != nil {
		log.Warnf("Could not get hostname: %v", err)
	} else {
		hostname = hn
	}
	info.Name = hostname

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// GET /version
func getVersion(c *context, w http.ResponseWriter, r *http.Request) {
	version := apitypes.Version{
		Version:      "swarm/" + version.VERSION,
		APIVersion:   APIVERSION,
		GoVersion:    runtime.Version(),
		GitCommit:    version.GITCOMMIT,
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Experimental: experimental.ENABLED,
		BuildTime:    version.BUILDTIME,
	}

	kernelVersion := "<unknown>"
	if kv, err := kernel.GetKernelVersion(); err != nil {
		log.Warnf("Could not get kernel version: %v", err)
	} else {
		kernelVersion = kv.String()
	}
	version.KernelVersion = kernelVersion

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// GET /images/get
func getImages(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names := r.Form["names"]

	// Create a map of engine address to the list of images it holds.
	engineImages := make(map[*cluster.Engine][]*cluster.Image)
	for _, image := range c.cluster.Images() {
		engineImages[image.Engine] = append(engineImages[image.Engine], image)
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
			proxy(engine, w, r)
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

	// TODO: apply node filter in engine?
	accepteds := filters.Get("node")
	// this struct helps grouping images
	// but still keeps their Engine infos as an array.
	groupImages := make(map[string]apitypes.Image)
	opts := cluster.ImageFilterOptions{
		apitypes.ImageListOptions{
			All:       boolValue(r, "all"),
			MatchName: r.FormValue("filter"),
			Filters:   filters,
		},
	}
	for _, image := range c.cluster.Images().Filter(opts) {
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

		// grouping images by Id, and concat their RepoTags
		if entry, existed := groupImages[image.ID]; existed {
			entry.RepoTags = append(entry.RepoTags, image.RepoTags...)
			entry.RepoDigests = append(entry.RepoDigests, image.RepoDigests...)
			groupImages[image.ID] = entry
		} else {
			groupImages[image.ID] = image.Image
		}
	}

	images := []apitypes.Image{}

	for _, image := range groupImages {
		// de-duplicate RepoTags
		result := []string{}
		seen := map[string]bool{}
		for _, val := range image.RepoTags {
			if _, ok := seen[val]; !ok {
				result = append(result, val)
				seen[val] = true
			}
		}
		image.RepoTags = result

		// de-duplicate RepoDigests
		result = []string{}
		seen = map[string]bool{}
		for _, val := range image.RepoDigests {
			if _, ok := seen[val]; !ok {
				result = append(result, val)
				seen[val] = true
			}
		}
		image.RepoDigests = result

		images = append(images, image)
	}
	sort.Sort(sort.Reverse(ImageSorter(images)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

// GET /networks
func getNetworks(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filters, err := dockerfilters.FromParam(r.Form.Get("filters"))
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	types := filters.Get("type")
	for _, typ := range types {
		if typ != "custom" && typ != "builtin" {
			httpError(w, fmt.Sprintf("Invalid filter: 'type'='%s'", typ), http.StatusBadRequest)
			return
		}
	}

	out := []*apitypes.NetworkResource{}
	networks := c.cluster.Networks().Filter(filters.Get("name"), filters.Get("id"), types)
	for _, network := range networks {
		tmp := (*network).NetworkResource
		if tmp.Scope == "local" {
			tmp.Name = network.Engine.Name + "/" + network.Name
		}
		out = append(out, &tmp)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// GET /networks/{networkid:.*}
func getNetwork(c *context, w http.ResponseWriter, r *http.Request) {
	var id = mux.Vars(r)["networkid"]
	if network := c.cluster.Networks().Uniq().Get(id); network != nil {
		// there could be duplicate container endpoints in network, need to remove redundant
		// see https://github.com/docker/swarm/issues/1969
		cleanNetwork := network.RemoveDuplicateEndpoints()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cleanNetwork)
		return
	}
	httpError(w, fmt.Sprintf("No such network: %s", id), http.StatusNotFound)
}

// GET /volumes/{volumename:.*}
func getVolume(c *context, w http.ResponseWriter, r *http.Request) {
	var name = mux.Vars(r)["volumename"]
	if volume := c.cluster.Volumes().Get(name); volume != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volume)
		return
	}
	httpError(w, fmt.Sprintf("No such volume: %s", name), http.StatusNotFound)
}

// GET /volumes
func getVolumes(c *context, w http.ResponseWriter, r *http.Request) {
	volumes := struct{ Volumes []*apitypes.Volume }{}

	for _, volume := range c.cluster.Volumes() {
		tmp := (*volume).Volume
		if tmp.Driver == "local" {
			tmp.Name = volume.Engine.Name + "/" + volume.Name
		}
		volumes.Volumes = append(volumes.Volumes, &tmp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(volumes)
}

// GET /containers/ps
// GET /containers/json
func getContainersJSON(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse flags.
	var (
		all    = boolValue(r, "all")
		limit  = intValueOrZero(r, "limit")
		before *cluster.Container
	)
	if value := r.FormValue("before"); value != "" {
		before = c.cluster.Container(value)
		if before == nil {
			httpError(w, fmt.Sprintf("No such container %s", value), http.StatusNotFound)
			return
		}
	}

	// Parse filters.
	filters, err := dockerfilters.FromParam(r.Form.Get("filters"))
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filtExited := []int{}
	for _, value := range filters.Get("exited") {
		code, err := strconv.Atoi(value)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filtExited = append(filtExited, code)
	}
	for _, value := range filters.Get("status") {
		if value == "exited" {
			all = true
		}
	}

	// Filtering: select the containers we want to return.
	candidates := []*cluster.Container{}
	for _, container := range c.cluster.Containers() {
		// Skip stopped containers unless -a was specified
		if (!container.Info.State.Running || !container.Engine.IsHealthy()) && !all && before == nil && limit <= 0 {
			continue
		}

		// Skip swarm containers unless -a was specified.
		if strings.Split(container.Image, ":")[0] == "swarm" && !all {
			continue
		}

		// Apply filters.
		if len(container.Names) > 0 {
			if !filters.Match("name", strings.TrimPrefix(container.Names[0], "/")) {
				continue
			}
		} else if len(filters.Get("name")) > 0 {
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
		if !filters.Match("node", container.Engine.Name) {
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
		if before != nil {
			if container.Id == before.Id {
				before = nil
			}
			continue
		}
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

	if !container.Engine.IsHealthy() {
		httpError(w, fmt.Sprintf("Container %s running on unhealthy node %s", name, container.Engine.Name), http.StatusInternalServerError)
		return
	}

	client, scheme, err := container.Engine.HTTPClientAndScheme()
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := client.Get(scheme + "://" + container.Engine.Addr + "/containers/" + container.Id + "/json")
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// cleanup
	defer resp.Body.Close()

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
		config = dockerclient.ContainerConfig{
			HostConfig: dockerclient.HostConfig{
				MemorySwappiness: -1,
			},
		}
		name = r.Form.Get("name")
	)

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Pass auth information along if present
	var authConfig *dockerclient.AuthConfig
	buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
	if err == nil {
		authConfig = &dockerclient.AuthConfig{}
		json.Unmarshal(buf, authConfig)
	}
	containerConfig := cluster.BuildContainerConfig(config)
	if err := containerConfig.Validate(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	container, err := c.cluster.CreateContainer(containerConfig, name, authConfig)
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
	volumes := boolValue(r, "v")
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := c.cluster.RemoveContainer(container, force, volumes); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /networks/create
func postNetworksCreate(c *context, w http.ResponseWriter, r *http.Request) {
	var request apitypes.NetworkCreate

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Driver == "" {
		request.Driver = "overlay"
	}

	response, err := c.cluster.CreateNetwork(&request)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// POST /volumes/create
func postVolumesCreate(c *context, w http.ResponseWriter, r *http.Request) {
	var request apitypes.VolumeCreateRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	volume, err := c.cluster.CreateVolume(&request)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(volume)
}

// POST  /images/create
func postImagesCreate(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wf := NewWriteFlusher(w)
	w.Header().Set("Content-Type", "application/json")

	if image := r.Form.Get("fromImage"); image != "" { //pull
		authConfig := dockerclient.AuthConfig{}
		buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
		if err == nil {
			json.Unmarshal(buf, &authConfig)
		}

		if tag := r.Form.Get("tag"); tag != "" {
			if tagHasDigest(tag) {
				image += "@" + tag
			} else {
				image += ":" + tag
			}
		}

		var errorMessage string
		errorFound := false
		callback := func(what, status string, err error) {
			if err != nil {
				errorFound = true
				errorMessage = err.Error()
				sendJSONMessage(wf, what, fmt.Sprintf("Pulling %s... : %s", image, err.Error()))
				return
			}
			if status == "" {
				sendJSONMessage(wf, what, fmt.Sprintf("Pulling %s...", image))
			} else {
				sendJSONMessage(wf, what, fmt.Sprintf("Pulling %s... : %s", image, status))
			}
		}
		c.cluster.Pull(image, &authConfig, callback)

		if errorFound {
			sendErrorJSONMessage(wf, 1, errorMessage)
		}

	} else { //import
		source := r.Form.Get("fromSrc")
		repo := r.Form.Get("repo")
		tag := r.Form.Get("tag")

		var errorMessage string
		errorFound := false
		callback := func(what, status string, err error) {
			if err != nil {
				errorFound = true
				errorMessage = err.Error()
				sendJSONMessage(wf, what, err.Error())
				return
			}
			sendJSONMessage(wf, what, status)
		}
		c.cluster.Import(source, repo, tag, r.Body, callback)
		if errorFound {
			sendErrorJSONMessage(wf, 1, errorMessage)
		}

	}
}

// POST /images/load
func postImagesLoad(c *context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// call cluster to load image on every node
	wf := NewWriteFlusher(w)
	var errorMessage string
	errorFound := false
	callback := func(what, status string, err error) {
		if err != nil {
			errorFound = true
			errorMessage = err.Error()
			sendJSONMessage(wf, what, fmt.Sprintf("Loading Image... : %s", err.Error()))
			return
		}

		if status == "" {
			sendJSONMessage(wf, what, "Loading Image...")
		} else {
			sendJSONMessage(wf, what, fmt.Sprintf("Loading Image... : %s", status))
		}
	}
	c.cluster.Load(r.Body, callback)
	if errorFound {
		sendErrorJSONMessage(wf, 1, errorMessage)
	}

}

// GET /events
func getEvents(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), 400)
		return
	}

	var until int64 = -1
	if r.Form.Get("until") != "" {
		u, err := strconv.ParseInt(r.Form.Get("until"), 10, 64)
		if err != nil {
			httpError(w, err.Error(), 400)
			return
		}
		until = u
	}

	c.eventsHandler.Add(r.RemoteAddr, w)

	w.Header().Set("Content-Type", "application/json")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	c.eventsHandler.Wait(r.RemoteAddr, until)
}

// POST /containers/{name:.*}/start
func postContainersStart(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container %s", name), http.StatusNotFound)
		return
	}

	hostConfig := &dockerclient.HostConfig{
		MemorySwappiness: -1,
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if len(buf) <= 2 {
		hostConfig = nil
	} else {
		if err := json.Unmarshal(buf, hostConfig); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := c.cluster.StartContainer(container, hostConfig); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /exec/{execid:.*}/start
func postExecStart(c *context, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Connection") == "" {
		proxyContainer(c, w, r)
	}
	proxyHijack(c, w, r)
}

// POST /containers/{name:.*}/exec
func postContainersExec(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container %s", name), http.StatusNotFound)
		return
	}

	client, scheme, err := container.Engine.HTTPClientAndScheme()
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := client.Post(scheme+"://"+container.Engine.Addr+"/containers/"+container.Id+"/exec", "application/json", r.Body)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// cleanup
	defer resp.Body.Close()

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
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

// DELETE /images/{name:.*}
func deleteImages(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var name = mux.Vars(r)["name"]
	force := boolValue(r, "force")

	out, err := c.cluster.RemoveImages(name, force)
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

// DELETE /networks/{networkid:.*}
func deleteNetworks(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var id = mux.Vars(r)["networkid"]

	if network := c.cluster.Networks().Uniq().Get(id); network != nil {
		if err := c.cluster.RemoveNetwork(network); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		httpError(w, fmt.Sprintf("No such network %s", id), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /volumes/{names:.*}
func deleteVolumes(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var name = mux.Vars(r)["name"]

	found, err := c.cluster.RemoveVolumes(name)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !found {
		httpError(w, fmt.Sprintf("No such volume %s", name), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /_ping
func ping(c *context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// POST /networks/{networkid:.*}/disconnect
func proxyNetworkDisconnect(c *context, w http.ResponseWriter, r *http.Request) {
	var networkid = mux.Vars(r)["networkid"]
	network := c.cluster.Networks().Uniq().Get(networkid)
	if network == nil {
		httpError(w, fmt.Sprintf("No such network: %s", networkid), http.StatusNotFound)
		return
	}
	// Set the network ID in the proxied URL path.
	r.URL.Path = strings.Replace(r.URL.Path, networkid, network.ID, 1)

	// make a copy of r.Body
	buf, _ := ioutil.ReadAll(r.Body)
	bodyCopy := ioutil.NopCloser(bytes.NewBuffer(buf))
	defer bodyCopy.Close()
	// restore r.Body stream as it'll be read again
	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	// Extract container info from r.Body copy
	var disconnect apitypes.NetworkDisconnect
	if err := json.NewDecoder(bodyCopy).Decode(&disconnect); err != nil {
		httpError(w, fmt.Sprintf("Container is not specified"), http.StatusNotFound)
		return
	}

	var engine *cluster.Engine

	if disconnect.Force && network.Scope == "global" {
		randomEngine, err := c.cluster.RANDOMENGINE()
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		engine = randomEngine
	} else {
		container := c.cluster.Container(disconnect.Container)
		if container == nil {
			httpError(w, fmt.Sprintf("No such container: %s", disconnect.Container), http.StatusNotFound)
			return
		}
		engine = container.Engine
	}

	cb := func(resp *http.Response) {
		// force fresh networks on this engine
		engine.RefreshNetworks()
	}

	// request is forwarded to the container's address
	err := proxyAsync(engine, w, r, cb)
	engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
	}
}

// POST /networks/{networkid:.*}/connect
func proxyNetworkConnect(c *context, w http.ResponseWriter, r *http.Request) {
	var networkid = mux.Vars(r)["networkid"]
	network := c.cluster.Networks().Uniq().Get(networkid)
	if network == nil {
		httpError(w, fmt.Sprintf("No such network: %s", networkid), http.StatusNotFound)
		return
	}
	// Set the network ID in the proxied URL path.
	r.URL.Path = strings.Replace(r.URL.Path, networkid, network.ID, 1)

	// make a copy of r.Body
	buf, _ := ioutil.ReadAll(r.Body)
	bodyCopy := ioutil.NopCloser(bytes.NewBuffer(buf))
	defer bodyCopy.Close()
	// restore r.Body stream as it'll be read again
	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	// Extract container info from r.Body copy
	var connect apitypes.NetworkConnect
	if err := json.NewDecoder(bodyCopy).Decode(&connect); err != nil {
		httpError(w, fmt.Sprintf("Container is not specified"), http.StatusNotFound)
		return
	}
	container := c.cluster.Container(connect.Container)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container: %s", connect.Container), http.StatusNotFound)
		return
	}

	cb := func(resp *http.Response) {
		// force fresh networks on this engine
		container.Engine.RefreshNetworks()
	}

	// request is forwarded to the container's address
	err := proxyAsync(container.Engine, w, r, cb)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
	}
}

// Proxy a request to the right node
func proxyContainer(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.Id, 1)
	}

	err = proxy(container.Engine, w, r)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a request to the right node and force refresh container
func proxyContainerAndForceRefresh(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
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

	err = proxyAsync(container.Engine, w, r, cb)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a request to the right node
func proxyImage(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if image := c.cluster.Image(name); image != nil {
		err := proxy(image.Engine, w, r)
		image.Engine.CheckConnectionErr(err)
		return
	}
	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// Proxy get image request to the right node
func proxyImageGet(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	for _, image := range c.cluster.Images() {
		if len(strings.SplitN(name, ":", 2)) == 2 && image.Match(name, true) ||
			len(strings.SplitN(name, ":", 2)) == 1 && image.Match(name, false) {
			err := proxy(image.Engine, w, r)
			image.Engine.CheckConnectionErr(err)
			return
		}
	}
	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// Proxy push image request to the right node
func proxyImagePush(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tag := r.Form.Get("tag")
	if tag != "" {
		name = name + ":" + tag
	}

	for _, image := range c.cluster.Images() {
		if tag != "" && image.Match(name, true) ||
			tag == "" && image.Match(name, false) {
			err := proxy(image.Engine, w, r)
			image.Engine.CheckConnectionErr(err)
			return
		}
	}

	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// POST /images/{name:.*}/tag
func postTagImage(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	repo := r.Form.Get("repo")
	tag := r.Form.Get("tag")
	force := boolValue(r, "force")

	// call cluster tag image
	if err := c.cluster.TagImage(name, repo, tag, force); err != nil {
		if strings.HasPrefix(err.Error(), "No such image") {
			httpError(w, err.Error(), http.StatusNotFound)
		} else {
			httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
	w.WriteHeader(http.StatusCreated)
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

	err = proxy(engine, w, r)
	engine.CheckConnectionErr(err)
	if err != nil {
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
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
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
	err = proxyAsync(container.Engine, w, r, cb)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /build
func postBuild(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buildImage := &apitypes.ImageBuildOptions{
		Dockerfile:     r.Form.Get("dockerfile"),
		Tags:           r.Form["t"],
		RemoteContext:  r.Form.Get("remote"),
		NoCache:        boolValue(r, "nocache"),
		PullParent:     boolValue(r, "pull"),
		Remove:         boolValue(r, "rm"),
		ForceRemove:    boolValue(r, "forcerm"),
		SuppressOutput: boolValue(r, "q"),
		Isolation:      containertypes.Isolation(r.Form.Get("isolation")),
		Memory:         int64ValueOrZero(r, "memory"),
		MemorySwap:     int64ValueOrZero(r, "memswap"),
		CPUShares:      int64ValueOrZero(r, "cpushares"),
		CPUPeriod:      int64ValueOrZero(r, "cpuperiod"),
		CPUQuota:       int64ValueOrZero(r, "cpuquota"),
		CPUSetCPUs:     r.Form.Get("cpusetcpus"),
		CPUSetMems:     r.Form.Get("cpusetmems"),
		CgroupParent:   r.Form.Get("cgroupparent"),
		ShmSize:        int64ValueOrZero(r, "shmsize"),
		Context:        r.Body,
	}

	buildArgsJSON := r.Form.Get("buildargs")
	if buildArgsJSON != "" {
		json.Unmarshal([]byte(buildArgsJSON), &buildImage.BuildArgs)
	}

	ulimitsJSON := r.Form.Get("ulimits")
	if ulimitsJSON != "" {
		json.Unmarshal([]byte(ulimitsJSON), &buildImage.Ulimits)
	}

	labelsJSON := r.Form.Get("labels")
	if labelsJSON != "" {
		json.Unmarshal([]byte(labelsJSON), &buildImage.Labels)
	}

	authEncoded := r.Header.Get("X-Registry-Config")
	if authEncoded != "" {
		buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Config"))
		if err == nil {
			json.Unmarshal(buf, &buildImage.AuthConfigs)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	wf := NewWriteFlusher(w)

	err := c.cluster.BuildImage(buildImage, wf)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /containers/{name:.*}/rename
func postRenameContainer(c *context, w http.ResponseWriter, r *http.Request) {
	_, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
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
	w.WriteHeader(http.StatusNoContent)

}

// Proxy a hijack request to the right node
func proxyHijack(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.Id, 1)
	}

	err = hijack(c.tlsConfig, container.Engine.Addr, w, r)
	container.Engine.CheckConnectionErr(err)
	if err != nil {
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
