package rax

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"log"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/beam"
	"github.com/dotcloud/docker/engine"
	"github.com/rackspace/gophercloud"
)

// As the name implies, this function dumbly waits for the desired host status
func doNothing(details *gophercloud.Server) error {
	return nil
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
