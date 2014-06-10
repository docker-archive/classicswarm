//
// Copyright (C) 2014 Rackspace Hosting Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backends

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/utils"
	"github.com/rackspace/gophercloud"
)

const (
	DASS_TARGET_PREFIX = "rdas_target_"
)

var (
	nilOptions = gophercloud.AuthOptions{}

	// ErrNoPassword errors occur when the value of the OS_PASSWORD environment variable cannot be determined.
	ErrNoPassword = fmt.Errorf("Environment variable OS_PASSWORD or OS_API_KEY needs to be set.")
)

// On status callback for when waiting on cloud actions
type onStatus func(details *gophercloud.Server) error

// As the name implies, this function dumbly waits for the desired host status
func doNothing(details *gophercloud.Server) error {
	return nil
}

// Shamelessly taken from docker core
func RandomString() string {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(id)
}

// Gets the user's auth options from the openstack env variables
func getAuthOptions(ctx *cli.Context) (string, gophercloud.AuthOptions, error) {
	provider := ctx.String("auth-url")
	username := ctx.String("auth-user")
	apiKey := os.Getenv("OS_API_KEY")
	password := os.Getenv("OS_PASSWORD")
	tenantId := os.Getenv("OS_TENANT_ID")
	tenantName := os.Getenv("OS_TENANT_NAME")

	if provider == "" {
		return "", nilOptions, fmt.Errorf("Please set an auth URL with the switch '--auth-url'")
	}

	if username == "" {
		return "", nilOptions, fmt.Errorf("Please set an auth user with the switch '--auth-user'")
	}

	if password == "" && apiKey == "" {
		return "", nilOptions, ErrNoPassword
	}

	ao := gophercloud.AuthOptions{
		Username:   username,
		Password:   password,
		ApiKey:     apiKey,
		TenantId:   tenantId,
		TenantName: tenantName,
	}

	return provider, ao, nil
}

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

// A hostbound action is an action that is supplied with an active client to a desired cloud host
type hostbound func(msg *beam.Message, client HttpClient) error

// A HostContext represents an active session with a cloud host
type HostContextCache struct {
	contexts map[string]*HostContext
}

func NewHostContextCache() (contextCache *HostContextCache) {
	return &HostContextCache{
		contexts: make(map[string]*HostContext),
	}
}

func (hcc *HostContextCache) Get(id, name string, rax *raxcloud) (context *HostContext, err error) {
	return NewHostContext(id, name, rax)
}

func (hcc *HostContextCache) GetCached(id, name string, rax *raxcloud) (context *HostContext, err error) {
	var found bool
	if context, found = hcc.contexts[id]; !found {
		if context, err = NewHostContext(id, name, rax); err == nil {
			hcc.contexts[id] = context
		}
	}

	return
}

func (hcc *HostContextCache) Close() {
	for _, context := range hcc.contexts {
		context.Close()
	}
}

type HostContext struct {
	id     string
	name   string
	rax    *raxcloud
	tunnel *os.Process
}

func NewHostContext(id, name string, rax *raxcloud) (hc *HostContext, err error) {
	if tunnelProcess, err := rax.openTunnel(id, 8000, rax.tunnelPort); err != nil {
		return nil, fmt.Errorf("Unable to tunnel to host %s(id:%s): %v", name, id, err)
	} else {
		return &HostContext{
			id:     id,
			name:   name,
			rax:    rax,
			tunnel: tunnelProcess,
		}, nil
	}
}

func (hc *HostContext) Close() {
	if hc.tunnel != nil {
		hc.tunnel.Kill()
		if state, err := hc.tunnel.Wait(); err != nil {
			fmt.Printf("Wait result: state:%v, err:%s\n", state, err)
		}

		hc.tunnel = nil
	}
}

func (hc *HostContext) exec(msg *beam.Message, action hostbound) (err error) {
	if hc.tunnel == nil {
		return fmt.Errorf("Tunnel not open to host %s(id:%s)", hc.name, hc.id)
	}

	if client, err := newDockerHttpClient("tcp://localhost:8000", "v1.10"); err == nil {
		return action(msg, client)
	} else {
		return fmt.Errorf("Unable to init http client: %v", err)
	}
}

// A Rackspace impl of the cloud provider interface
type raxcloud struct {
	remoteUser      string
	keyFile         string
	regionOverride  string
	image           string
	flavor          string
	instance        string
	scalingStrategy string
	tunnelPort      int
	hostCtxCache    *HostContextCache

	server *beam.Server
	gopher gophercloud.CloudServersProvider
}

// Returns a pointer to a new raxcloud instance
func RaxCloud() beam.Sender {
	rax := &raxcloud{
		hostCtxCache: NewHostContextCache(),
	}

	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(rax.init))

	return backend
}

// Initializes the plugin by handling job arguments
func (rax *raxcloud) init(msg *beam.Message) (err error) {
	app := cli.NewApp()
	app.Name = "rax"
	app.Usage = "Deploy Docker containers onto your Rackspace Cloud account. Accepts environment variables as listed in: https://github.com/openstack/python-novaclient/#command-line-api"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"auth-user", "", "Sets the username to use when authenticating against Rackspace Identity or Keystone."},
		cli.StringFlag{"auth-url", "https://identity.api.rackspacecloud.com/v2.0/tokens", "Sets the URL to call against when authenticating."},
		cli.StringFlag{"instance", "", "Sets the server instance to target for specific commands such as version."},
		cli.StringFlag{"region", "DFW", "Sets the Rackspace region to target. This overrides any environment variables for region."},
		cli.StringFlag{"key", "/etc/swarmd/rax.id_rsa", "Sets SSH keys file to use when initializing secure connections."},
		cli.StringFlag{"user", "root", "Username to use when initializing secure connections."},
		cli.StringFlag{"image", "", "Image to use when provisioning new hosts."},
		cli.StringFlag{"flavor", "", "Flavor to use when provisioning new hosts."},
		cli.IntFlag{"tun-port", 8000, "Remote port to bind to when creating SSH tunnels on hosts."},
		cli.StringFlag{"auto-scaling", "none", "Flag that when set determines the mode used for auto-scaling."},
	}

	app.Action = func(ctx *cli.Context) {
		err = rax.run(ctx)
	}

	name := string(msg.Verb)
	if len(msg.Args) > 1 {
		app.Run(append([]string{name}, msg.Args...))
	}

	// Failed to init?
	if err != nil {
		return err
	}

	if rax.gopher == nil {
		err = app.Run([]string{name, "help"})

		if err == nil {
			return fmt.Errorf("")
		} else {
			return err
		}
	}

	rax.server = beam.NewServer()
	rax.server.OnAttach(beam.Handler(rax.attach))
	rax.server.OnLs(beam.Handler(rax.ls))
	rax.server.OnStart(beam.Handler(rax.start))
	rax.server.OnStop(beam.Handler(rax.stop))
	rax.server.OnSpawn(beam.Handler(rax.spawn))

	_, err = msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: rax.server})
	return err
}

func (rax *raxcloud) configure(ctx *cli.Context) (err error) {
	if keyFile := ctx.String("key"); keyFile != "" {
		rax.keyFile = keyFile
		rax.tunnelPort = ctx.Int("tun-port")

		if regionOverride := ctx.String("region"); regionOverride != "" {
			rax.regionOverride = regionOverride
		}

		if remoteUser := ctx.String("user"); remoteUser != "" {
			rax.remoteUser = remoteUser
		}

		if instance := ctx.String("instance"); instance != "" {
			rax.instance = instance
		}

		if image := ctx.String("image"); image != "" {
			rax.image = image
		} else {
			return fmt.Errorf("Please set an image ID with --image <id>")
		}

		if flavor := ctx.String("flavor"); flavor != "" {
			rax.flavor = flavor
		} else {
			return fmt.Errorf("Please set a flavor ID with --flavor <id>")
		}

		if scalingStrategy := ctx.String("auto-scaling"); scalingStrategy != "" {
			rax.scalingStrategy = scalingStrategy
		}

		return rax.createGopherCtx(ctx)
	} else {
		return fmt.Errorf("Please set a SSH key with --key <keyfile>")
	}
}

func (rax *raxcloud) start(msg *beam.Message) (err error) {
	msg.Ret.Send(&beam.Message{Verb: beam.Ack})
	return nil
}

func (rax *raxcloud) attach(msg *beam.Message) (err error) {
	var ctx *HostContext

	if msg.Args[0] == "" {
		msg.Ret.Send(&beam.Message{
			Verb: beam.Ack,
			Ret:  rax.server,
		})

		// Copied this from the forwarder example - something more elegant
		// with "sync" would be nice but I'll get to it later.
		for {
			time.Sleep(15 * time.Second)
			(&beam.Object{msg.Ret}).Log("rax: heartbeat")
		}

		return nil
	} else if ctx, err = rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/json", msg.Args[0])

			if resp, err := client.Get(path, ""); err != nil {
				return err
			} else {
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}

				if resp.StatusCode != 200 {
					return fmt.Errorf("%s", respBody)
				}

				c := rax.newContainer(msg.Args[0])
				msg.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c})
			}

			return
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}
}

func (rax *raxcloud) ls(msg *beam.Message) (err error) {
	var ctx *HostContext

	if ctx, err = rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := "/containers/json"

			if resp, err := client.Get(path, ""); err != nil {
				return err
			} else {
				// FIXME: check for response error
				createdTable := engine.NewTable("Created", 0)
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("read body: %v", err)
				}

				if _, err := createdTable.ReadListFrom(body); err != nil {
					return fmt.Errorf("readlist: %v", err)
				}

				names := []string{}
				for _, env := range createdTable.Data {
					names = append(names, env.GetList("Names")[0][1:])
				}

				if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: names}); err != nil {
					return fmt.Errorf("send response: %v", err)
				}
			}

			return
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}
}

func (rax *raxcloud) spawn(msg *beam.Message) (err error) {
	var ctx *HostContext

	if ctx, err = rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := "/containers/create"

			if resp, err := client.Post(path, msg.Args[0]); err != nil {
				return err
			} else {
				var container struct {
					Id string
				}

				if body, err := ioutil.ReadAll(resp.Body); err != nil {
					return fmt.Errorf("Failed to copy response body: %v", err)
				} else if err := json.Unmarshal(body, &container); err != nil {
					return fmt.Errorf("Failed to read container info from body: %v\nBody: %s", err, string(body))
				} else {
					fmt.Printf("%s\n", container.Id)
				}
			}

			return
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}
}

func (rax *raxcloud) run(ctx *cli.Context) (err error) {
	if err = rax.configure(ctx); err == nil {
		var name string

		if _, name, err = rax.findTargetHost(); err == nil {
			if name == "" {
				return rax.createHost(DASS_TARGET_PREFIX + RandomString()[:12])
			}
		}
	}

	return
}

func (rax *raxcloud) stop(msg *beam.Message) (err error) {
	rax.hostCtxCache.Close()

	return nil
}

func (rax *raxcloud) getHostContext() (hc *HostContext, err error) {
	if id, name, err := rax.findTargetHost(); err != nil {
		return nil, err
	} else {
		return rax.hostCtxCache.GetCached(id, name, rax)
	}
}

func (rax *raxcloud) addressFor(id string) (address string, err error) {
	var server *gophercloud.Server

	if server, err = rax.gopher.ServerById(id); err == nil {
		address = server.AccessIPv4
	}

	return
}

func (rax *raxcloud) createHost(name string) (err error) {
	if image, err := rax.gopher.ImageById(rax.image); image == nil || err != nil {
		return fmt.Errorf("Failed to find image: %s. Are you sure your image ID is correct and is in the region: %s?", rax.image, rax.region())
	}

	createServerInfo := gophercloud.NewServer{
		Name:      name,
		ImageRef:  rax.image,
		FlavorRef: rax.flavor,
	}

	if newServer, err := rax.gopher.CreateServer(createServerInfo); err == nil {
		log.Printf("Waiting for host build to complete: %s\n", name)
		return rax.waitForStatusById(newServer.Id, "ACTIVE", doNothing)
	} else {
		return err
	}
}

func (rax *raxcloud) openTunnel(id string, localPort, remotePort int) (*os.Process, error) {
	ip, err := rax.addressFor(id)

	if err != nil {
		return nil, err
	}

	options := []string{
		"-o", "PasswordAuthentication=no",
		"-o", "LogLevel=quiet",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "CheckHostIP=no",
		"-o", "StrictHostKeyChecking=no",
		"-i", rax.keyFile,
		"-A",
		"-p", "22",
		fmt.Sprintf("%s@%s", rax.remoteUser, ip),
		"-N",
		"-f",
		"-L", fmt.Sprintf("%d:localhost:%d", localPort, remotePort),
	}

	cmd := exec.Command("ssh", options...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func (rax *raxcloud) region() (region string) {
	region = os.Getenv("OS_REGION_NAME")

	if rax.regionOverride != "" {
		region = rax.regionOverride
	}

	return region
}

func (rax *raxcloud) createGopherCtx(cli *cli.Context) (err error) {
	var (
		provider    string
		authOptions gophercloud.AuthOptions
		apiCriteria gophercloud.ApiCriteria
		access      *gophercloud.Access
		csp         gophercloud.CloudServersProvider
	)

	// Create our auth options set by the user's environment
	provider, authOptions, err = getAuthOptions(cli)
	if err != nil {
		return err
	}

	// Set our API criteria
	apiCriteria, err = gophercloud.PopulateApi("rackspace-us")
	if err != nil {
		apiCriteria, err = gophercloud.PopulateApi("rackspace-uk")

		if err != nil {
			return fmt.Errorf("Unable to populate Rackspace API: %v", err)
		}
	}

	// Set the region
	apiCriteria.Type = "compute"
	apiCriteria.Region = rax.region()

	if apiCriteria.Region == "" {
		return fmt.Errorf("No region set. Please set the OS_REGION_NAME environment variable or use the --region flag.")
	}

	// Attempt to authenticate
	access, err = gophercloud.Authenticate(provider, authOptions)
	if err != nil {
		fmt.Printf("ERROR: %v\n\n\n", err)
		return err
	}

	// Get a CSP ref!
	csp, err = gophercloud.ServersApi(access, apiCriteria)
	if err != nil {
		return err
	}

	rax.gopher = csp
	return nil
}

func (rax *raxcloud) findDockerHosts() (ids map[string]string, err error) {
	var servers []gophercloud.Server

	if servers, err = rax.gopher.ListServersLinksOnly(); err == nil {
		ids = make(map[string]string)

		for _, server := range servers {
			if strings.HasPrefix(server.Name, DASS_TARGET_PREFIX) {
				ids[server.Id] = server.Name
			}
		}
	}

	return
}

func (rax *raxcloud) findDeploymentHosts() (ids map[string]string, err error) {
	var servers map[string]string

	if servers, err = rax.findDockerHosts(); err == nil {
		ids = make(map[string]string)

		for id, name := range servers {
			var serverDetails *gophercloud.Server

			if serverDetails, err = rax.gopher.ServerById(id); err == nil {
				if serverDetails.Status == "ACTIVE" {
					ids[id] = name
				}
			} else {
				break
			}
		}
	}

	return
}

func (rax *raxcloud) findTargetHost() (id, name string, err error) {
	var deploymentHosts map[string]string

	if deploymentHosts, err = rax.findDeploymentHosts(); err == nil {
		for id, name = range deploymentHosts {
			break
		}
	}

	return
}

func (rax *raxcloud) serverIdByName(name string) (string, error) {
	if servers, err := rax.gopher.ListServersLinksOnly(); err == nil {
		for _, server := range servers {
			if server.Name == name {
				return server.Id, nil
			}
		}

		return "", fmt.Errorf("Unable to locate server by name: %s", name)
	} else {
		return "", err
	}
}

func (rax *raxcloud) waitForStatusByName(name, status string, action onStatus) error {
	id, err := rax.serverIdByName(name)

	for err == nil {
		return rax.waitForStatusById(id, status, action)
	}

	return err
}

func (rax *raxcloud) waitForStatusById(id, status string, action onStatus) (err error) {
	var details *gophercloud.Server
	for err == nil {
		if details, err = rax.gopher.ServerById(id); err == nil {
			log.Printf("Status: %s\n", details.Status)

			if details.Status == status {
				action(details)
				break
			}

			time.Sleep(10 * time.Second)
		}
	}
	return
}

func (rax *raxcloud) newContainer(id string) beam.Sender {
	container := &rax_container{
		rax: rax,
		id:  id,
	}

	instance := beam.NewServer()
	instance.OnAttach(beam.Handler(container.attach))
	instance.OnStart(beam.Handler(container.start))
	instance.OnStop(beam.Handler(container.stop))
	instance.OnGet(beam.Handler(container.get))
	return instance
}

type rax_container struct {
	rax *raxcloud
	id  string
}

func (c *rax_container) attach(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()
			go copyOutput(msg.Ret, stdoutR, "stdout")
			go copyOutput(msg.Ret, stderrR, "stderr")

			client.Hijack("POST", path, nil, stdoutW, stderrW)

			return
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) start(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/start", c.id)
			resp, err := client.Post(path, "{}")
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

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) stop(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/start", c.id)
			resp, err := client.Post(path, "{}")
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

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) get(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/json", c.id)
			resp, err := client.Get(path, "")
			if err != nil {
				return err
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status code 200, got %d:\n%s", resp.StatusCode, respBody)
			}

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{string(respBody)}}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}
