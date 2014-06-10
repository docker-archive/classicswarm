package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/tutumcloud/go-tutum"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	tutumConnectorHost    = "https://docker.tutum.co:49460"
	tutumConnectorVersion = "v1.11"
)

func Tutum() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		if len(ctx.Args) == 2 {
			tutum.User = ctx.Args[0]
			tutum.ApiKey = ctx.Args[1]
		}
		if !tutum.IsAuthenticated() {
			return fmt.Errorf("You need to provide your Tutum credentials in ~/.tutum or environment variables TUTUM_USER and TUTUM_APIKEY")
		}
		tutumDockerConnector, err := newConnector(tutumConnectorHost, tutumConnectorVersion)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		t := &tutumBackend{
			tutumDockerConnector: tutumDockerConnector,
			Server:               beam.NewServer(),
		}
		t.Server.OnAttach(beam.Handler(t.attach))
		t.Server.OnStart(beam.Handler(t.ack))
		t.Server.OnLs(beam.Handler(t.ls))
		t.Server.OnSpawn(beam.Handler(t.spawn))
		_, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: t.Server})
		return err
	}))
	return backend
}

type tutumBackend struct {
	tutumDockerConnector *tutumDockerConnector
	*beam.Server
}

func (t *tutumBackend) attach(ctx *beam.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: t.Server})
		for {
			time.Sleep(1 * time.Second)
		}
	} else {
		c := t.newContainer(ctx.Args[0])
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
	}
	return nil
}

func (t *tutumBackend) ack(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: t.Server})
	return nil
}

func (t *tutumBackend) ls(ctx *beam.Message) error {
	resp, err := t.tutumDockerConnector.call("GET", "/containers/json", "")
	if err != nil {
		return fmt.Errorf("%s: get: %v", t.tutumDockerConnector.URL.String(), err)
	}
	c := engine.NewTable("Created", 0)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: read body: %v", t.tutumDockerConnector.URL.String(), err)
	}
	if _, err := c.ReadListFrom(body); err != nil {
		return fmt.Errorf("%s: readlist: %v", t.tutumDockerConnector.URL.String(), err)
	}
	ids := []string{}
	for _, env := range c.Data {
		ids = append(ids, env.GetList("Id")[0])
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: ids}); err != nil {
		return fmt.Errorf("%s: send response: %v", t.tutumDockerConnector.URL.String(), err)
	}
	return nil
}

func (t *tutumBackend) spawn(ctx *beam.Message) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("tutum: spawn takes exactly 1 argument, got %d", len(ctx.Args))
	}
	resp, err := t.tutumDockerConnector.call("POST", "/containers/create", ctx.Args[0])
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return fmt.Errorf("expected status code 201, got %d:\n%s", resp.StatusCode, respBody)
	}
	var respJson struct{ Id string }
	if err = json.Unmarshal(respBody, &respJson); err != nil {
		return err
	}
	c := t.newContainer(respJson.Id)
	if _, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c}); err != nil {
		return err
	}
	return nil
}

func (t *tutumBackend) newContainer(id string) beam.Sender {
	c := &tutumContainer{tutumBackend: t, id: id}
	instance := beam.NewServer()
	instance.OnGet(beam.Handler(c.get))
	instance.OnStart(beam.Handler(c.start))
	instance.OnStop(beam.Handler(c.stop))
	return instance
}

type tutumContainer struct {
	tutumBackend *tutumBackend
	id           string
}

func (c *tutumContainer) get(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/json", c.id)
	resp, err := c.tutumBackend.tutumDockerConnector.call("GET", path, "")
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("%s", respBody)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s", respBody)
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{string(respBody)}}); err != nil {
		return err
	}
	return nil
}

func (c *tutumContainer) start(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/start", c.id)
	resp, err := c.tutumBackend.tutumDockerConnector.call("POST", path, "")
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("expected status code 204, got %d:\n%s", resp.StatusCode, respBody)
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	}
	return nil
}

func (c *tutumContainer) stop(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/stop", c.id)
	resp, err := c.tutumBackend.tutumDockerConnector.call("POST", path, "")
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("expected status code 204, got %d:\n%s", resp.StatusCode, respBody)
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	}
	return nil
}

type tutumDockerConnector struct {
	URL     *url.URL
	version string
}

func newConnector(peer, version string) (*tutumDockerConnector, error) {
	u, err := url.Parse(peer)
	if err != nil {
		return nil, err
	}
	c := &tutumDockerConnector{
		URL:     u,
		version: version,
	}
	return c, nil
}

func (c *tutumDockerConnector) call(method, path, body string) (*http.Response, error) {
	path = fmt.Sprintf("/%s%s", c.version, path)
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
	authHeader := fmt.Sprintf("ApiKey %s:%s", tutum.User, tutum.ApiKey)
	req.Header.Add("Authorization", authHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%s %s: read body: %#v", method, path, err)
		}
		return nil, fmt.Errorf("%s %s: %s", method, path, body)
	}
	return resp, nil
}
