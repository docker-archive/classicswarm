package backends

import (
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
	"time"
)

func Forward() beam.Sender {
	return ForwardWithClient(newClient())
}

func ForwardWithClient(client *client) beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		if len(ctx.Args) != 1 {
			return fmt.Errorf("forward: spawn takes exactly 1 argument, got %d", len(ctx.Args))
		}
		client.setURL(ctx.Args[0])
		f := &forwarder{
			client: client,
			Server: beam.NewServer(),
		}
		f.Server.OnAttach(beam.Handler(f.attach))
		f.Server.OnStart(beam.Handler(f.start))
		f.Server.OnLs(beam.Handler(f.ls))
		f.Server.OnSpawn(beam.Handler(f.spawn))
		_, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: f.Server})
		return err
	}))
	return backend
}

type forwarder struct {
	client *client
	*beam.Server
}

func (f *forwarder) attach(ctx *beam.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: f.Server})
		for {
			time.Sleep(1 * time.Second)
			(&beam.Object{ctx.Ret}).Log("forward: heartbeat")
		}
	} else {
		c := f.newContainer(ctx.Args[0])
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
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

func (f *forwarder) spawn(ctx *beam.Message) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("forward: spawn takes exactly 1 argument, got %d", len(ctx.Args))
	}
	resp, err := f.client.call("POST", "/containers/create", ctx.Args[0])
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
	c := f.newContainer(respJson.Id)
	if _, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c}); err != nil {
		return err
	}
	return nil
}

func (f *forwarder) newContainer(id string) beam.Sender {
	c := &container{forwarder: f, id: id}
	instance := beam.NewServer()
	instance.OnAttach(beam.Handler(c.attach))
	instance.OnStart(beam.Handler(c.start))
	instance.OnStop(beam.Handler(c.stop))
	instance.OnGet(beam.Handler(c.get))
	return instance
}

type container struct {
	forwarder *forwarder
	id        string
}

func (c *container) attach(ctx *beam.Message) error {
	if _, err := ctx.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	}

	path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	go copyOutput(ctx.Ret, stdoutR, "stdout")
	go copyOutput(ctx.Ret, stderrR, "stderr")
	c.forwarder.client.hijack("POST", path, nil, stdoutW, stderrW)

	return nil
}

func copyOutput(sender beam.Sender, reader io.Reader, tag string) {
	chunk := make([]byte, 4096)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			sender.Send(&beam.Message{Verb: beam.Log, Args: []string{tag, string(chunk[0:n])}})
		}
		if err != nil {
			message := fmt.Sprintf("Error reading from stream: %v", err)
			sender.Send(&beam.Message{Verb: beam.Error, Args: []string{message}})
			break
		}
	}
}

func (c *container) start(ctx *beam.Message) error {
	path := fmt.Sprintf("/containers/%s/start", c.id)
	resp, err := c.forwarder.client.call("POST", path, "{}")
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
	resp, err := c.forwarder.client.call("POST", path, "{}")
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
	resp, err := c.forwarder.client.call("GET", path, "")
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
