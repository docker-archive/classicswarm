package backends

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type DockerClientConfig struct {
	Scheme          string
	URLHost         string
	TLSClientConfig *tls.Config
}

func DockerClient() beam.Sender {
	return DockerClientWithConfig(&DockerClientConfig{
		Scheme:  "http",
		URLHost: "dummy.host",
	})
}

func DockerClientWithConfig(config *DockerClientConfig) beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		if len(ctx.Args) != 1 {
			return fmt.Errorf("dockerclient: spawn takes exactly 1 argument, got %d", len(ctx.Args))
		}
		client := newClient()
		client.scheme = config.Scheme
		client.urlHost = config.URLHost
		client.transport.TLSClientConfig = config.TLSClientConfig
		client.setURL(ctx.Args[0])
		b := &dockerClientBackend{
			client: client,
			Server: beam.NewServer(),
		}
		b.Server.OnAttach(beam.Handler(b.attach))
		b.Server.OnStart(beam.Handler(b.start))
		b.Server.OnLs(beam.Handler(b.ls))
		b.Server.OnSpawn(beam.Handler(b.spawn))
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: b.Server})
		return err
	}))
	return backend
}

type dockerClientBackend struct {
	client *client
	*beam.Server
}

func (b *dockerClientBackend) attach(ctx *beam.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: b.Server})
		<-make(chan struct{})
	} else {
		path := fmt.Sprintf("/containers/%s/json", ctx.Args[0])
		resp, err := b.client.call("GET", path, "")
		if err != nil {
			return err
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("%s", respBody)
		}
		c := b.newContainer(ctx.Args[0])
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
	}
	return nil
}

func (b *dockerClientBackend) start(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (b *dockerClientBackend) ls(ctx *beam.Message) error {
	resp, err := b.client.call("GET", "/containers/json", "")
	if err != nil {
		return fmt.Errorf("get: %v", err)
	}
	// FIXME: check for response error
	c := engine.NewTable("Created", 0)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %v", err)
	}
	if _, err := c.ReadListFrom(body); err != nil {
		return fmt.Errorf("readlist: %v", err)
	}
	names := []string{}
	for _, env := range c.Data {
		names = append(names, env.GetList("Names")[0][1:])
	}
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: names}); err != nil {
		return fmt.Errorf("send response: %v", err)
	}
	return nil
}

func (b *dockerClientBackend) spawn(ctx *beam.Message) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("dockerclient: spawn takes exactly 1 argument, got %d", len(ctx.Args))
	}
	resp, err := b.client.call("POST", "/containers/create", ctx.Args[0])
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
	c := b.newContainer(respJson.Id)
	if _, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c}); err != nil {
		return err
	}
	return nil
}

func (b *dockerClientBackend) newContainer(id string) beam.Sender {
	c := &container{backend: b, id: id}
	instance := beam.NewServer()
	instance.OnAttach(beam.Handler(c.attach))
	instance.OnStart(beam.Handler(c.start))
	instance.OnStop(beam.Handler(c.stop))
	instance.OnGet(beam.Handler(c.get))
	return instance
}

type container struct {
	backend *dockerClientBackend
	id      string
}

func (c *container) attach(ctx *beam.Message) error {
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	}

	path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	go beam.EncodeStream(ctx.Ret, stdoutR, "stdout")
	go beam.EncodeStream(ctx.Ret, stderrR, "stderr")
	c.backend.client.hijack("POST", path, nil, stdoutW, stderrW)

	return nil
}

func (c *container) start(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/start", c.id)
	resp, err := c.backend.client.call("POST", path, "{}")
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

func (c *container) stop(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/stop", c.id)
	resp, err := c.backend.client.call("POST", path, "")
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

func (c *container) get(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/json", c.id)
	resp, err := c.backend.client.call("GET", path, "")
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
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

type client struct {
	transport *http.Transport
	urlHost   string
	scheme    string
	version   string
}

func newClient() *client {
	return &client{
		transport: &http.Transport{},
		urlHost:   "dummy.host",
		scheme:    "http",
		version:   "v1.11",
	}
}

func (c *client) setURL(url string) {
	parts := strings.SplitN(url, "://", 2)
	proto, host := parts[0], parts[1]
	c.transport.Dial = func(_, _ string) (net.Conn, error) {
		return net.Dial(proto, host)
	}
}

func (c *client) call(method, path, body string) (*http.Response, error) {
	path = fmt.Sprintf("/%s%s", c.version, path)
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u.Host = c.urlHost
	u.Scheme = c.scheme
	req, err := http.NewRequest(method, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Transport: c.transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *client) hijack(method, path string, in io.ReadCloser, stdout, stderr io.Writer) error {
	path = fmt.Sprintf("/%s%s", c.version, path)
	dial, err := c.transport.Dial("ignored", "ignored")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return err
	}

	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()

	clientconn.Do(req)

	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	receiveStdout := utils.Go(func() (err error) {
		defer func() {
			if in != nil {
				in.Close()
			}
		}()
		_, err = utils.StdCopy(stdout, stderr, br)
		utils.Debugf("[hijack] End of stdout")
		return err
	})
	sendStdin := utils.Go(func() error {
		if in != nil {
			io.Copy(rwc, in)
			utils.Debugf("[hijack] End of stdin")
		}
		if tcpc, ok := rwc.(*net.TCPConn); ok {
			if err := tcpc.CloseWrite(); err != nil {
				utils.Debugf("Couldn't send EOF: %s", err)
			}
		} else if unixc, ok := rwc.(*net.UnixConn); ok {
			if err := unixc.CloseWrite(); err != nil {
				utils.Debugf("Couldn't send EOF: %s", err)
			}
		}
		// Discard errors due to pipe interruption
		return nil
	})
	if err := <-receiveStdout; err != nil {
		utils.Debugf("Error receiveStdout: %s", err)
		return err
	}

	if err := <-sendStdin; err != nil {
		utils.Debugf("Error sendStdin: %s", err)
		return err
	}
	return nil
}
