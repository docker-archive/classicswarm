package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
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
		client, err := newClient(ctx.Args[0], "v1.11")
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		f := &forwarder{
			client: client,
			Server: beam.NewServer(),
		}
		f.Server.OnAttach(beam.Handler(f.attach))
		f.Server.OnStart(beam.Handler(f.start))
		f.Server.OnLs(beam.Handler(f.ls))
		f.Server.OnSpawn(beam.Handler(f.spawn))
		_, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: f.Server})
		return err
	}))
	return backend
}

type forwarder struct {
	client *client
	*beam.Server
}

func (f *forwarder) attach(ctx *beam.Message) error {
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: f.Server})
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *client) hijack(method, path string, in io.ReadCloser, stdout, stderr io.Writer) error {
	path = fmt.Sprintf("/%s%s", c.version, path)
	dial, err := net.Dial("tcp", c.URL.Host)
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
		log.Println("[hijack] End of stdout")
		return err
	})
	sendStdin := utils.Go(func() error {
		if in != nil {
			io.Copy(rwc, in)
			log.Println("[hijack] End of stdin")
		}
		if tcpc, ok := rwc.(*net.TCPConn); ok {
			if err := tcpc.CloseWrite(); err != nil {
				log.Printf("Couldn't send EOF: %s\n", err)
			}
		} else if unixc, ok := rwc.(*net.UnixConn); ok {
			if err := unixc.CloseWrite(); err != nil {
				log.Printf("Couldn't send EOF: %s\n", err)
			}
		}
		// Discard errors due to pipe interruption
		return nil
	})
	if err := <-receiveStdout; err != nil {
		log.Printf("Error receiveStdout: %s\n", err)
		return err
	}

	if err := <-sendStdin; err != nil {
		log.Printf("Error sendStdin: %s\n", err)
		return err
	}
	return nil
}
