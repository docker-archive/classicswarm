package backends

import (
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/runconfig"
	"github.com/tutumcloud/go-tutum"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var (
	tutumConnector        = "docker.tutum.co:49460"
	tutumConnectorVersion = "v1.11"
)

func Tutum() engine.Installer {
	return &tutumBackend{}
}

type tutumBackend struct {
}

func (t *tutumBackend) Install(eng *engine.Engine) error {
	eng.Register("tutum", func(job *engine.Job) engine.Status {
		if len(job.Args) == 1 {
			tutumConnector = job.Args[0]
		}
		if !tutum.IsAuthenticated() {
			return job.Errorf("You need to provide your Tutum credentials in ~/.tutum or environment variables TUTUM_USER and TUTUM_APIKEY")
		}
		job.Eng.Register("containers", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/json?all=%s&limit=%s",
				url.QueryEscape(job.Getenv("all")),
				url.QueryEscape(job.Getenv("limit")),
			)
			resp, err := tutumConnectorCall("GET", path, "")
			if err != nil {
				return job.Errorf("%s: get: %v", path, err)
			}
			c := engine.NewTable("Created", 0)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return job.Errorf("%s: read body: %v", path, err)
			}
			if _, err := c.ReadListFrom(body); err != nil {
				return job.Errorf("%s: readlist: %v", path, err)
			}
			c.WriteListTo(job.Stdout)
			return engine.StatusOK
		})
		job.Eng.Register("create", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/create",
			)
			config := runconfig.ContainerConfigFromJob(job)
			data, err := json.Marshal(config)
			if err != nil {
				return job.Errorf("%s: json marshal: %v", path, err)
			}
			resp, err := tutumConnectorCall("POST", path, string(data))
			if err != nil {
				return job.Errorf("%s: post: %v", path, err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return job.Errorf("%s: read body: %#v", path, err)
			}
			var containerOut struct {
				Id       string
				Warnings []string
			}
			err = json.Unmarshal([]byte(body), &containerOut)
			_, err = job.Printf("%s\n", containerOut.Id)
			if err != nil {
				return job.Errorf("%s: write body: %#v", path, err)
			}
			log.Printf("%s", string(body))
			return engine.StatusOK
		})
		job.Eng.Register("container_delete", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s?force=%s",
				job.Args[0],
				url.QueryEscape(job.Getenv("forceRemove")),
			)
			_, err := tutumConnectorCall("DELETE", path, "")
			if err != nil {
				return job.Errorf("%s: delete: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("start", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf("/containers/%s/start", job.Args[0])
			config := runconfig.ContainerConfigFromJob(job)
			data, err := json.Marshal(config)
			if err != nil {
				return job.Errorf("%s: json marshal: %v", path, err)
			}
			_, err = tutumConnectorCall("POST", path, string(data))
			if err != nil {
				return job.Errorf("%s: post: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("stop", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s/stop?t=%s",
				job.Args[0],
				url.QueryEscape(job.Getenv("t")),
			)
			_, err := tutumConnectorCall("POST", path, "")
			if err != nil {
				return job.Errorf("%s: post: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("kill", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s/kill?signal=%s",
				job.Args[0],
				job.Args[1],
			)
			_, err := tutumConnectorCall("POST", path, "")
			if err != nil {
				return job.Errorf("%s: post: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("restart", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s/restart?t=%s",
				job.Args[0],
				url.QueryEscape(job.Getenv("t")),
			)
			_, err := tutumConnectorCall("POST", path, "")
			if err != nil {
				return job.Errorf("%s: post: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("inspect", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s/json",
				job.Args[0],
			)
			resp, err := tutumConnectorCall("GET", path, "")
			if err != nil {
				return job.Errorf("%s: get: %v", path, err)
			}
			_, err = io.Copy(job.Stdout, resp.Body)
			if err != nil {
				return job.Errorf("%s: copy stream: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("logs", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := fmt.Sprintf(
				"/containers/%s/logs?stdout=%s&stderr=%s",
				job.Args[0],
				url.QueryEscape(job.Getenv("stdout")),
				url.QueryEscape(job.Getenv("stderr")),
			)
			resp, err := tutumConnectorCall("GET", path, "")
			if err != nil {
				return job.Errorf("%s: get: %v", path, err)
			}
			_, err = io.Copy(job.Stdout, resp.Body)
			if err != nil {
				return job.Errorf("%s: copy stream: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.Register("version", func(job *engine.Job) engine.Status {
			log.Printf("Received '%s' operation....", job.Name)
			path := "/version"
			resp, err := tutumConnectorCall("GET", path, "")
			if err != nil {
				return job.Errorf("%s: get: %v", path, err)
			}
			_, err = io.Copy(job.Stdout, resp.Body)
			if err != nil {
				return job.Errorf("%s: copy stream: %v", path, err)
			}
			return engine.StatusOK
		})
		job.Eng.RegisterCatchall(func(job *engine.Job) engine.Status {
			return job.Errorf("Operation not yet supported: %s", job.Name)
		})
		return engine.StatusOK
	})
	return nil
}

func tutumConnectorCall(method, path, body string) (*http.Response, error) {
	apiPath := fmt.Sprintf(
		"/%s%s",
		tutumConnectorVersion,
		path,
	)
	u, err := url.Parse(apiPath)
	if err != nil {
		return nil, err
	}
	u.Host = tutumConnector
	u.Scheme = "https"
	log.Printf("[tutum] >> Calling connector: %s %s", method, path)
	req, err := http.NewRequest(method, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	authHeader := fmt.Sprintf("ApiKey %s:%s", tutum.User, tutum.ApiKey)
	req.Header.Add("Authorization", authHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	log.Printf("[tutum] << Response: %d", resp.StatusCode)
	if resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%s %s: read body: %#v", method, path, err)
		}
		return nil, fmt.Errorf("%s %s: %s", method, path, body)
	}
	return resp, nil
}
