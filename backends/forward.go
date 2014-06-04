package backends

import (
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Forward() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		if len(ctx.Args) != 1 {
			return fmt.Errorf("forward: spawn takes exactly 1 argument, got %d", len(ctx.Args))
		}
		client, err := newClient(ctx.Args[0], "v0.10")
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		f := &forwarder{client: client}
		instance := beam.NewServer()
		instance.OnAttach(beam.Handler(f.attach))
		instance.OnStart(beam.Handler(f.start))
		instance.OnLs(beam.Handler(f.ls))
		_, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: instance})
		return err
	}))
	return backend
}

type forwarder struct {
	client *client
}

func (f *forwarder) attach(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	for {
		time.Sleep(1 * time.Second)
		(&beam.Object{ctx.Ret}).Log("forward: heartbeat")
	}
	return nil
}

func (f *forwarder) start(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (f *forwarder) ls(ctx *beam.Message) error {
	resp, err := f.client.call("GET", "/containers/json", "")
	if err != nil {
		return fmt.Errorf("%s: get: %v", f.client.URL.String(), err)
	}
	// FIXME: check for response error
	c := engine.NewTable("Created", 0)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: read body: %v", f.client.URL.String(), err)
	}
	if _, err := c.ReadListFrom(body); err != nil {
		return fmt.Errorf("%s: readlist: %v", f.client.URL.String(), err)
	}
	names := []string{}
	for _, env := range c.Data {
		names = append(names, env.GetList("Names")[0][1:])
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: names}); err != nil {
		return fmt.Errorf("%s: send response: %v", f.client.URL.String(), err)
	}
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
