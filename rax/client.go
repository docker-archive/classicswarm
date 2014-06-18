package rax

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/dotcloud/docker/utils"
)

// HTTP Client with some syntax sugar
type HttpClient interface {
	Transport() (transport *http.Transport)
	Call(method, path, body string) (*http.Response, error)
	Get(path, data string) (resp *http.Response, err error)
	Delete(path, data string) (resp *http.Response, err error)
	Post(path, data string) (resp *http.Response, err error)
	Put(path, data string) (resp *http.Response, err error)
	Hijack(method, path string, in io.ReadCloser, stdout, stderr io.Writer) (err error)
}

type DockerHttpClient struct {
	Url       *url.URL
	scheme    string
	addr      string
	version   string
	transport *http.Transport
}

// Returns a new docker http client
func newDockerHttpClient(peer, version string) (client HttpClient, err error) {
	targetUrl, err := url.Parse(peer)
	if err != nil {
		return nil, err
	}

	targetUrl.Scheme = "http"

	client = &DockerHttpClient{
		transport: &http.Transport{},
		Url:       targetUrl,
		version:   version,
		scheme:    "http",
	}

	parts := strings.SplitN(peer, "://", 2)
	proto, host := parts[0], parts[1]

	client.Transport().Dial = func(_, _ string) (net.Conn, error) {
		return net.Dial(proto, host)
	}

	return client, nil
}

func (client *DockerHttpClient) Transport() (transport *http.Transport) {
	return client.transport
}

func (client *DockerHttpClient) Call(method, path, body string) (*http.Response, error) {
	targetUrl, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	targetUrl.Host = client.Url.Host
	targetUrl.Scheme = client.Url.Scheme

	req, err := http.NewRequest(method, targetUrl.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: client.Transport(),
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (client *DockerHttpClient) Hijack(method, path string, in io.ReadCloser, stdout, stderr io.Writer) error {
	path = fmt.Sprintf("/%s%s", client.version, path)
	dial, err := client.Transport().Dial("ignored", "ignored")
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

func (client *DockerHttpClient) Get(path, data string) (resp *http.Response, err error) {
	return client.Call("GET", path, data)
}

func (client *DockerHttpClient) Post(path, data string) (resp *http.Response, err error) {
	return client.Call("POST", path, data)
}

func (client *DockerHttpClient) Put(path, data string) (resp *http.Response, err error) {
	return client.Call("PUT", path, data)
}

func (client *DockerHttpClient) Delete(path, data string) (resp *http.Response, err error) {
	return client.Call("DELETE", path, data)
}
