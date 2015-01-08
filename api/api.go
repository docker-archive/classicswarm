package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

const APIVERSION = "1.16"

type context struct {
	cluster       *cluster.Cluster
	scheduler     *scheduler.Scheduler
	eventsHandler *eventsHandler
	debug         bool
	version       string
	tlsConfig     *tls.Config
}

type handler func(c *context, w http.ResponseWriter, r *http.Request)

// GET /info
func getInfo(c *context, w http.ResponseWriter, r *http.Request) {
	nodes := c.cluster.Nodes()
	driverStatus := [][2]string{{"\bNodes", fmt.Sprintf("%d", len(nodes))}}

	for _, node := range nodes {
		driverStatus = append(driverStatus, [2]string{node.Name, node.Addr})
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
		Version:    "swarm/" + c.version,
		ApiVersion: APIVERSION,
		GoVersion:  runtime.Version(),
		GitCommit:  "n/a",
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	json.NewEncoder(w).Encode(version)
}

// GET /containers/ps
// GET /containers/json
func getContainersJSON(c *context, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
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
		if !container.Node().IsHealthy() {
			tmp.Status = "Pending"
		}
		// TODO remove the Node Name in the name when we have a good solution
		tmp.Names = make([]string, len(container.Names))
		for i, name := range container.Names {
			tmp.Names[i] = "/" + container.Node().Name + name
		}
		// insert node IP
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
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		n, err := json.Marshal(container.Node())
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// insert Node field
		data = bytes.Replace(data, []byte("\"Name\":\"/"), []byte(fmt.Sprintf("\"Node\":%s,\"Name\":\"/", n)), -1)

		// insert node IP
		data = bytes.Replace(data, []byte("\"HostIp\":\"0.0.0.0\""), []byte(fmt.Sprintf("\"HostIp\":%q", container.Node().IP)), -1)
		w.Write(data)

	}
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

	//Find the node to place container firstly
	node, err := c.scheduler.SelectNodeForContainer(&config)

	if err != nil {
		log.Debug("Cannot find the node for create the container")
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//Create the abassador containers for links if need
	err = processLinks(c, node, &config.HostConfig)
	if err != nil {
		log.Debug("Failed to create the amabassadors for the links")
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//Create the container with the updated links
	container, err := node.Create(&config, name, true)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "{%q:%q}", "Id", container.Id)
	return
}

// DELETE /containers/{name:.*}
func deleteAssociatedAmbassadorContainers(c *context, container *cluster.Container) error {

	containers := c.cluster.Containers()

	src := container.Info.Name
	if len(src) > 1 {
		//Find the associated container by prefix/suffix of name
		cname := src[1:]
		suffix := "_ambassador_" + cname

		for _, ct := range containers {
			name := ct.Info.Name
			if strings.HasSuffix(name, suffix) && name[1:] == getAmbassadorName(ct.Node(), cname) {
				//TODO adding image checking
				log.Infof("Delete ambassador container'%s for %s", name, src)
				if err := c.scheduler.RemoveContainer(ct, true); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DELETE /containers/{name:.*}
func deleteContainer(c *context, w http.ResponseWriter, r *http.Request) {
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
	var err error

	//Delete the associated ambassador containers
	err = deleteAssociatedAmbassadorContainers(c, container)

	if err == nil {
		err = c.scheduler.RemoveContainer(container, force)
	}

	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
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

// GET /_ping
func ping(c *context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

// Proxy a request to the right node and do a force refresh
func proxyContainerAndForceRefresh(c *context, w http.ResponseWriter, r *http.Request) {
	container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := proxy(c.tlsConfig, container, w, r); err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
	}

	log.Debugf("[REFRESH CONTAINER] --> %s", container.Id)
	container.Node().ForceRefreshContainer(container.Container)
}

// Proxy a request to the right node
func proxyContainer(c *context, w http.ResponseWriter, r *http.Request) {
	container, err := getContainerFromVars(c, mux.Vars(r))
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := proxy(c.tlsConfig, container, w, r); err != nil {
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

	if err := hijack(c.tlsConfig, container, w, r); err != nil {
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
	log.Error(err)
	http.Error(w, err, status)
}

func createRouter(c *context, enableCors bool) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]handler{
		"GET": {
			"/_ping":                          ping,
			"/events":                         getEvents,
			"/info":                           getInfo,
			"/version":                        getVersion,
			"/images/json":                    notImplementedHandler,
			"/images/viz":                     notImplementedHandler,
			"/images/search":                  notImplementedHandler,
			"/images/get":                     notImplementedHandler,
			"/images/{name:.*}/get":           notImplementedHandler,
			"/images/{name:.*}/history":       notImplementedHandler,
			"/images/{name:.*}/json":          notImplementedHandler,
			"/containers/ps":                  getContainersJSON,
			"/containers/json":                getContainersJSON,
			"/containers/{name:.*}/export":    proxyContainer,
			"/containers/{name:.*}/changes":   proxyContainer,
			"/containers/{name:.*}/json":      getContainerJSON,
			"/containers/{name:.*}/top":       proxyContainer,
			"/containers/{name:.*}/logs":      proxyContainer,
			"/containers/{name:.*}/attach/ws": notImplementedHandler,
			"/exec/{execid:.*}/json":          proxyContainer,
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
			"/containers/{name:.*}/kill":    proxyContainer,
			"/containers/{name:.*}/pause":   proxyContainer,
			"/containers/{name:.*}/unpause": proxyContainer,
			"/containers/{name:.*}/restart": proxyContainer,
			"/containers/{name:.*}/start":   startContainer,
			"/containers/{name:.*}/stop":    proxyContainer,
			"/containers/{name:.*}/wait":    proxyContainer,
			"/containers/{name:.*}/resize":  proxyContainer,
			"/containers/{name:.*}/attach":  proxyHijack,
			"/containers/{name:.*}/copy":    notImplementedHandler,
			"/containers/{name:.*}/exec":    proxyContainerAndForceRefresh,
			"/exec/{execid:.*}/start":       proxyHijack,
			"/exec/{execid:.*}/resize":      proxyContainer,
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

var AMBASSADOR_IMAGE string

//Get ambassador image from the OS env
func getAmbassadorImage() string {
	if AMBASSADOR_IMAGE == "" {
		img := os.Getenv("AMBASSADOR_IMAGE")
		if img == "" {
			img = "svendowideit/ambassador:latest"
		} else {
			list := strings.Split(img, "/")
			name := list[len(list)-1]
			if strings.Index(name, ":") < 0 {
				//Append the latest to the image
				img += ":latest"
			}
		}
		AMBASSADOR_IMAGE = img
	}
	return AMBASSADOR_IMAGE
}

func getAmbassadorName(node *cluster.Node, name string) string {
	nodeID := strings.Replace(node.ID, ":", "", -1)
	return fmt.Sprintf("node_%s_ambassador_%s", nodeID, name)
}

func processLinks(c *context, containerNode *cluster.Node, config *dockerclient.HostConfig) error {

	var EMPTY struct{}

	links := config.Links
	if links == nil {
		return nil
	}
	containers := c.cluster.Containers()

	cache := map[string](*dockerclient.ContainerInfo){}

	addr := containerNode.Addr

	var newLinks []string

	for _, link := range links {
		//Parse the link info
		linkInfo := strings.Split(link, ":")
		name, alias := linkInfo[0], linkInfo[1]
		linkContainerName := "/" + name
		for _, target := range containers {
			if target.Info.Name == linkContainerName {
				if addr == target.Node().Addr {
					log.Debug("No additional work for the container link on the same host")
				} else {
					//Update the link
					ambassadorName := getAmbassadorName(containerNode, name)
					link = ambassadorName + ":" + alias
					//Create ambassador container for cross-host link
					//TODO: Remove ambassador when the container is removed.
					var targetInfo *dockerclient.ContainerInfo
					var err error
					targetInfo = cache[target.Id]
					if targetInfo == nil {
						targetInfo, err = target.Node().InspectContainer(target.Id)
						if err == nil && targetInfo != nil {
							cache[target.Id] = targetInfo
							//Check if ambassadorName exists, if no create a new one

							ambassadorInfo, _ := containerNode.InspectContainer(ambassadorName)
							if ambassadorInfo == nil {

								var ambassadorConfig dockerclient.ContainerConfig
								var env []string
								ambassadorConfig.ExposedPorts = make(map[string]struct{})
								ambassadorConfig.Image = getAmbassadorImage()
								ipAddr := targetInfo.NetworkSettings.IpAddress
								//Set the port as the environment variable
								for p := range targetInfo.NetworkSettings.Ports {
									portInfo := strings.Split(p, "/")
									port, protocol := portInfo[0], portInfo[1]
									env = append(env, fmt.Sprintf("%s_PORT_%s_%s=%s://%s:%s", strings.ToUpper(alias), port, strings.ToUpper(protocol), protocol, ipAddr, port))
									ambassadorConfig.ExposedPorts[p] = EMPTY
								}
								//Copy the environment variables from linked container
								env = append(env, targetInfo.Config.Env...)

								ambassadorConfig.Env = env
								log.Infof("Create ambassador container %s with exposed ports: %v", ambassadorName, ambassadorConfig.ExposedPorts)
								var ambassadorContainer *cluster.Container
								ambassadorContainer, err := containerNode.Create(&ambassadorConfig, ambassadorName, false)
								if err != nil {
									log.Warningf("Failed to create the ambassador container %s: %v", ambassadorName, err)
									return err
								}
								var hostConfig dockerclient.HostConfig
								err = containerNode.StartContainer(ambassadorContainer.Id, &hostConfig)
								if err != nil {
									log.Warningf("Failed to start the ambassador container %s: %v", ambassadorName, err)
									return err
								}
							}
						} else {
							log.Warningf("Failed to inspect link container %s: %v", target.Id, err)
							return err
						}
					}
				}
				break
			}
		}
		newLinks = append(newLinks, link)
	}
	//Update the Links
	config.Links = newLinks
	return nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// Start the container and process the links for cross-host links
func startContainerHandler(c *context, w http.ResponseWriter, r *http.Request, container *cluster.Container) error {
	var config dockerclient.HostConfig
	var err error
	if r.ContentLength > 0 { //Ignore handling if no request body
		err = json.NewDecoder(r.Body).Decode(&config)

		if err == nil {
			err = processLinks(c, container.Node(), &config)
			if err == nil {
				new_body, _ := json.Marshal(config)
				r.Body = nopCloser{bytes.NewBuffer(new_body)}
				r.ContentLength = 0
			}
		}
	}

	return err
}

type ProxyHandler func(c *context, w http.ResponseWriter, r *http.Request, container *cluster.Container) error

func abstractContainerProxyImpl(c *context, w http.ResponseWriter, r *http.Request, handler ProxyHandler) {
	container := c.cluster.Container(mux.Vars(r)["name"])
	if container != nil {
		// The handler to intercept the request processing.
		if handler != nil {
			err := handler(c, w, r, container)
			if err != nil {
				httpError(w, err.Error(), http.StatusBadRequest)
				return
			}

		}
		// Use a new client for each request
		client := &http.Client{}

		// RequestURI may not be sent to client
		r.RequestURI = ""

		parts := strings.SplitN(container.Node().Addr, "://", 2)
		if len(parts) == 2 {
			r.URL.Scheme = parts[0]
			r.URL.Host = parts[1]
		} else {
			r.URL.Scheme = "http"
			r.URL.Host = parts[0]
		}

		log.Debugf("[PROXY] --> %s %s", r.Method, r.URL)
		resp, err := client.Do(r)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func startContainer(c *context, w http.ResponseWriter, r *http.Request) {
	abstractContainerProxyImpl(c, w, r, startContainerHandler)
}
