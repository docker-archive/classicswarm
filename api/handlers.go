package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	apitypes "github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	typesversions "github.com/docker/docker/api/types/versions"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/parsers/kernel"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/experimental"
	"github.com/docker/swarm/version"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	// APIVERSION is the default API version supported by swarm manager
	APIVERSION = "1.39"
	// PLATFORM defines the value of Version.Platform.Name returned by swarm.
	// Basically, swarm is only shipped as a community product. Further, it's
	// assumed that if anything is programmatically looking at this field,
	// they're more than likely either string matching the full value (which we
	// ought not return here -- we are not exactly the same platform as the
	// engines we're managing) or they're string-matching on "Community".
	PLATFORM = "Docker Swarm - Community"
)

var (
	ShouldRefreshOnNodeFilter  = false
	ContainerNameRefreshFilter = ""
)

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
		NCPU:              int(c.cluster.TotalCpus()),
		MemTotal:          c.cluster.TotalMemory(),
		HTTPProxy:         os.Getenv("http_proxy"),
		HTTPSProxy:        os.Getenv("https_proxy"),
		NoProxy:           os.Getenv("no_proxy"),
		SystemTime:        time.Now().Format(time.RFC3339Nano),
		ExperimentalBuild: experimental.ENABLED,
		SystemStatus:      c.statusHandler.Status(),
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

	version.Platform.Name = PLATFORM

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
		if !engine.IsHealthy() {
			continue
		}
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

	httpError(w, fmt.Sprintf("Unable to find a healthy engine containing all images: %s", names), http.StatusNotFound)
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

	// Deprecated "filter" Filter. This is required for backward compatibility.
	filterParam := r.Form.Get("filter")
	if typesversions.LessThan(c.apiVersion, "1.28") && filterParam != "" {
		filters.Add("reference", filterParam)
	}

	// TODO: apply node filter in engine?
	accepteds := filters.Get("node")
	// this struct helps grouping images
	// but still keeps their Engine infos as an array.
	groupImages := make(map[string]apitypes.ImageSummary)
	opts := cluster.ImageFilterOptions{
		ImageListOptions: apitypes.ImageListOptions{
			All:     boolValue(r, "all"),
			Filters: filters,
		},
	}
	if opts.Filters.Include("dangling") &&
		!(opts.Filters.ExactMatch("dangling", "false") || opts.Filters.ExactMatch("dangling", "true")) {
		httpError(w, "Invalid filter: 'type'='dangling'", http.StatusBadRequest)
		return
	}

	imagesToFilter := c.cluster.Images()

	// This piece of code validates that the before and since fields for the filtering
	for _, filterType := range [2]string{"before", "since"} {
		if opts.Filters.Include(filterType) && imagesToFilter.GetImageFromField(filterType, opts) == nil {
			httpError(
				w,
				fmt.Sprintf("Invalid filter: 'type'='%s': no such image found: '%s'", filterType, opts.GetIDOrName(filterType)),
				http.StatusBadRequest,
			)
			return
		}
	}

	for _, image := range imagesToFilter.Filter(opts) {
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

		// grouping images by Id, and concatenate their RepoTags
		if entry, existed := groupImages[image.ID]; existed {
			entry.RepoTags = append(entry.RepoTags, image.RepoTags...)
			entry.RepoDigests = append(entry.RepoDigests, image.RepoDigests...)
			groupImages[image.ID] = entry
		} else {
			groupImages[image.ID] = image.ImageSummary
		}
	}

	images := []apitypes.ImageSummary{}

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
	networks := c.cluster.Networks().Filter(filters)
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
		json.NewEncoder(w).Encode(cleanNetwork.NetworkResource)
		return
	}
	httpError(w, fmt.Sprintf("No such network: %s", id), http.StatusNotFound)
}

// GET /volumes/{volumename:.*}
func getVolume(c *context, w http.ResponseWriter, r *http.Request) {
	var name = mux.Vars(r)["volumename"]
	if volume := c.cluster.Volumes().Get(name); volume != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volume.Volume)
		return
	}
	httpError(w, fmt.Sprintf("No such volume: %s", name), http.StatusNotFound)
}

// GET /volumes
func getVolumes(c *context, w http.ResponseWriter, r *http.Request) {
	volumesListResponse := volumetypes.VolumeListOKBody{}

	// Parse filters
	filters, err := dockerfilters.FromParam(r.URL.Query().Get("filters"))
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	names := filters.Get("name")
	nodes := filters.Get("node")
	for _, volume := range c.cluster.Volumes() {
		// Check if the volume matches any name filters
		found := false
		for _, name := range names {
			if strings.Contains(volume.Name, name) {
				found = true
				break
			}
		}

		if len(names) > 0 && !found {
			// Do not include this volume in the response if it doesn't match
			// a name filter, if any exist.
			continue
		}

		if filters.Include("label") {
			if !filters.MatchKVList("label", volume.Labels) {
				continue
			}
		}

		tmp := (*volume).Volume
		if tmp.Driver == "local" {
			// Check if the volume matches any node filters
			found = false
			for _, node := range nodes {
				if volume.Engine.Name == node {
					found = true
					break
				}
			}
			if len(nodes) > 0 && !found {
				continue
			}
			tmp.Name = volume.Engine.Name + "/" + volume.Name
		}
		volumesListResponse.Volumes = append(volumesListResponse.Volumes, &tmp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(volumesListResponse)
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

	if ShouldRefreshOnNodeFilter {
		nodes := filters.Get("node")
		for _, node := range nodes {
			err := c.cluster.RefreshEngine(node)
			if err != nil {
				log.Debugf("could not match node filter for %s: %s", node, err)
			}
		}
	}
	if ContainerNameRefreshFilter != "" {
		names := filters.Get("name")
		for _, name := range names {
			if name == ContainerNameRefreshFilter {
				err := c.cluster.RefreshEngines()
				if err != nil {
					log.Debugf("names filter detected but unable to refresh all engines: %s", err)
				}
				break
			}
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
		if !filters.Match("id", container.ID) {
			continue
		}
		if !filters.MatchKVList("label", container.Config.Labels) {
			continue
		}
		if !filters.Match("status", cluster.StateString(container.Info.State)) {
			continue
		}
		if !filters.Match("node", container.Engine.Name) {
			continue
		}
		if !filters.ExactMatch("health", cluster.HealthString(container.Info.State)) {
			continue
		}
		if filters.Include("is-task") {
			_, isTask := container.Config.Labels["com.docker.swarm.task"]
			if filters.ExactMatch("is-task", "true") && !isTask {
				continue
			}
			if filters.ExactMatch("is-task", "false") && isTask {
				continue
			}
		}
		if filters.Include("volume") {
			volumeExist := fmt.Errorf("volume mounted in container")
			err := filters.WalkValues("volume", func(value string) error {
				for _, mount := range container.Info.Mounts {
					if mount.Name == value || mount.Destination == value {
						return volumeExist
					}
				}
				return nil
			})
			if err != volumeExist {
				continue
			}
		}
		if filters.Include("network") {
			networkExist := fmt.Errorf("network attached to container")
			err := filters.WalkValues("network", func(value string) error {
				if _, ok := container.NetworkSettings.Networks[value]; ok {
					return networkExist
				}
				for _, nw := range container.NetworkSettings.Networks {
					if nw == nil {
						continue
					}
					if strings.HasPrefix(nw.NetworkID, value) {
						return networkExist
					}
				}
				return nil
			})
			if err != networkExist {
				continue
			}
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

	// Convert cluster.Container back into apitypes.Container.
	out := []*apitypes.Container{}
	for _, container := range candidates {
		if before != nil {
			if container.ID == before.ID {
				before = nil
			}
			continue
		}
		// Create a copy of the underlying apitypes.Container so we can
		// make changes without messing with cluster.Container.
		tmp := (*container).Container

		// Update the Status. The one we have is stale from the last `docker ps` the engine sent.
		// `Status()` will generate a new one
		tmp.Status = cluster.FullStateString(container.Info.State)
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
		tmp.Ports = make([]apitypes.Port, len(container.Ports))
		for i, port := range container.Ports {
			tmp.Ports[i] = port
			if ip := net.ParseIP(port.IP); ip != nil && ip.IsUnspecified() {
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

	con, err := container.Engine.InspectContainer(container.ID)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// insert Node field
	con.Node = container.Engine.EngineToContainerNode()

	// update zero ip to engine ip, including IPv4 and IPv6
	if con.NetworkSettings != nil {
		for _, portBindings := range con.NetworkSettings.Ports {
			for key, portBinding := range portBindings {
				if ip := net.ParseIP(portBinding.HostIP); ip == nil || ip.IsUnspecified() {
					portBindings[key].HostIP = container.Engine.IP
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(con)
}

// POST /containers/create
func postContainersCreate(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var (
		defaultMemorySwappiness = int64(-1)
		name                    = r.Form.Get("name")
		config                  = cluster.ContainerConfig{
			HostConfig: containertypes.HostConfig{
				Resources: containertypes.Resources{
					MemorySwappiness: &(defaultMemorySwappiness),
				},
			},
		}
	)

	oldconfig := cluster.OldContainerConfig{
		ContainerConfig: config,
		Memory:          0,
		MemorySwap:      0,
		CPUShares:       0,
		CPUSet:          "",
	}

	if err := json.NewDecoder(r.Body).Decode(&oldconfig); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// make sure HostConfig fields are consolidated before creating container
	cluster.ConsolidateResourceFields(&oldconfig)
	config = oldconfig.ContainerConfig

	// Pass auth information along if present
	var authConfig *apitypes.AuthConfig
	buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
	if err == nil {
		authConfig = &apitypes.AuthConfig{}
		json.Unmarshal(buf, authConfig)
	}
	containerConfig := cluster.BuildContainerConfig(config.Config, config.HostConfig, config.NetworkingConfig)
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
	fmt.Fprintf(w, "{%q:%q}", "Id", container.ID)
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
	var request apitypes.NetworkCreateRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Driver == "" {
		request.Driver = "overlay"
	}

	response, err := c.cluster.CreateNetwork(request.Name, &request.NetworkCreate)
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
	var request volumetypes.VolumeCreateBody

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
		authConfig := apitypes.AuthConfig{}
		buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Auth"))
		if err == nil {
			json.Unmarshal(buf, &authConfig)
		}
		tag := r.Form.Get("tag")
		image := getImageRef(image, tag)

		var errorMessage string
		errorFound := false
		nonOSErrorFound := false
		successfulPull := false
		callback := func(msg cluster.JSONMessageWrapper) {
			msg.Msg.ID = msg.EngineName
			if msg.Msg.Status != "" {
				msg.Msg.Status = fmt.Sprintf("Pulling %s... : %s", image, msg.Msg.Status)
			} else {
				msg.Msg.Status = fmt.Sprintf("Pulling %s...", image)
			}
			// If we get a successful pull on any node, mark
			// successfulPull as true.
			if msg.Success {
				successfulPull = true
			}
			if msg.Err != nil {
				errorFound = true
				errorMessage = msg.Err.Error()
				if !strings.Contains(errorMessage, "image operating system") {
					nonOSErrorFound = true
				}
				msg.Msg.Status = fmt.Sprintf("Pulling %s... : %s", image, errorMessage)
			}
			json.NewEncoder(wf).Encode(msg.Msg)
		}
		c.cluster.Pull(image, &authConfig, callback)

		if errorFound {
			// If some nodes successfully pulled the image and the
			// rest failed because the image was the wrong OS
			// (e.g. we tried to pull a Linux-based image on a
			// Windows node), we should still consider the pull
			// successful because we loaded the image on as many
			// nodes as we could.
			if !successfulPull || nonOSErrorFound {
				sendErrorJSONMessage(wf, 1, errorMessage)
			}
		}

	} else { //import
		source := r.Form.Get("fromSrc")
		repo := r.Form.Get("repo")
		tag := r.Form.Get("tag")

		var errorMessage string
		errorFound := false
		callback := func(msg cluster.JSONMessageWrapper) {
			msg.Msg.ID = msg.EngineName
			if msg.Err != nil {
				errorFound = true
				errorMessage = msg.Err.Error()
				msg.Msg.Status = errorMessage
			}
			json.NewEncoder(wf).Encode(msg.Msg)

		}
		ref := getImageRef(repo, tag)
		c.cluster.Import(source, ref, tag, r.Body, callback)
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
	callback := func(msg cluster.JSONMessageWrapper) {
		msg.Msg.ID = msg.EngineName
		if msg.Msg.Status != "" {
			msg.Msg.Status = fmt.Sprintf("Loading Image... : %s", msg.Msg.Status)
		} else {
			msg.Msg.Status = "Loading Image..."
		}

		if msg.Err != nil {
			errorFound = true
			errorMessage = msg.Err.Error()
			msg.Msg.Status = fmt.Sprintf("Loading Image... : %s", errorMessage)
		}
		json.NewEncoder(wf).Encode(msg.Msg)
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

	w.Header().Set("Content-Type", "application/json")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	eventsChan, cancelFunc := c.eventsHandler.Watch()
	defer cancelFunc()

	// create timer for --until
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
	)
	if until > 0 {
		dur := time.Unix(until, 0).Sub(time.Now())
		timer = time.NewTimer(dur)
		timerCh = timer.C
	}
	var closeNotify <-chan bool
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		closeNotify = closeNotifier.CloseNotify()
	}

	for {
		select {
		case eChan, ok := <-eventsChan:
			if !ok {
				return
			}
			e, ok := eChan.(*cluster.Event)
			if !ok {
				break
			}
			data, err := normalizeEvent(e)
			if err != nil {
				return
			}
			_, err = w.Write(data)
			if err != nil {
				log.Debugf("failed to write event to output stream %s", err.Error())
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-closeNotify:
			return
		case <-timerCh:
			return
		}
	}
}

// POST /containers/{name:.*}/start
func postContainersStart(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container %s", name), http.StatusNotFound)
		return
	}

	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if err := c.cluster.StartContainer(container); err != nil {
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

	execConfig := apitypes.ExecConfig{}
	if err := json.NewDecoder(r.Body).Decode(&execConfig); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(execConfig.Cmd) == 0 {
		httpError(w, fmt.Sprintf("No exec command specified"), http.StatusBadRequest)
		return
	}

	execCreateResp, err := container.Engine.CreateContainerExec(container.ID, execConfig)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add execID to the container, so the later exec/start will work
	container.Info.ExecIDs = append(container.Info.ExecIDs, execCreateResp.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(execCreateResp)
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
func networkDisconnect(c *context, w http.ResponseWriter, r *http.Request) {
	var networkid = mux.Vars(r)["networkid"]
	network := c.cluster.Networks().Uniq().Get(networkid)
	if network == nil {
		httpError(w, fmt.Sprintf("No such network: %s", networkid), http.StatusNotFound)
		return
	}
	// If the incoming request used the network's name instead of the ID,
	// make the request to the daemon with the network's name as well.
	if strings.Contains(networkid, network.Name) {
		networkid = network.Name
	} else {
		networkid = network.ID
	}

	// make a copy of r.Body
	buf, _ := ioutil.ReadAll(r.Body)
	bodyCopy := ioutil.NopCloser(bytes.NewBuffer(buf))
	defer bodyCopy.Close()
	// restore r.Body stream as it'll be read again
	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	// Extract container info from r.Body copy
	var disconnect apitypes.NetworkDisconnect
	if err := json.NewDecoder(bodyCopy).Decode(&disconnect); err != nil {
		httpError(w, "Container is not specified", http.StatusNotFound)
		return
	}

	container := c.cluster.Container(disconnect.Container)
	if container == nil {
		httpError(w, fmt.Sprintf("No such container: %s", disconnect.Container), http.StatusNotFound)
		return
	}
	engine := container.Engine

	// First try to disconnect the container on its associated engine, and
	// then try a random engine if we can't connect to that engine. We
	// try the associated engine first because on 1.12+ clusters, the
	// network may not be known on all nodes.
	err := engine.NetworkDisconnect(container, networkid, disconnect.Force)
	if err != nil {
		if cluster.IsConnectionError(err) && disconnect.Force && network.Scope == "global" {
			log.Warnf("Could not connect to engine %s: %s, trying to disconnect %s from %s on a random engine", engine.Name, err, disconnect.Container, network.Name)
			randomEngine, randomEngineErr := c.cluster.RANDOMENGINE()
			if randomEngineErr != nil {
				log.Warnf("Could not get a random engine: %s", randomEngineErr)
				httpError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = randomEngine.NetworkDisconnect(container, networkid, disconnect.Force)
			if err != nil {
				httpError(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		httpError(w, "Container is not specified", http.StatusNotFound)
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
		// force refresh this container so that it is up to date in the cache
		container.Engine.UpdateNetworkContainers(container.ID, true)
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
			return
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.ID, 1)
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
			return
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.ID, 1)
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

	ref := getImageRef(repo, tag)
	// call cluster tag image
	if err := c.cluster.TagImage(name, ref, force); err != nil {
		if strings.HasPrefix(err.Error(), "No such image") {
			httpError(w, err.Error(), http.StatusNotFound)
		} else {
			httpError(w, err.Error(), http.StatusInternalServerError)
		}
		return
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
			return
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.RawQuery = strings.Replace(r.URL.RawQuery, name, container.ID, 1)
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
		NetworkMode:    r.Form.Get("networkmode"),
		CPUShares:      int64ValueOrZero(r, "cpushares"),
		CPUPeriod:      int64ValueOrZero(r, "cpuperiod"),
		CPUQuota:       int64ValueOrZero(r, "cpuquota"),
		CPUSetCPUs:     r.Form.Get("cpusetcpus"),
		CPUSetMems:     r.Form.Get("cpusetmems"),
		CgroupParent:   r.Form.Get("cgroupparent"),
		ShmSize:        int64ValueOrZero(r, "shmsize"),
		Squash:         boolValue(r, "squash"),
		SessionID:      r.Form.Get("session"),
		BuildID:        r.Form.Get("buildid"),
		Target:         r.Form.Get("target"),
		Platform:       r.Form.Get("platform"),
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

	cacheFromJSON := r.Form.Get("cachefrom")
	if cacheFromJSON != "" {
		json.Unmarshal([]byte(cacheFromJSON), &buildImage.CacheFrom)
	}

	authEncoded := r.Header.Get("X-Registry-Config")
	if authEncoded != "" {
		buf, err := base64.URLEncoding.DecodeString(r.Header.Get("X-Registry-Config"))
		if err == nil {
			json.Unmarshal(buf, &buildImage.AuthConfigs)
		}
	}

	builderVersion, err := parseVersion(r.Form.Get("version"))
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buildImage.Version = builderVersion

	outputsJSON := r.FormValue("outputs")
	if outputsJSON != "" {
		var outputs []apitypes.ImageBuildOutput
		if err := json.Unmarshal([]byte(outputsJSON), &outputs); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		buildImage.Outputs = outputs
	}

	w.Header().Set("Content-Type", "application/json")
	wf := NewWriteFlusher(w)

	var errorMessage string
	errorFound := false
	callback := func(msg cluster.JSONMessageWrapper) {
		if builderVersion != apitypes.BuilderBuildKit {
			msg.Msg.ID = msg.EngineName
		}
		if msg.Err != nil {
			errorFound = true
			errorMessage = msg.Err.Error()
			osType := MatchImageOSError(errorMessage)
			if osType != "" {
				msg.Msg.Status = fmt.Sprintf("Could not build image: %s. Consider using --build-arg 'constraint:ostype==%s'", errorMessage, osType)
			}
		}
		json.NewEncoder(wf).Encode(msg.Msg)
	}
	err = c.cluster.BuildImage(r.Body, buildImage, callback)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if errorFound {
		sendErrorJSONMessage(wf, 1, errorMessage)
	}
}

func parseVersion(s string) (apitypes.BuilderVersion, error) {
	if s == "" || s == string(apitypes.BuilderV1) {
		return apitypes.BuilderV1, nil
	}
	if s == string(apitypes.BuilderBuildKit) {
		return apitypes.BuilderBuildKit, nil
	}
	return "", fmt.Errorf("invalid version %s", s)
}

// POST /build/cancel
func postBuildCancel(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buildID := r.FormValue("id")
	if buildID == "" {
		httpError(w, "build ID not provided", http.StatusBadRequest)
		return
	}

	if err := c.cluster.BuildCancel(buildID); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST /containers/{name:.*}/rename
func postRenameContainer(c *context, w http.ResponseWriter, r *http.Request) {
	_, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
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
		return
	}
	w.WriteHeader(http.StatusNoContent)

}

// POST /session
func postSession(c *context, w http.ResponseWriter, r *http.Request) {
	// calls to /session have the SessionID in the
	// "X-Docker-Expose-Session-Uuid" header, so get that
	sessionID := r.Header.Get("X-Docker-Expose-Session-Uuid")
	// the sessionID cannot be empty -- if it is, return an error to the user.
	// we could return such an error from the c.cluster.Session call, but then
	// we would have to parse that error back out in order to correctly return
	// a 400 response, so it's easier to just do it here
	if sessionID == "" {
		httpError(w, "no session ID provided", http.StatusBadRequest)
		return
	}

	engine, err := c.cluster.Session(sessionID)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// now, hijack the connection and forward to this engine.
	err = hijack(c.tlsConfig, engine.Addr, w, r)
	engine.CheckConnectionErr(err)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a hijack request to the right node
func proxyHijack(c *context, w http.ResponseWriter, r *http.Request) {
	name, container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		if container == nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set the full container ID in the proxied URL path.
	if name != "" {
		r.URL.Path = strings.Replace(r.URL.Path, name, container.ID, 1)
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

// headerFlusher is a convenient wrapper around http.ResponseWriter which
// always flushes response headers to the client immediately.
type headerFlusher struct {
	http.ResponseWriter
}

func (h headerFlusher) WriteHeader(status int) {
	h.ResponseWriter.WriteHeader(status)

	// Try to flush the header immediately.
	if flusher, ok := h.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// POST /containers/{name:.*}/wait
// This endpoint is special because it is important to flush the response
// header immediately.
func postContainersWait(c *context, w http.ResponseWriter, r *http.Request) {
	proxyContainerAndForceRefresh(c, headerFlusher{w}, r)
}
