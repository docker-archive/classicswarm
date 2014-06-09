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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/runconfig"
	"github.com/rackspace/gophercloud"
)

const (
	DASS_TARGET_PREFIX = "rdas_target_"
)

var (
	nilOptions = gophercloud.AuthOptions{}

	// ErrNoAuthUrl errors occur when the value of the OS_AUTH_URL environment variable cannot be determined.
	ErrNoAuthUrl = fmt.Errorf("Environment variable OS_AUTH_URL needs to be set.")

	// ErrNoUsername errors occur when the value of the OS_USERNAME environment variable cannot be determined.
	ErrNoUsername = fmt.Errorf("Environment variable OS_USERNAME needs to be set.")

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
func getAuthOptions() (string, gophercloud.AuthOptions, error) {
	provider := os.Getenv("OS_AUTH_URL")
	username := os.Getenv("OS_USERNAME")
	apiKey := os.Getenv("OS_API_KEY")
	password := os.Getenv("OS_PASSWORD")
	tenantId := os.Getenv("OS_TENANT_ID")
	tenantName := os.Getenv("OS_TENANT_NAME")

	if provider == "" {
		return "", nilOptions, ErrNoAuthUrl
	}

	if username == "" {
		return "", nilOptions, ErrNoUsername
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
	Call(method, path, body string) (*http.Response, error)
	Get(path, data string) (resp *http.Response, err error)
	Delete(path, data string) (resp *http.Response, err error)
	Post(path, data string) (resp *http.Response, err error)
	Put(path, data string) (resp *http.Response, err error)
}

type DockerHttpClient struct {
	Url           *url.URL
	proto         string
	addr          string
	dockerVersion string
}

// Returns a new docker http client
func newDockerHttpClient(peer, dockerVersion string) (client HttpClient, err error) {
	targetUrl, err := url.Parse(peer)
	if err != nil {
		return nil, err
	}

	targetUrl.Scheme = "http"

	return &DockerHttpClient{
		Url:           targetUrl,
		dockerVersion: dockerVersion,
	}, nil
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
type hostbound func(client HttpClient) engine.Status

// A HostContext represents an active session with a cloud host
type HostContextCache struct {
	contexts map[string]*HostContext
}

func NewHostContextCache() (contextCache *HostContextCache) {
	return &HostContextCache{
		contexts: make(map[string]*HostContext),
	}
}

func (hcc *HostContextCache) Get(id, name string, rax *RaxCloud) (context *HostContext, err error) {
	return NewHostContext(id, name, rax)
}

func (hcc *HostContextCache) GetCached(id, name string, rax *RaxCloud) (context *HostContext, err error) {
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
	rax    *RaxCloud
	tunnel *os.Process
}

func NewHostContext(id, name string, rax *RaxCloud) (hc *HostContext, err error) {
	if tunnelProcess, err := rax.openTunnel(id, 8000, 8000); err != nil {
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

func (hc *HostContext) exec(job *engine.Job, action hostbound) (status engine.Status) {
	if hc.tunnel == nil {
		return job.Errorf("Tunnel not open to host %s(id:%s)", hc.name, hc.id)
	}

	if client, err := newDockerHttpClient("tcp://localhost:8000", "v1.10"); err == nil {
		return action(client)
	} else {
		return job.Errorf("Unable to init http client: %v", err)
	}
}

// A Rackspace impl of the cloud provider interface
type RaxCloud struct {
	remoteUser      string
	keyFile         string
	regionOverride  string
	image           string
	flavor          string
	instance        string
	scalingStrategy string
	hostCtxCache    *HostContextCache

	gopher gophercloud.CloudServersProvider
}

// Returns a pointer to a new RaxCloud instance
func RaxCloudBackend() *RaxCloud {
	return &RaxCloud{
		hostCtxCache: NewHostContextCache(),
	}
}

// Installs the RaxCloud backend into an engine
func (rax *RaxCloud) Install(eng *engine.Engine) (err error) {
	eng.Register("rax", func(job *engine.Job) (status engine.Status) {
		return rax.init(job)
	})

	return nil
}

// Initializes the plugin by handling job arguments
func (rax *RaxCloud) init(job *engine.Job) (status engine.Status) {
	app := cli.NewApp()
	app.Name = "rax"
	app.Usage = "Deploy Docker containers onto your Rackspace Cloud account. Accepts environment variables as listed in: https://github.com/openstack/python-novaclient/#command-line-api"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"instance", "", "Sets the server instance to target for specific commands such as version."},
		cli.StringFlag{"region", "DFW", "Sets the Rackspace region to target. This overrides any environment variables for region."},
		cli.StringFlag{"key", "/etc/swarmd/rax.id_rsa", "Sets SSH keys file to use when initializing secure connections."},
		cli.StringFlag{"user", "root", "Username to use when initializing secure connections."},
		cli.StringFlag{"image", "", "Image to use when provisioning new hosts."},
		cli.StringFlag{"flavor", "", "Flavor to use when provisioning new hosts."},
		cli.StringFlag{"auto-scaling", "none", "Flag that when set determines the mode used for auto-scaling."},
	}

	var err error
	app.Action = func(ctx *cli.Context) {
		err = rax.run(ctx, job.Eng)
	}

	if len(job.Args) > 1 {
		app.Run(append([]string{job.Name}, job.Args...))
	}

	// Failed to init?
	if err != nil || rax.gopher == nil {
		app.Run([]string{"rax", "help"})

		if err == nil {
			return engine.StatusErr
		} else {
			return job.Error(err)
		}
	}

	return engine.StatusOK
}

func (rax *RaxCloud) configure(ctx *cli.Context) (err error) {
	if keyFile := ctx.String("key"); keyFile != "" {
		rax.keyFile = keyFile

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

		return rax.createGopherCtx()
	} else {
		return fmt.Errorf("Please set a SSH key with --key <keyfile>")
	}
}

func (rax *RaxCloud) run(ctx *cli.Context, eng *engine.Engine) (err error) {
	if err = rax.configure(ctx); err == nil {
		if _, name, err := rax.findTargetHost(); err == nil {
			if name == "" {
				rax.createHost(DASS_TARGET_PREFIX + RandomString()[:12])
			}
		} else {
			return err
		}

		eng.Register("create", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := "/containers/create"
					config := runconfig.ContainerConfigFromJob(job)

					if data, err := json.Marshal(config); err != nil {
						return job.Errorf("marshaling failure : %v", err)
					} else if resp, err := client.Post(path, string(data)); err != nil {
						return job.Error(err)
					} else {
						var container struct {
							Id       string
							Warnings []string
						}

						if body, err := ioutil.ReadAll(resp.Body); err != nil {
							return job.Errorf("Failed to copy response body: %v", err)
						} else if err := json.Unmarshal([]byte(body), &container); err != nil {
							return job.Errorf("Failed to read container info from body: %v", err)
						} else {
							job.Printf("%s\n", container.Id)
						}
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("container_delete", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s?force=%s", job.Args[0], url.QueryEscape(job.Getenv("forceRemove")))

					if _, err := client.Delete(path, ""); err != nil {
						return job.Error(err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("containers", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/json?all=%s&limit=%s",
						url.QueryEscape(job.Getenv("all")), url.QueryEscape(job.Getenv("limit")))

					if resp, err := client.Get(path, ""); err != nil {
						return job.Error(err)
					} else {
						created := engine.NewTable("Created", 0)

						if body, err := ioutil.ReadAll(resp.Body); err != nil {
							return job.Errorf("Failed to copy response body: %v", err)
						} else if created.ReadListFrom(body); err != nil {
							return job.Errorf("Failed to read list from body: %v", err)
						} else {
							created.WriteListTo(job.Stdout)
						}
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("version", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := "/version"

					if resp, err := client.Get(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					} else if _, err := io.Copy(job.Stdout, resp.Body); err != nil {
						return job.Errorf("Failed to copy response body: %v", err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("start", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/start", job.Args[0])
					config := runconfig.ContainerConfigFromJob(job)

					if data, err := json.Marshal(config); err != nil {
						return job.Errorf("marshaling failure : %v", err)
					} else if _, err := client.Post(path, string(data)); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("stop", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/stop?t=%s", job.Args[0], url.QueryEscape(job.Getenv("t")))

					if _, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("kill", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/kill?signal=%s", job.Args[0], job.Args[1])

					if _, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("restart", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/restart?t=%s", url.QueryEscape(job.Getenv("t")))

					if _, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("inspect", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/json", job.Args[0])

					if resp, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					} else if _, err := io.Copy(job.Stdout, resp.Body); err != nil {
						return job.Errorf("Failed to copy response body: %v", err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("attach", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/attach?stream=%s&stdout=%s&stderr=%s", job.Args[0],
						url.QueryEscape(job.Getenv("stream")),
						url.QueryEscape(job.Getenv("stdout")),
						url.QueryEscape(job.Getenv("stderr")))

					if resp, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					} else if _, err := io.Copy(job.Stdout, resp.Body); err != nil {
						return job.Errorf("Failed to copy response body: %v", err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("pull", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/images/create?fromImage=%s&tag=%s", job.Args[0], url.QueryEscape(job.Getenv("tag")))

					if resp, err := client.Post(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					} else if _, err := io.Copy(job.Stdout, resp.Body); err != nil {
						return job.Errorf("Failed to copy response body: %v", err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.Register("logs", func(job *engine.Job) (status engine.Status) {
			if ctx, err := rax.getHostContext(); err == nil {
				defer ctx.Close()

				return ctx.exec(job, func(client HttpClient) (status engine.Status) {
					path := fmt.Sprintf("/containers/%s/logs?stdout=%s&stderr=%s", job.Args[0], url.QueryEscape(job.Getenv("stdout")), url.QueryEscape(job.Getenv("stderr")))

					if resp, err := client.Get(path, ""); err != nil {
						return job.Errorf("Failed call %s(path:%s): %v", job.Name, path, err)
					} else if _, err := io.Copy(job.Stdout, resp.Body); err != nil {
						return job.Errorf("Failed to copy response body: %v", err)
					}

					return engine.StatusOK
				})
			} else {
				return job.Errorf("Failed to create host context: %v", err)
			}
		})

		eng.RegisterCatchall(func(job *engine.Job) engine.Status {
			log.Printf("[UNIMPLEMENTED] %s %#v %#v %#v", job.Name, *job, job.Env(), job.Args)
			return engine.StatusOK
		})
	}

	return err
}

func (rax *RaxCloud) Close() {
	rax.hostCtxCache.Close()
}

func (rax *RaxCloud) getHostContext() (hc *HostContext, err error) {
	if id, name, err := rax.findTargetHost(); err != nil {
		return nil, err
	} else {
		return rax.hostCtxCache.Get(id, name, rax)
	}
}

func (rax *RaxCloud) addressFor(id string) (address string, err error) {
	var server *gophercloud.Server

	if server, err = rax.gopher.ServerById(id); err == nil {
		address = server.AccessIPv4
	}

	return
}

func (rax *RaxCloud) createHost(name string) (err error) {
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

func (rax *RaxCloud) openTunnel(id string, localPort, remotePort int) (*os.Process, error) {
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

func (rax *RaxCloud) createGopherCtx() (err error) {
	var (
		provider    string
		authOptions gophercloud.AuthOptions
		apiCriteria gophercloud.ApiCriteria
		access      *gophercloud.Access
		csp         gophercloud.CloudServersProvider
	)

	// Create our auth options set by the user's environment
	provider, authOptions, err = getAuthOptions()
	if err != nil {
		return err
	}

	// Set our API criteria
	apiCriteria, err = gophercloud.PopulateApi("rackspace-us")
	if err != nil {
		return err
	}

	// Set the region
	apiCriteria.Type = "compute"
	apiCriteria.Region = os.Getenv("OS_REGION_NAME")

	if rax.regionOverride != "" {
		log.Printf("Overriding region set in env with %s", rax.regionOverride)
		apiCriteria.Region = rax.regionOverride
	}

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

func (rax *RaxCloud) findDockerHosts() (ids map[string]string, err error) {
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

func (rax *RaxCloud) findDeploymentHosts() (ids map[string]string, err error) {
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

func (rax *RaxCloud) findTargetHost() (id, name string, err error) {
	var deploymentHosts map[string]string

	if deploymentHosts, err = rax.findDeploymentHosts(); err == nil {
		for id, name = range deploymentHosts {
			break
		}
	}

	return
}

func (rax *RaxCloud) serverIdByName(name string) (string, error) {
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

func (rax *RaxCloud) waitForStatusByName(name, status string, action onStatus) error {
	id, err := rax.serverIdByName(name)

	for err == nil {
		return rax.waitForStatusById(id, status, action)
	}

	return err
}

func (rax *RaxCloud) waitForStatusById(id, status string, action onStatus) (err error) {
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
