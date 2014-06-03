package backends

import (
	"bytes"
	"log"
	"net/url"

	"github.com/dotcloud/docker/engine"
	"github.com/openshift/geard/containers"
	chttp "github.com/openshift/geard/containers/http"
	http "github.com/openshift/geard/http"
	"github.com/openshift/geard/jobs"
	"github.com/openshift/geard/port"
)

func GeardBackend() engine.Installer {
	return &geard{}
}

type geard struct {
}

func (s *geard) Install(eng *engine.Engine) error {
	eng.Register("geard", func(job *engine.Job) engine.Status {
		if len(job.Args) < 1 {
			return job.Errorf("usage: %s <proto>://<addr>", job.Name)
		}
		addr := job.Args[0]
		target, err := url.Parse(addr)
		if err != nil {
			return job.Errorf("%s is not a valid URL to a geard host: %s", addr, err.Error())
		}

		transport := http.NewHttpTransport()

		job.Eng.Register("create", func(job *engine.Job) engine.Status {
			if len(job.Args) < 1 {
				return job.Errorf("usage: --name is required for installing containers with geard")
			}
			id := job.Args[0]
			image := job.Getenv("Image")
			cmd := job.GetenvList("Cmd")

			// hack, for the sake of example
			pairs := port.PortPairs{}
			for i := range cmd {
				if cmd[i] == "-p" && i < len(cmd)-1 {
					arg, err := port.FromPortPairHeader(cmd[i+1])
					if err != nil {
						return job.Errorf("usage: expected -p %s to be a pair of ports: %s", cmd[i+1], err.Error())
					}
					pairs = append(pairs, arg...)
				}
			}

			req := &chttp.HttpInstallContainerRequest{}
			req.InstallContainerRequest.Id = containers.Identifier(id)
			req.InstallContainerRequest.Image = image
			req.InstallContainerRequest.Ports = pairs

			buf := bytes.Buffer{}
			res := &jobs.ClientResponse{Output: &buf}
			if err := transport.ExecuteRemote(target, req, res); err != nil {
				return job.Errorf("Failed to connect to %s: %s", addr, err.Error())
			}
			if res.Error != nil {
				return job.Errorf("Install container failed", res.Error.Error())
			}

			log.Print(buf.String())

			_, err := job.Printf("%s\n", id)
			if err != nil {
				return job.Errorf("%s: write body: %#v", target.String(), err)
			}

			return engine.StatusOK
		})
		job.Eng.Register("start", func(job *engine.Job) engine.Status {
			if len(job.Args) != 1 {
				return job.Errorf("usage: %s <container>", job.Name)
			}

			id := job.Args[0]
			req := &chttp.HttpStartContainerRequest{}
			req.StartedContainerStateRequest.Id = containers.Identifier(id)

			buf := bytes.Buffer{}
			res := &jobs.ClientResponse{Output: &buf}
			if err := transport.ExecuteRemote(target, req, res); err != nil {
				return job.Errorf("Failed to connect to %s: %s", addr, err.Error())
			}
			if res.Error != nil {
				return job.Errorf("Start container failed", res.Error.Error())
			}

			log.Print(buf.String())
			return engine.StatusOK
		})

		job.Eng.Register("stop", func(job *engine.Job) engine.Status {
			if len(job.Args) != 1 {
				return job.Errorf("usage: %s <container>", job.Name)
			}

			id := job.Args[0]
			req := &chttp.HttpStopContainerRequest{}
			req.StoppedContainerStateRequest.Id = containers.Identifier(id)

			buf := bytes.Buffer{}
			res := &jobs.ClientResponse{Output: &buf}
			if err := transport.ExecuteRemote(target, req, res); err != nil {
				return job.Errorf("Failed to connect to %s: %s", addr, err.Error())
			}
			if res.Error != nil {
				return job.Errorf("Stop container failed", res.Error.Error())
			}

			log.Print(buf.String())
			return engine.StatusOK
		})

		job.Eng.RegisterCatchall(func(job *engine.Job) engine.Status {
			log.Printf("%#v %#v %#v", *job, job.Env(), job.Args)
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
