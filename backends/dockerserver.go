package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
	"github.com/dotcloud/docker/api"
	"github.com/dotcloud/docker/pkg/version"
	dockerContainerConfig "github.com/dotcloud/docker/runconfig"
	dockerutils "github.com/dotcloud/docker/utils"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

func DockerServer() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnVerb(libswarm.Spawn, libswarm.Handler(func(ctx *libswarm.Message) error {
		instance := utils.Task(func(in libswarm.Receiver, out libswarm.Sender) {
			url := "tcp://localhost:4243"
			if len(ctx.Args) > 0 {
				url = ctx.Args[0]
			}
			err := listenAndServe(url, out)
			if err != nil {
				fmt.Printf("listenAndServe: %v", err)
			}
		})
		_, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: instance})
		return err
	}))
	return backend
}

type HttpApiFunc func(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func listenAndServe(urlStr string, out libswarm.Sender) error {
	fmt.Println("Starting Docker server...")
	r, err := createRouter(out)
	if err != nil {
		return err
	}

	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	var hostAndPath string
	// For Unix sockets we need to capture the path as well as the host
	if parsedUrl.Scheme == "unix" {
		hostAndPath = "/" + parsedUrl.Host + parsedUrl.Path
	} else {
		hostAndPath = parsedUrl.Host
	}

	l, err := net.Listen(parsedUrl.Scheme, hostAndPath)
	if err != nil {
		return err
	}
	httpSrv := http.Server{Addr: hostAndPath, Handler: r}
	return httpSrv.Serve(l)
}

func ping(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	_, err := w.Write([]byte{'O', 'K'})
	return err
}

type containerJson struct {
	Config          *dockerContainerConfig.Config
	HostConfig      *dockerContainerConfig.HostConfig
	ID              string
	Created         string
	Driver          string
	ExecDriver      string
	NetworkSettings struct {
		Bridge      string
		Gateway     string
		IPAddress   string
		IPPrefixLen int
		Ports       map[string][]map[string]string
	}
	State struct {
		Running    bool
		StartedAt  string
		FinishedAt string
		ExitCode   int
		Paused     bool
		Pid        int
	}
	Name           string
	Args           []string
	ProcessLbael   string
	ResolvConfPath string
	HostnamePath   string
	HostsPath      string
	MountLabel     string
	Volumes        map[string]string
	VolumesRW      map[string]bool
}

func getContainerJson(out libswarm.Sender, containerID string) (containerJson, error) {
	o := libswarm.AsClient(out)

	_, containerOut, err := o.Attach(containerID)
	if err != nil {
		return containerJson{}, err
	}
	container := libswarm.AsClient(containerOut)
	responseJson, err := container.Get()
	if err != nil {
		return containerJson{}, err
	}
	var response containerJson

	if err = json.Unmarshal([]byte(responseJson), &response); err != nil {
		return containerJson{}, err
	}

	return response, nil
}

func getContainerInfo(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	container, err := getContainerJson(out, vars["name"])
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, container)
}

func getContainersJSON(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	o := libswarm.AsClient(out)
	names, err := o.Ls()
	if err != nil {
		return err
	}

	var responses []interface{}

	for _, name := range names {
		response, err := getContainerJson(out, name)
		if err != nil {
			return err
		}
		created, err := time.Parse(time.RFC3339, response.Created)
		if err != nil {
			return err
		}
		var state string
		if response.State.Running {
			state = "Up"
		} else {
			state = fmt.Sprintf("Exited (%d)", response.State.ExitCode)
		}
		type port struct {
			IP          string
			PrivatePort int
			PublicPort  int
			Type        string
		}
		var ports []port
		for p, mappings := range response.NetworkSettings.Ports {
			var portnum int
			var proto string
			_, err := fmt.Sscanf(p, "%d/%s", &portnum, &proto)
			if err != nil {
				return err
			}
			if len(mappings) > 0 {
				for _, mapping := range mappings {
					hostPort, err := strconv.Atoi(mapping["HostPort"])
					if err != nil {
						return err
					}
					newport := port{
						IP:          mapping["HostIp"],
						PrivatePort: portnum,
						PublicPort:  hostPort,
						Type:        proto,
					}
					ports = append(ports, newport)
				}
			} else {
				newport := port{
					PublicPort: portnum,
					Type:       proto,
				}
				ports = append(ports, newport)
			}
		}
		responses = append(responses, map[string]interface{}{
			"Id":      response.ID,
			"Command": strings.Join(response.Config.Cmd, " "),
			"Created": created.Unix(),
			"Image":   response.Config.Image,
			"Names":   []string{response.Name},
			"Ports":   ports,
			"Status":  state,
		})
	}

	return writeJSON(w, http.StatusOK, responses)
}

func postContainersCreate(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	name := r.Form.Get("name")
	if name != "" {
		var reqJson map[string]interface{}
		if err = json.Unmarshal(body, &reqJson); err != nil {
			return err
		}
		reqJson["name"] = name
		body, err = json.Marshal(reqJson)
		if err != nil {
			return err
		}
	}

	container, err := libswarm.AsClient(out).Spawn(string(body))
	if err != nil {
		return err
	}

	responseJson, err := container.Get()
	if err != nil {
		return err
	}

	var response struct{ Id string }
	if err = json.Unmarshal([]byte(responseJson), &response); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, response)
}

func postContainersStart(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// TODO: r.Body

	name := vars["name"]
	_, containerOut, err := libswarm.AsClient(out).Attach(name)
	container := libswarm.AsClient(containerOut)
	if err != nil {
		return err
	}
	if err := container.Start(); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func postContainersStop(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	_, containerOut, err := libswarm.AsClient(out).Attach(name)
	container := libswarm.AsClient(containerOut)
	if err != nil {
		return err
	}
	if err := container.Stop(); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}

func postContainersAttach(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	inStream, outStream, err := hijackServer(w)
	if err != nil {
		return err
	}
	defer func() {
		if tcpc, ok := inStream.(*net.TCPConn); ok {
			tcpc.CloseWrite()
		} else {
			inStream.Close()
		}
	}()
	defer func() {
		if tcpc, ok := outStream.(*net.TCPConn); ok {
			tcpc.CloseWrite()
		} else if closer, ok := outStream.(io.Closer); ok {
			closer.Close()
		}
	}()

	fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n")

	// TODO: if a TTY, then no multiplexing is done
	errStream := dockerutils.NewStdWriter(outStream, dockerutils.Stderr)
	outStream = dockerutils.NewStdWriter(outStream, dockerutils.Stdout)

	_, containerOut, err := libswarm.AsClient(out).Attach(vars["name"])
	if err != nil {
		return err
	}
	container := libswarm.AsClient(containerOut)

	containerR, _, err := container.Attach("")
	var tasks sync.WaitGroup
	go func() {
		defer tasks.Done()
		err := utils.DecodeStream(outStream, containerR, "stdout")
		if err != nil {
			fmt.Printf("decodestream: %v\n", err)
		}
	}()
	tasks.Add(1)
	go func() {
		defer tasks.Done()
		err := utils.DecodeStream(errStream, containerR, "stderr")
		if err != nil {
			fmt.Printf("decodestream: %v\n", err)
		}
	}()
	tasks.Add(1)
	tasks.Wait()

	return nil
}

func postContainersWait(out libswarm.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// TODO: this should wait for container to end out output correct
	// exit status
	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"StatusCode": "0",
	})
}

func createRouter(out libswarm.Sender) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":                  ping,
			"/containers/json":        getContainersJSON,
			"/containers/{name}/json": getContainerInfo,
		},
		"POST": {
			"/containers/create":           postContainersCreate,
			"/containers/{name:.*}/attach": postContainersAttach,
			"/containers/{name:.*}/start":  postContainersStart,
			"/containers/{name:.*}/stop":   postContainersStop,
			"/containers/{name:.*}/wait":   postContainersWait,
		},
		"DELETE":  {},
		"OPTIONS": {},
	}

	for method, routes := range m {
		for route, fct := range routes {
			localRoute := route
			localFct := fct
			localMethod := method

			f := makeHttpHandler(out, localMethod, localRoute, localFct, version.Version("0.11.0"))

			// add the new route
			if localRoute == "" {
				r.Methods(localMethod).HandlerFunc(f)
			} else {
				r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(f)
				r.Path(localRoute).Methods(localMethod).HandlerFunc(f)
			}
		}
	}

	return r, nil
}

func makeHttpHandler(out libswarm.Sender, localMethod string, localRoute string, handlerFunc HttpApiFunc, dockerVersion version.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		fmt.Printf("Calling %s %s\n", localMethod, localRoute)

		version := version.Version(mux.Vars(r)["version"])
		if version == "" {
			version = api.APIVERSION
		}

		if err := handlerFunc(out, version, w, r, mux.Vars(r)); err != nil {
			fmt.Printf("Error: %s", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}
