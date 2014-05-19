package backends

import (
	"fmt"
	"github.com/dotcloud/docker/engine"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func Forward() engine.Installer {
	return &forwarder{}
}

type forwarder struct {
}

func (f *forwarder) Install(eng *engine.Engine) error {
	eng.Register("forward", func(job *engine.Job) engine.Status {
		if len(job.Args) != 1 {
			return job.Errorf("usage: %s <proto>://<addr>", job.Name)
		}
		client, err := newClient(job.Args[0], "v0.10")
		if err != nil {
			return job.Errorf("%v", err)
		}
		job.Eng.Register("containers", client.containers)
		return engine.StatusOK
	})
	return nil
}

type client struct {
	URL     *url.URL
	proto   string
	addr    string
	version string
}

func newClient(peer, version string) (*client, error) {
	u, err := url.Parse(peer)
	if err != nil {
		return nil, err
	}
	c := &client{
		URL:     u,
		version: version,
	}
	c.URL.Scheme = "http"
	return c, nil
}

func (c *client) call(method, path, body string) (*http.Response, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u.Host = c.URL.Host
	u.Scheme = c.URL.Scheme
	req, err := http.NewRequest(method, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *client) containers(job *engine.Job) engine.Status {
	path := fmt.Sprintf(
		"/containers/json?all=%s&size=%s&since=%s&before=%s&limit=%s",
		url.QueryEscape(job.Getenv("all")),
		url.QueryEscape(job.Getenv("size")),
		url.QueryEscape(job.Getenv("since")),
		url.QueryEscape(job.Getenv("before")),
		url.QueryEscape(job.Getenv("limit")),
	)
	resp, err := c.call("GET", path, "")
	if err != nil {
		return job.Errorf("%s: get: %v", c.URL.String(), err)
	}
	// FIXME: check for response error
	t := engine.NewTable("Created", 0)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return job.Errorf("%s: read body: %v", c.URL.String(), err)
	}
	fmt.Printf("---> '%s'\n", body)
	if _, err := t.ReadListFrom(body); err != nil {
		return job.Errorf("%s: readlist: %v", c.URL.String(), err)
	}
	t.WriteListTo(job.Stdout)
	return engine.StatusOK
}
