package backends

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm"
	"github.com/docker/libswarm/utils"
	"github.com/docker/docker/engine"
	dockerutils "github.com/docker/docker/utils"
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

func DockerClient() libswarm.Sender {
	return DockerClientWithConfig(&DockerClientConfig{
		Scheme:  "http",
		URLHost: "dummy.host",
	})
}

func DockerClientWithConfig(config *DockerClientConfig) libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnSpawn(func(cmd ...string) (libswarm.Sender, error) {
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
			Server: libswarm.NewServer(),
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
	*libswarm.Server
}

func (b *dockerClientBackend) attach(name string, ret libswarm.Sender) error {
	if name == "" {
		ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: b.Server})
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
		ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: c})
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

func (b *dockerClientBackend) createContainer(cmd ...string) (libswarm.Sender, error) {
	if len(cmd) != 1 {
		return nil, fmt.Errorf("dockerclient: spawn takes exactly 1 argument, got %d", len(cmd))
	}

	param := cmd[0]
	containerValues := url.Values{}

	var postParam map[string]interface{}
	if err := json.Unmarshal([]byte(param), &postParam); err != nil {
		return nil, err
	}
	if name, ok := postParam["name"]; ok {
		containerValues.Set("name", name.(string))
		delete(postParam, "name")
		tmp, err := json.Marshal(postParam)
		if err != nil {
			return nil, err
		}
		param = string(tmp)
	}

	resp, err := b.client.call("POST", "/containers/create?"+containerValues.Encode(), param)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("expected status code 201, got %d:\n%s", resp.StatusCode, respBody)
	}

	var respJson struct{ Id string }
	if err = json.Unmarshal(respBody, &respJson); err != nil {
		return nil, err
	}
	return b.newContainer(respJson.Id), nil
}

func (b *dockerClientBackend) createImage(cmd ...string) error {
	if len(cmd) != 1 {
		return fmt.Errorf("dockerclient: image name is the only expected argument, got %d", len(cmd))
	}

	var respJson struct{ Image string }
	if err := json.Unmarshal([]byte(cmd[0]), &respJson); err != nil {
		return err
	}

	imageTag := strings.Split(respJson.Image, ":")

	var tag = "latest"
	image := imageTag[0]

	if len(imageTag) > 1 {
		tag = imageTag[1]
	}

	url := fmt.Sprintf("/images/create?fromImage=%s&tag=%s", image, tag)

	resp, err := b.client.call("POST", url, "")
	if err != nil {
		return err
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (b *dockerClientBackend) spawn(cmd ...string) (libswarm.Sender, error) {
	sender, err := b.createContainer(cmd...)

	if err != nil {
		err = b.createImage(cmd...)
		if err != nil {
			return sender, err
		}
		sender, err = b.createContainer(cmd...)
		if err != nil {
			return sender, err
		}
	}

	return sender, nil
}

func (b *dockerClientBackend) newContainer(id string) libswarm.Sender {
	c := &container{backend: b, id: id}
	instance := libswarm.NewServer()
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

func (c *container) attach(name string, ret libswarm.Sender) error {
	if _, err := ret.Send(&libswarm.Message{Verb: libswarm.Ack}); err != nil {
		return err
	}

	path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	go utils.EncodeStream(ret, stdoutR, "stdout")
	go utils.EncodeStream(ret, stderrR, "stderr")
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

	receiveStdout := dockerutils.Go(func() (err error) {
		defer func() {
			if in != nil {
				in.Close()
			}
		}()
		_, err = dockerutils.StdCopy(stdout, stderr, br)
		dockerutils.Debugf("[hijack] End of stdout")
		return err
	})
	sendStdin := dockerutils.Go(func() error {
		if in != nil {
			io.Copy(rwc, in)
			dockerutils.Debugf("[hijack] End of stdin")
		}
		if tcpc, ok := rwc.(*net.TCPConn); ok {
			if err := tcpc.CloseWrite(); err != nil {
				dockerutils.Debugf("Couldn't send EOF: %s", err)
			}
		} else if unixc, ok := rwc.(*net.UnixConn); ok {
			if err := unixc.CloseWrite(); err != nil {
				dockerutils.Debugf("Couldn't send EOF: %s", err)
			}
		}
		// Discard errors due to pipe interruption
		return nil
	})
	if err := <-receiveStdout; err != nil {
		dockerutils.Debugf("Error receiveStdout: %s", err)
		return err
	}

	if err := <-sendStdin; err != nil {
		dockerutils.Debugf("Error sendStdin: %s", err)
		return err
	}
	return nil
}
