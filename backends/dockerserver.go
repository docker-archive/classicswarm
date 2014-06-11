package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/api"
	"github.com/dotcloud/docker/pkg/version"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func DockerServer() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		instance := beam.Task(func(in beam.Receiver, out beam.Sender) {
			url := "tcp://localhost:4243"
			if len(ctx.Args) > 0 {
				url = ctx.Args[0]
			}
			err := listenAndServe(url, out)
			if err != nil {
				fmt.Printf("listenAndServe: %v", err)
			}
		})
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}

type HttpApiFunc func(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func listenAndServe(urlStr string, out beam.Sender) error {
	fmt.Println("Starting Docker server...")
	r, err := createRouter(out)
	if err != nil {
		return err
	}

	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	l, err := net.Listen(parsedUrl.Scheme, parsedUrl.Host)
	if err != nil {
		return err
	}
	httpSrv := http.Server{Addr: parsedUrl.Host, Handler: r}
	return httpSrv.Serve(l)
}

func ping(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	_, err := w.Write([]byte{'O', 'K'})
	return err
}

func getContainersJSON(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	o := beam.Obj(out)
	names, err := o.Ls()
	if err != nil {
		return err
	}

	var responses []interface{}

	for _, name := range names {
		_, containerOut, err := o.Attach(name)
		if err != nil {
			return err
		}
		container := beam.Obj(containerOut)
		responseJson, err := container.Get()
		if err != nil {
			return err
		}
		var response struct {
			ID      string
			Created string
			Name    string
			Config  struct {
				Cmd   []string
				Image string
			}
			State struct {
				Running    bool
				StartedAt  string
				FinishedAt string
				ExitCode   int
			}
		}
		if err = json.Unmarshal([]byte(responseJson), &response); err != nil {
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
		responses = append(responses, map[string]interface{}{
			"Id":      response.ID,
			"Command": strings.Join(response.Config.Cmd, " "),
			"Created": created.Unix(),
			"Image":   response.Config.Image,
			"Names":   []string{response.Name},
			"Ports":   []string{},
			"Status":  state,
		})
	}

	return writeJSON(w, http.StatusOK, responses)
}

func postContainersCreate(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	container, err := beam.Obj(out).Spawn(string(body))
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

func postContainersStart(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	// TODO: r.Body

	name := vars["name"]
	_, containerOut, err := beam.Obj(out).Attach(name)
	container := beam.Obj(containerOut)
	if err != nil {
		return err
	}
	if err := container.Start(); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func postContainersStop(out beam.Sender, version version.Version, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if vars == nil {
		return fmt.Errorf("Missing parameter")
	}

	name := vars["name"]
	_, containerOut, err := beam.Obj(out).Attach(name)
	container := beam.Obj(containerOut)
	if err != nil {
		return err
	}
	if err := container.Stop(); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func createRouter(out beam.Sender) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":           ping,
			"/containers/json": getContainersJSON,
		},
		"POST": {
			"/containers/create":          postContainersCreate,
			"/containers/{name:.*}/start": postContainersStart,
			"/containers/{name:.*}/stop":  postContainersStop,
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

func makeHttpHandler(out beam.Sender, localMethod string, localRoute string, handlerFunc HttpApiFunc, dockerVersion version.Version) http.HandlerFunc {
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
