package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcluster"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

type HttpApiFunc func(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request)

// GET /info
func getInfo(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	var driverStatus [][2]string

	for ID, node := range c.Nodes() {
		driverStatus = append(driverStatus, [2]string{ID, node.Addr})
	}
	info := struct {
		Containers                             int
		Driver, ExecutionDriver                string
		DriverStatus                           [][2]string
		KernelVersion, OperatingSystem         string
		MemoryLimit, SwapLimit, IPv4Forwarding bool
	}{
		len(c.Containers()),
		"libcluster", "libcluster",
		driverStatus,
		"N/A", "N/A",
		true, true, true,
	}

	json.NewEncoder(w).Encode(info)
}

// GET /containers/ps
// GET /containers/json
func getContainersJSON(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all := r.Form.Get("all") == "1"

	out := []dockerclient.Container{}
	for _, container := range c.Containers() {
		// Skip stopped containers unless -a was specified.
		if !strings.Contains(container.Status, "Up") && !all {
			continue
		}
		out = append(out, container.Container)
	}

	sort.Sort(sort.Reverse(ContainerSorter(out)))
	json.NewEncoder(w).Encode(out)
}

// GET /containers/{name:.*}/json
func getContainerJSON(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	container := c.Container(mux.Vars(r)["name"])
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

// DELETE /containers/{name:.*}
func deleteContainer(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := mux.Vars(r)["name"]
	force := r.Form.Get("force") == "1"
	container := c.Container(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := container.Node().Remove(container, force); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// GET /_ping
func ping(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// Redirect a GET request to the right node
func redirectContainer(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	container := c.Container(mux.Vars(r)["name"])
	if container != nil {
		re := regexp.MustCompile("/v([0-9.]*)") // TODO: discuss about skipping the version or not

		newURL, _ := url.Parse(container.Node().Addr)
		newURL.RawQuery = r.URL.RawQuery
		newURL.Path = re.ReplaceAllLiteralString(r.URL.Path, "")
		fmt.Println("REDIR ->", newURL.String())
		http.Redirect(w, r, newURL.String(), http.StatusSeeOther)
	}
}

func notImplementedHandler(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not supported in clustering mode.", http.StatusNotImplemented)
}

func createRouter(c *libcluster.Cluster) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":                          ping,
			"/events":                         notImplementedHandler,
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
			"/containers/create":            notImplementedHandler,
			"/containers/{name:.*}/kill":    notImplementedHandler,
			"/containers/{name:.*}/pause":   notImplementedHandler,
			"/containers/{name:.*}/unpause": notImplementedHandler,
			"/containers/{name:.*}/restart": notImplementedHandler,
			"/containers/{name:.*}/start":   notImplementedHandler,
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
			"": notImplementedHandler,
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

func ListenAndServe(c *libcluster.Cluster, addr string) error {
	r, err := createRouter(c)
	if err != nil {
		return err
	}
	s := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s.ListenAndServe()
}
