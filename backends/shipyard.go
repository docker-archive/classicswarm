package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Shipyard() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		if len(ctx.Args) != 3 {
			return fmt.Errorf("Shipyard: Usage <shipyard URL> <user> <pass>")
		}

		c := &shipyard{url: ctx.Args[0], user: ctx.Args[1], pass: ctx.Args[2]}

		c.Server = beam.NewServer()
		c.Server.OnAttach(beam.Handler(c.attach))
		c.Server.OnStart(beam.Handler(c.start))
		c.Server.OnLs(beam.Handler(c.containers))
		c.OnGet(beam.Handler(c.containerInspect))
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
		return err
	}))
	return backend
}

func (c *shipyard) attach(ctx *beam.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
		for {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (c *shipyard) start(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

type shipyard struct {
	url, user, pass string
	*beam.Server
}

func (c *shipyard) containers(ctx *beam.Message) error {
	out, err := c.gateway("GET", "containers", "")
	if err != nil {
		return err
	}
	var data shipyardObjects
	json.Unmarshal(out, &data)
	var ids []string
	for _, c := range data.Objects {
		ids = append(ids, c.Id)
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: ids}); err != nil {
		return err
	}
	return nil
}

type shipyardObjects struct {
	Objects []shipyardObject `json:"objects"`
}

type shipyardObject struct {
	Id string `json:"container_id"`
}

func (c *shipyard) containerInspect(ctx *beam.Message) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("Expected 1 container id, got %s", len(ctx.Args))
	}
	path := fmt.Sprintf("containers/details/%s", ctx.Args[0])
	out, err := c.gateway("GET", path, "")
	if err != nil {
		return err
	}
	var data shipyardObject
	json.Unmarshal(out, &data)
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{"foo", "bar"}}); err != nil {
		return err
	}
	return nil
}

func (c *shipyard) gateway(method, path, body string) ([]byte, error) {
	apiPath := fmt.Sprintf("%s/api/v1/%s/?format=json", c.url, path)
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
