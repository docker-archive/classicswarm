package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcluster"
	"github.com/docker/libcluster/scheduler"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

type context struct {
	cluster       *libcluster.Cluster
	scheduler     *scheduler.Scheduler
	eventsHandler *eventsHandler
	debug         bool
}

type handler func(c *context, w http.ResponseWriter, r *http.Request)

// GET /info
func getInfo(c *context, w http.ResponseWriter, r *http.Request) {
	driverStatus := [][2]string{{"\bNodes", fmt.Sprintf("%d", len(c.cluster.Nodes()))}}

	for ID, node := range c.cluster.Nodes() {
		driverStatus = append(driverStatus, [2]string{ID, node.Addr})
	}
	info := struct {
		Containers      int
		DriverStatus    [][2]string
		NEventsListener int
		Debug           bool
	}{
		len(c.cluster.Containers()),
		driverStatus,
		c.eventsHandler.Size(),
		c.debug,
	}

	json.NewEncoder(w).Encode(info)
}

// GET /containers/ps
// GET /containers/json
func getContainersJSON(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all := r.Form.Get("all") == "1"

	out := []*dockerclient.Container{}
	for _, container := range c.cluster.Containers() {
		tmp := (*container).Container
		// Skip stopped containers unless -a was specified.
		if !strings.Contains(tmp.Status, "Up") && !all {
			continue
		}
		// TODO remove the Node ID in the name when we have a good solution
		tmp.Names = make([]string, len(container.Names))
		for i, name := range container.Names {
			tmp.Names[i] = "/" + container.Node().ID + name
		}
		tmp.Ports = make([]dockerclient.Port, len(container.Ports))
		for i, port := range container.Ports {
			tmp.Ports[i] = port
			if port.IP == "0.0.0.0" {
				tmp.Ports[i].IP = container.Node().IP
			}
		}
		out = append(out, &tmp)
	}

	sort.Sort(sort.Reverse(ContainerSorter(out)))
	json.NewEncoder(w).Encode(out)
}

// GET /containers/{name:.*}/json
func getContainerJSON(c *context, w http.ResponseWriter, r *http.Request) {
	container := c.cluster.Container(mux.Vars(r)["name"])
	if container != nil {
		resp, err := http.Get(container.Node().Addr + "/containers/" + container.Id + "/json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bytes.Replace(data, []byte("\"HostIp\":\"0.0.0.0\""), []byte(fmt.Sprintf("\"HostIp\":%q", container.Node().IP)), -1))
	}
}

// POST /containers/create
func postContainersCreate(c *context, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var config dockerclient.ContainerConfig

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	container, err := c.scheduler.CreateContainer(&config, r.Form.Get("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
	return
}

// POST /containers/{name:.*}/start
func postContainersStart(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := container.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
}

// POST /containers/{name:.*}/pause
func postContainerPause(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := container.Pause(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
}

// POST /containers/{name:.*}/unpause
func postContainerUnpause(c *context, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.cluster.Container(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := container.Unpause(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
}

// DELETE /containers/{name:.*}
func deleteContainer(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := mux.Vars(r)["name"]
	force := r.Form.Get("force") == "1"
	container := c.cluster.Container(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := c.scheduler.RemoveContainer(container, force); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// GET /events
func getEvents(c *context, w http.ResponseWriter, r *http.Request) {
	c.eventsHandler.Lock()
	c.eventsHandler.ws[r.RemoteAddr] = w
	c.eventsHandler.cs[r.RemoteAddr] = make(chan struct{})
	c.eventsHandler.Unlock()
	w.Header().Set("Content-Type", "application/json")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	<-c.eventsHandler.cs[r.RemoteAddr]
}

// GET /_ping
func ping(c *context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// Redirect a GET request to the right node
func redirectContainer(c *context, w http.ResponseWriter, r *http.Request) {
	container := c.cluster.Container(mux.Vars(r)["name"])
	if container != nil {
		re := regexp.MustCompile("/v([0-9.]*)") // TODO: discuss about skipping the version or not

		newURL, _ := url.Parse(container.Node().Addr)
		newURL.RawQuery = r.URL.RawQuery
		newURL.Path = re.ReplaceAllLiteralString(r.URL.Path, "")
		log.Debugf("REDIRECT TO %s", newURL.String())
		http.Redirect(w, r, newURL.String(), http.StatusSeeOther)
	}
}

// Default handler for methods not supported by clustering.
func notImplementedHandler(c *context, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not supported in clustering mode.", http.StatusNotImplemented)
}

func optionsHandler(c *context, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func writeCorsHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
}

func createRouter(c *context, enableCors bool) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]handler{
		"GET": {
			"/_ping":                          ping,
			"/events":                         getEvents,
			"/info":                           getInfo,
			"/version":                        notImplementedHandler,
			"/images/json":                    notImplementedHandler,
			"/images/viz":                     notImplementedHandler,
			"/images/search":                  notImplementedHandler,
			"/images/get":                     notImplementedHandler,
			"/images/{name:.*}/get":           notImplementedHandler,
			"/images/{name:.*}/history":       notImplementedHandler,
			"/images/{name:.*}/json":          notImplementedHandler,
			"/containers/ps":                  getContainersJSON,
			"/containers/json":                getContainersJSON,
			"/containers/{name:.*}/export":    redirectContainer,
			"/containers/{name:.*}/changes":   redirectContainer,
			"/containers/{name:.*}/json":      getContainerJSON,
			"/containers/{name:.*}/top":       redirectContainer,
			"/containers/{name:.*}/logs":      redirectContainer,
			"/containers/{name:.*}/attach/ws": notImplementedHandler,
		},
		"POST": {
			"/auth":                         notImplementedHandler,
			"/commit":                       notImplementedHandler,
			"/build":                        notImplementedHandler,
			"/images/create":                notImplementedHandler,
			"/images/load":                  notImplementedHandler,
			"/images/{name:.*}/push":        notImplementedHandler,
			"/images/{name:.*}/tag":         notImplementedHandler,
			"/containers/create":            postContainersCreate,
			"/containers/{name:.*}/kill":    notImplementedHandler,
			"/containers/{name:.*}/pause":   postContainerPause,
			"/containers/{name:.*}/unpause": postContainerUnpause,
			"/containers/{name:.*}/restart": notImplementedHandler,
			"/containers/{name:.*}/start":   postContainersStart,
			"/containers/{name:.*}/stop":    notImplementedHandler,
			"/containers/{name:.*}/wait":    notImplementedHandler,
			"/containers/{name:.*}/resize":  notImplementedHandler,
			"/containers/{name:.*}/attach":  notImplementedHandler,
			"/containers/{name:.*}/copy":    notImplementedHandler,
			"/containers/{name:.*}/exec":    notImplementedHandler,
			"/exec/{name:.*}/start":         notImplementedHandler,
			"/exec/{name:.*}/resize":        notImplementedHandler,
		},
		"DELETE": {
			"/containers/{name:.*}": deleteContainer,
			"/images/{name:.*}":     notImplementedHandler,
		},
		"OPTIONS": {
			"": optionsHandler,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			log.Debugf("Registering %s, %s", method, route)

			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				log.Infof("%s %s", r.Method, r.RequestURI)
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

	return r, nil
}

func ListenAndServe(c *libcluster.Cluster, s *scheduler.Scheduler, addr string) error {
	context := &context{
		cluster:   c,
		scheduler: s,
		eventsHandler: &eventsHandler{
			ws: make(map[string]io.Writer),
			cs: make(map[string]chan struct{}),
		},
	}
	c.Events(context.eventsHandler)
	r, err := createRouter(context, false)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return server.ListenAndServe()
}
