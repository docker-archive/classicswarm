package backends

import (
	"errors"
	"fmt"
	"github.com/docker/libswarm/beam"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"syscall"
	"time"

	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/ec2"
)

type ec2Config struct {
	securityGroup string
	instanceType  string
	zone          string
	ami           string
	tag           string
	sshUser       string
	sshKey        string
	sshLocalPort  string
	sshRemotePort string
	keypair       string
	region        aws.Region
}

type ec2Client struct {
	config         *ec2Config
	ec2Conn        *ec2.EC2
	Server         *beam.Server
	instance       *ec2.Instance
	sshTunnel      *os.Process
	dockerInstance *beam.Object
}

func (c *ec2Client) get(ctx *beam.Message) error {
	output, err := c.dockerInstance.Get()
	if err != nil {
		return err
	}
	ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{output}})
	return nil
}

func (c *ec2Client) start(ctx *beam.Message) error {
	if instance, err := c.findInstance(); err != nil {
		return err
	} else if instance != nil {
		fmt.Printf("Found existing instance: %s\n", instance.InstanceId)
		c.instance = instance
	} else {
		if err := c.startInstance(); err != nil {
			return err
		}

		if err := c.tagtInstance(); err != nil {
			return err
		}
	}

	c.initDockerClientInstance(c.instance)
	c.waitForSsh()
	c.startSshTunnel()
	c.waitForDockerDaemon()
	fmt.Printf("ec2 service up and running: region: %s zone: %s\n",
		c.config.region.Name, c.config.zone)
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})

	return nil
}

func (c *ec2Client) spawn(ctx *beam.Message) error {
	out, err := c.dockerInstance.Spawn(ctx.Args...)
	if err != nil {
		return err
	}
	ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: out})
	return nil
}

func (c *ec2Client) ls(ctx *beam.Message) error {
	output, err := c.dockerInstance.Ls()
	if err != nil {
		return err
	}
	ctx.Ret.Send(&beam.Message{Verb: beam.Set, Args: output})
	return nil
}

func (c *ec2Client) stop(ctx *beam.Message) error {
	c.dockerInstance.Stop()
	return nil
}

func (c *ec2Client) attach(ctx *beam.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.Server})
		<-make(chan struct{})
	} else {
		_, out, err := c.dockerInstance.Attach(ctx.Args[0])
		if err != nil {
			return err
		}
		ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: out})
	}

	return nil
}

func defaultSshKeyPath() string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	return path.Join(dir, ".ssh", "id_rsa")
}

func defaultConfigValues() (config *ec2Config) {
	config = new(ec2Config)
	config.region = aws.USEast
	config.ami = "ami-7c807d14"
	config.instanceType = "t1.micro"
	config.zone = "us-east-1a"
	config.sshUser = "ec2-user"
	config.tag = "docker-ec2-swarm"
	config.sshLocalPort = "4910"
	config.sshRemotePort = "4243"
	config.sshKey = defaultSshKeyPath()
	return config
}

func newConfig(args []string) (config *ec2Config, err error) {
	var optValPair []string
	var opt, val string

	config = defaultConfigValues()

	for _, value := range args {
		optValPair = strings.Split(value, "=")
		opt, val = optValPair[0], optValPair[1]

		switch opt {
		case "--region":
			config.region = convertToRegion(val)
		case "--zone":
			config.zone = val
		case "--tag":
			config.tag = val
		case "--ami":
			config.ami = val
		case "--keypair":
			config.keypair = val
		case "--security_group":
			config.securityGroup = val
		case "--instance_type":
			config.instanceType = val
		case "--ssh_user":
			config.sshUser = val
		case "--ssh_key":
			config.sshKey = val
		default:
			fmt.Printf("Unrecognizable option: %s value: %s", opt, val)
		}
	}
	return config, nil
}

func convertToRegion(input string) aws.Region {
	switch input {
	case "us-east-1":
		return aws.USEast
	case "us-west-1":
		return aws.USWest
	case "us-west-2":
		return aws.USWest2
	case "eu-west-1":
		return aws.EUWest
	case "sa-east-1":
		return aws.SAEast
	case "ap-northeast-1":
		return aws.APNortheast
	case "ap-southeast-1":
		return aws.APSoutheast
	case "ap-southeast-2":
		return aws.APSoutheast2
	default:
		fmt.Println("Unrecognizable region, default to: us-east-1")
		return aws.USEast
	}
}

func awsInit(config *ec2Config) (ec2Conn *ec2.EC2, err error) {
	auth, err := aws.EnvAuth()

	if err != nil {
		return nil, err
	}

	return ec2.New(auth, config.region), nil
}

func (c *ec2Client) findInstance() (instance *ec2.Instance, err error) {
	filter := ec2.NewFilter()
	filter.Add("tag:Name", c.config.tag)
	resp, err := c.ec2Conn.Instances([]string{}, filter)

	if err != nil {
		return nil, err
	} else {
		if resp.Reservations == nil {
			return nil, nil
		}

		instance := resp.Reservations[0].Instances[0]

		if instance.State.Name == "running" || instance.State.Name == "pending" {
			return &instance, nil
		}

		return nil, nil
	}
}

func (c *ec2Client) tagtInstance() error {
	ec2Tags := []ec2.Tag{ec2.Tag{"Name", c.config.tag}}
	if _, err := c.ec2Conn.CreateTags([]string{c.instance.InstanceId}, ec2Tags); err != nil {
		return err
	}
	return nil
}

func (c *ec2Client) startInstance() error {
	options := ec2.RunInstances{
		ImageId:      c.config.ami,
		InstanceType: c.config.instanceType,
		KeyName:      c.config.keypair,
		AvailZone:    c.config.zone,
		// TODO: allow more than one sg in the future
		SecurityGroups: []ec2.SecurityGroup{ec2.SecurityGroup{Name: c.config.securityGroup}},
		UserData:       []byte(userdata),
	}

	resp, err := c.ec2Conn.RunInstances(&options)
	if err != nil {
		return err
	}

	// TODO (aaron): this really could be multiple instances, not just 1
	i := resp.Instances[0]

	for i.State.Name != "running" {
		time.Sleep(3 * time.Second)
		fmt.Printf("Waiting for instance to come up.  Current State: %s\n",
			i.State.Name)

		resp, err := c.ec2Conn.Instances([]string{i.InstanceId}, ec2.NewFilter())

		if err != nil {
			return err
		}

		i = resp.Reservations[0].Instances[0]
	}

	c.instance = &i

	fmt.Printf("Instance up and running - id: %s\n", i.InstanceId)
	return nil
}

func (c *ec2Client) initDockerClientInstance(instance *ec2.Instance) error {
	dockerClient := DockerClientWithConfig(&DockerClientConfig{
		Scheme:  "http",
		URLHost: "localhost",
	})

	dockerBackend := beam.Obj(dockerClient)
	url := fmt.Sprintf("tcp://localhost:%s", c.config.sshLocalPort)
	dockerInstance, err := dockerBackend.Spawn(url)
	c.dockerInstance = dockerInstance

	if err != nil {
		return err
	}
	return nil
}

func (c *ec2Client) waitForDockerDaemon() {
	fmt.Println("waiting for docker daemon on remote machine to be available.")
	for {
		resp, _ := http.Get("http://localhost:" + c.config.sshLocalPort)
		// wait for a response.  any response to know docker daemon is up
		if resp != nil {
			break
		}
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	fmt.Println()
}

func (c *ec2Client) waitForSsh() {
	fmt.Println("waiting for ssh to be available.  make sure ssh is open on port 22.")
	conn := waitFor(c.instance.IPAddress, "22")
	conn.Close()
}

func waitFor(ip, port string) (conn net.Conn) {
	ipPort := fmt.Sprintf("%s:%s", ip, port)
	var err error

	for {
		conn, err = net.DialTimeout("tcp", ipPort, time.Duration(3)*time.Second)
		if err != nil {
			fmt.Print(".")
			time.Sleep(2 * time.Second)
		} else {
			fmt.Println()
			break
		}
	}

	return conn
}

func signalHandler(client *ec2Client) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		client.Close()
		os.Exit(0)
	}()
}

func Ec2() beam.Sender {
	backend := beam.NewServer()
	backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {
		var config, err = newConfig(ctx.Args)

		if err != nil {
			return err
		}

		ec2Conn, err := awsInit(config)
		if err != nil {
			return err
		}

		client := &ec2Client{config, ec2Conn, beam.NewServer(), nil, nil, nil}
		client.Server.OnSpawn(beam.Handler(client.spawn))
		client.Server.OnStart(beam.Handler(client.start))
		client.Server.OnStop(beam.Handler(client.stop))
		client.Server.OnAttach(beam.Handler(client.attach))
		client.Server.OnLs(beam.Handler(client.ls))
		client.Server.OnGet(beam.Handler(client.get))

		signalHandler(client)
		_, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: client.Server})

		return err
	}))

	return backend
}

func (c *ec2Client) Close() {
	if c.sshTunnel != nil {
		c.sshTunnel.Kill()
		if state, err := c.sshTunnel.Wait(); err != nil {
			fmt.Printf("Wait result: state:%v, err:%s\n", state, err)
		}
		c.sshTunnel = nil
	}
}

// thx to the rax.go :)
func (c *ec2Client) startSshTunnel() error {
	if c.instance == nil {
		return errors.New("no valid ec2 instance found.")
	}

	options := []string{
		"-o", "PasswordAuthentication=no",
		"-o", "LogLevel=quiet",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "CheckHostIP=no",
		"-o", "StrictHostKeyChecking=no",
		"-i", c.config.sshKey,
		"-A",
		"-p", "22",
		fmt.Sprintf("%s@%s", c.config.sshUser, c.instance.IPAddress),
		"-N",
		"-f",
		"-L", fmt.Sprintf("%s:localhost:%s", c.config.sshLocalPort, c.config.sshRemotePort),
	}

	cmd := exec.Command("ssh", options...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	c.sshTunnel = cmd.Process

	return nil
}

// TODO (aaron): load this externally
const userdata = `
#!/bin/bash
yum install -y docker
cat << EOF > /etc/sysconfig/docker
other_args="-H tcp://127.0.0.1:4243"
EOF
service docker start
`
