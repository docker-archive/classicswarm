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
	backend.OnSpawn(func(cmd ...string) (beam.Sender, error) {
		if len(cmd) != 1 {
			return nil, fmt.Errorf("dockerclient: spawn takes exactly 1 argument, got %d", len(cmd))
		}
		client := newClient()
		client.scheme = config.Scheme
		client.urlHost = config.URLHost
		client.transport.TLSClientConfig = config.TLSClientConfig
		client.setURL(cmd[0])
		b := &dockerClientBackend{
			client: client,
			Server: beam.NewServer(),
		}
		b.Server.OnAttach(b.attach)
		b.Server.OnStart(b.start)
		b.Server.OnLs(b.ls)
		b.Server.OnSpawn(b.spawn)
		return b.Server, nil
	})
	return backend
}

type dockerClientBackend struct {
	client *client
	*beam.Server
}

func (b *dockerClientBackend) attach(name string, ret beam.Sender) error {
	if name == "" {
		ret.Send(&beam.Message{Verb: beam.Ack, Ret: b.Server})
		<-make(chan struct{})
	} else {
		path := fmt.Sprintf("/containers/%s/json", name)
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
		c := b.newContainer(name)
		ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
	}
	return nil
}

func (b *dockerClientBackend) start() error {
	return nil
}

func (b *dockerClientBackend) ls() ([]string, error) {
	resp, err := b.client.call("GET", "/containers/json", "")
	if err != nil {
		return nil, fmt.Errorf("get: %v", err)
	}
	// FIXME: check for response error
	c := engine.NewTable("Created", 0)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}
	if _, err := c.ReadListFrom(body); err != nil {
		return nil, fmt.Errorf("readlist: %v", err)
	}
	names := []string{}
	for _, env := range c.Data {
		names = append(names, env.GetList("Names")[0][1:])
	}
	return names, nil
}

func (b *dockerClientBackend) spawn(cmd ...string) (beam.Sender, error) {
	if len(cmd) != 1 {
		return nil, fmt.Errorf("dockerclient: spawn takes exactly 1 argument, got %d", len(cmd))
	}
	resp, err := b.client.call("POST", "/containers/create", cmd[0])
	if err != nil {
		return nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("expected status code 201, got %d:\n%s", resp.StatusCode, respBody)
	}
	var respJson struct{ Id string }
	if err = json.Unmarshal(respBody, &respJson); err != nil {
		return nil, err
	}
	return b.newContainer(respJson.Id), nil
}

func (b *dockerClientBackend) newContainer(id string) beam.Sender {
	c := &container{backend: b, id: id}
	instance := beam.NewServer()
	instance.OnAttach(c.attach)
	instance.OnStart(c.start)
	instance.OnStop(c.stop)
	instance.OnGet(c.get)
	return instance
}

type container struct {
	backend *dockerClientBackend
	id      string
}

func (c *container) attach(name string, ret beam.Sender) error {
	if _, err := ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	}

	path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	go beam.EncodeStream(ret, stdoutR, "stdout")
	go beam.EncodeStream(ret, stderrR, "stderr")
	c.backend.client.hijack("POST", path, nil, stdoutW, stderrW)

	return nil
}

func (c *container) start() error {
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
	return nil
}

func (c *container) stop() error {
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
	return nil
}

func (c *container) get() (string, error) {
	path := fmt.Sprintf("/containers/%s/json", c.id)
	resp, err := c.backend.client.call("GET", path, "")
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s", respBody)
	}
	return string(respBody), nil
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
