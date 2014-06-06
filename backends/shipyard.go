package backends

import (
	"fmt"
	"github.com/dotcloud/docker/engine"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func Shipyard() engine.Installer {
	return &shipyard{}
}

type shipyard struct {
	url, user, pass string
}

func (s *shipyard) Install(eng *engine.Engine) error {
	eng.Register("shipyard", func(job *engine.Job) engine.Status {
		if len(job.Args) != 3 {
			return job.Errorf("usage: <shipyard url> <user> <pass>")
		}
		s.url = job.Args[0]
		s.user = job.Args[1]
		s.pass = job.Args[2]
		job.Eng.Register("containers", s.containers)
		job.Eng.Register("container_inspect", s.container_inspect)
		return engine.StatusOK
	})

	return nil
}

func (c *shipyard) containers(job *engine.Job) engine.Status {
	var args string
	if job.Getenv("limit") != "" {
		args = fmt.Sprintf("%s&limit=%s", args, url.QueryEscape(job.Getenv("limit")))
	}
	if job.Getenv("offset") != "" {
		args = fmt.Sprintf("%s&offset=%s", args, url.QueryEscape(job.Getenv("offset")))
	}

	path := fmt.Sprintf("containers/?%s", args)
	out, err := c.gateway("GET", path, "")
	if err != nil {
		return job.Errorf("%s: get: %v", path, err)
	}
	log.Printf("%s", out)
	return engine.StatusOK
}

func (c *shipyard) container_inspect(job *engine.Job) engine.Status {
	path := fmt.Sprintf("containers/details/%s", job.Args[0])
	out, err := c.gateway("GET", path, "")
	if err != nil {
		return job.Errorf("%s: get: %v", path, err)
	}
	log.Printf("%s", out)
	return engine.StatusOK
}

func (c *shipyard) gateway(method, path, body string) ([]byte, error) {
	apiPath := fmt.Sprintf("%s/api/v1/%s&format=json", c.url, path)
	url, err := url.Parse(apiPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "ApiKey "+c.api_key())
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *shipyard) api_key() string {
	return c.user + ":" + c.pass
}
