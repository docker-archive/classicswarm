package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	dockerfilters "github.com/docker/docker/pkg/parsers/filters"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/version"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

const APIVERSION = "1.16"

type context struct {
	cluster       cluster.Cluster
	eventsHandler *eventsHandler
	debug         bool
	tlsConfig     *tls.Config
}

type handler func(c *context, w http.ResponseWriter, r *http.Request)

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
		ApiVersion string
		GoVersion  string
		GitCommit  string
		Os         string
		Arch       string
	}{
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
				if accepted == image.Node.Name() || accepted == image.Node.ID() {
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
		if !container.Node.IsHealthy() {
			tmp.Status = "Pending"
		}
		// TODO remove the Node Name in the name when we have a good solution
		tmp.Names = make([]string, len(container.Names))
		for i, name := range container.Names {
			tmp.Names[i] = "/" + container.Node.Name() + name
		}
		// insert node IP
		tmp.Ports = make([]dockerclient.Port, len(container.Ports))
		for i, port := range container.Ports {
			tmp.Ports[i] = port
			if port.IP == "0.0.0.0" {
				tmp.Ports[i].IP = container.Node.IP()
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

	resp, err := client.Get(scheme + "://" + container.Node.Addr() + "/containers/" + container.Id + "/json")
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

	// insert Node field
	data = bytes.Replace(data, []byte("\"Name\":\"/"), []byte(fmt.Sprintf("\"Node\":%s,\"Name\":\"/", cluster.SerializeNode(container.Node))), -1)

	// insert node IP
	data = bytes.Replace(data, []byte("\"HostIp\":\"0.0.0.0\""), []byte(fmt.Sprintf("\"HostIp\":%q", container.Node.IP())), -1)

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

	resp, err := client.Post(scheme+"://"+container.Node.Addr()+"/containers/"+container.Id+"/exec", "application/json", r.Body)
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

	id := struct{ Id string }{}

	if err := json.Unmarshal(data, &id); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add execID to the container, so the later exec/start will work
	container.Info.ExecIDs = append(container.Info.ExecIDs, id.Id)

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

	client, scheme := newClientAndScheme(c.tlsConfig)
	defer closeIdleConnections(client)

	matchedImages := []*cluster.Image{}
	for _, image := range c.cluster.Images() {
		if image.Match(name) {
			fmt.Println("matched", image)
			matchedImages = append(matchedImages, image)
		}
	}

	size := len(matchedImages)
	if size == 0 {
		httpError(w, fmt.Sprintf("No such image %s", name), http.StatusNotFound)
		return
	}
	wf := NewWriteFlusher(w)

	for i, image := range matchedImages {
		fmt.Println("delete", image)
		req, err := http.NewRequest("DELETE", scheme+"://"+image.Node.Addr()+"/images/"+name, nil)
		if err != nil {
			if size == 1 {
				httpError(w, err.Error(), http.StatusInternalServerError)
			}
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			if size == 1 {
				httpError(w, err.Error(), http.StatusInternalServerError)
			}
			continue
		}
		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if size == 1 {
				httpError(w, err.Error(), http.StatusInternalServerError)
			}
			continue
		}
		if resp.StatusCode != 200 {
			if size == 1 {
				w.WriteHeader(resp.StatusCode)
				io.Copy(NewWriteFlusher(w), resp.Body)
			}
			continue
		}

		sdata := bytes.NewBuffer(data).String()
		if i != 0 {
			w.Header().Set("Content-Type", "application/json")
			sdata = strings.Replace(sdata, "[", ",", -1)
		}

		if i != size-1 {
			sdata = strings.Replace(sdata, "]", "", -1)
		}
		fmt.Fprintf(wf, sdata)
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

	if err := proxy(c.tlsConfig, container.Node.Addr(), w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}
}

// Proxy a request to the right node
func proxyImage(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if image := c.cluster.Image(name); image != nil {
		proxy(c.tlsConfig, image.Node.Addr(), w, r)
		return
	}
	httpError(w, fmt.Sprintf("No such image: %s", name), http.StatusNotFound)
}

// Proxy a request to a random node
func proxyRandom(c *context, w http.ResponseWriter, r *http.Request) {
	candidates := []cluster.Node{}

	// FIXME: doesn't work if there are no container in the cluster
	// remove proxyRandom and implemente the features locally
	for _, container := range c.cluster.Containers() {
		candidates = append(candidates, container.Node)
	}

	healthFilter := &filter.HealthFilter{}
	accepted, err := healthFilter.Filter(nil, candidates)

	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := proxy(c.tlsConfig, accepted[rand.Intn(len(accepted))].Addr(), w, r); err != nil {
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

	if err := hijack(c.tlsConfig, container.Node.Addr(), w, r); err != nil {
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

func writeCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
}

func httpError(w http.ResponseWriter, err string, status int) {
	log.WithField("status", status).Errorf("HTTP error: %v", err)
	http.Error(w, err, status)
}

func createRouter(c *context, enableCors bool) *mux.Router {
	r := mux.NewRouter()
	m := map[string]map[string]handler{
		"GET": {
			"/_ping":                          ping,
			"/events":                         getEvents,
			"/info":                           getInfo,
			"/version":                        getVersion,
			"/images/json":                    getImagesJSON,
			"/images/viz":                     notImplementedHandler,
			"/images/search":                  proxyRandom,
			"/images/get":                     notImplementedHandler,
			"/images/{name:.*}/get":           proxyImage,
			"/images/{name:.*}/history":       proxyImage,
			"/images/{name:.*}/json":          proxyImage,
			"/containers/ps":                  getContainersJSON,
			"/containers/json":                getContainersJSON,
			"/containers/{name:.*}/export":    proxyContainer,
			"/containers/{name:.*}/changes":   proxyContainer,
			"/containers/{name:.*}/json":      getContainerJSON,
			"/containers/{name:.*}/top":       proxyContainer,
			"/containers/{name:.*}/logs":      proxyContainer,
			"/containers/{name:.*}/stats":     proxyContainer,
			"/containers/{name:.*}/attach/ws": notImplementedHandler,
			"/exec/{execid:.*}/json":          proxyContainer,
		},
		"POST": {
			"/auth":                         proxyRandom,
			"/commit":                       notImplementedHandler,
			"/build":                        notImplementedHandler,
			"/images/create":                postImagesCreate,
			"/images/load":                  notImplementedHandler,
			"/images/{name:.*}/push":        notImplementedHandler,
			"/images/{name:.*}/tag":         notImplementedHandler,
			"/containers/create":            postContainersCreate,
			"/containers/{name:.*}/kill":    proxyContainer,
			"/containers/{name:.*}/pause":   proxyContainer,
			"/containers/{name:.*}/unpause": proxyContainer,
			"/containers/{name:.*}/rename":  proxyContainer,
			"/containers/{name:.*}/restart": proxyContainer,
			"/containers/{name:.*}/start":   proxyContainer,
			"/containers/{name:.*}/stop":    proxyContainer,
			"/containers/{name:.*}/wait":    proxyContainer,
			"/containers/{name:.*}/resize":  proxyContainer,
			"/containers/{name:.*}/attach":  proxyHijack,
			"/containers/{name:.*}/copy":    proxyContainer,
			"/containers/{name:.*}/exec":    postContainersExec,
			"/exec/{execid:.*}/start":       proxyHijack,
			"/exec/{execid:.*}/resize":      proxyContainer,
		},
		"DELETE": {
			"/containers/{name:.*}": deleteContainers,
			"/images/{name:.*}":     deleteImages,
		},
		"OPTIONS": {
			"": optionsHandler,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			log.WithFields(log.Fields{"method": method, "route": route}).Debug("Registering HTTP route")

			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				log.WithFields(log.Fields{"method": r.Method, "uri": r.RequestURI}).Info("HTTP request received")
				if enableCors {
					writeCorsHeaders(w, r)
				}
				localFct(c, w, r)
			}
			localMethod := method

			// add the new route
			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r
}
