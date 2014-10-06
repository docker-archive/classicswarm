package backends

import (
	"encoding/json"
	"fmt"
	"github.com/docker/libswarm"
	"github.com/goinggo/mapstructure"
	"github.com/openstack/goopenstack"
	"io/ioutil"
	"strings"
	"time"
)

type osConfig struct {
	flavor        string
	image         string
	securityGroup string
	keyname       string
	//Add more openstack configuration as needed
}

type Config struct {
	Image string `json:"image"`
}

type State struct {
	Running  bool `json:"Running"`
	ExitCode int  `json:"ExitCode"`
}

type containerProp struct {
	Config  *Config `json:"Config"`
	Id      string  `json:"ID"`
	Created string  `json:Created`
	Name    string  `json:Name`
	State   *State  `json:State`
}

type novaServerProp struct {
	Hostname string `jpath:"server.OS-EXT-SRV-ATTR:host"`
	Image    string `jpath:"server.image.id"`
	ID       string `jpath:"server.id"`
	Created  string `jpath:"server.created"`
	Status   string `jpath:"server.status"`
	Name     string `jpath:"server.name"`
}

type novaCreateProp struct {
	Id string `jpath:"server.id"`
}

func defaultOsConfigValues() (config *osConfig) {
	config = new(osConfig)
	config.flavor = "1"
	config.image = "cirros"
	config.securityGroup = "default"

	return config
}

func parseConfig(args []string) (config *osConfig, err error) {
	var optValPair []string
	var opt, val string

	config = defaultOsConfigValues()

	for _, value := range args {
		optValPair = strings.Split(value, "=")
		opt, val = optValPair[0], optValPair[1]

		switch opt {
		case "--flavor":
			flavorName := val
			config.flavor, err = openstack.GetFlavorID(flavorName)
			if err != nil {
				return nil, fmt.Errorf("Invalid flavor ID")
			}
		case "--security_group":
			config.securityGroup = val
		case "--key-name":
			config.keyname = val
		default:
			fmt.Printf("Unrecognizable option: %s value: %s", opt, val)
			return nil, fmt.Errorf("parse Error")
		}
	}
	return config, nil
}

func Openstack() libswarm.Sender {
	backend := libswarm.NewServer()
	fmt.Printf("Initializing Openstack backend engine\n")
	backend.OnVerb(libswarm.Spawn, libswarm.Handler(func(ctx *libswarm.Message) error {
		if !openstack.IsAuthenticated() {
			fmt.Println("Openstack environment varibales are not set: OS_TENANT_NAME,OS_USERNAME,OS_PASSWORD,OS_AUTH_URL")
			return fmt.Errorf("Access Error")
		}
		var config, err = parseConfig(ctx.Args)
		if err != nil {
			return err
		}

		client := &openstackBackend{config, libswarm.NewServer()}
		client.Server.OnVerb(libswarm.Attach, libswarm.Handler(client.attach))
		client.Server.OnVerb(libswarm.Start, libswarm.Handler(client.ack))
		client.Server.OnVerb(libswarm.Ls, libswarm.Handler(client.ls))
		client.Server.OnVerb(libswarm.Spawn, libswarm.Handler(client.spawn))
		_, err = ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: client.Server})
		return err
	}))
	return backend
}

type openstackBackend struct {
	config *osConfig
	Server *libswarm.Server
}

func (os *openstackBackend) attach(ctx *libswarm.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: os.Server})
		for {
			time.Sleep(1 * time.Second)
		}
	} else {
		c := os.newNovaClient(ctx.Args[0])
		ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: c})
	}
	return nil
}

func (os *openstackBackend) ack(ctx *libswarm.Message) error {
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: os.Server})
	return nil
}

func (os *openstackBackend) ls(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.PublicDomain)

	resp, err := openstack.OpenstackCall(conn, "GET", "/servers", "")
	if err != nil {
		return fmt.Errorf("ls server: get: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ls server : read body: %v", err)
	}
	var resData map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resData); err != nil {
		panic(err)
	}
	ids := []string{}
	for idx := range resData["servers"] {
		server := resData["servers"][idx]
		ids = append(ids, (server["id"]).(string))
	}
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Set, Args: ids})
	return nil
}

func (os *openstackBackend) spawn(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.AdminDomain)
	var reqJson map[string]interface{}
	if err := json.Unmarshal([]byte(ctx.Args[0]), &reqJson); err != nil {
		return err
	}

	serverName := reqJson["name"].(string)
	imageName := reqJson["image"].(string)
	imageID, err := openstack.GetImageID(imageName)
	if err != nil {
		return fmt.Errorf("Invalid Image Name")
	}
	/*
		var createArgs = &novaCreateArgs{
			server: &CreateParams{
				name:      serverName,
				flavorRef: os.config.flavor,
				imageRef:  os.config.image,
			},
		}
		inArgs, _ := json.Marshal(createArgs)
	*/
	body := fmt.Sprintf("{\"server\":{\"flavorRef\": \"%s\", \"name\": \"%s\", \"imageRef\": \"%s\"}}", os.config.flavor, serverName, imageID)
	path := fmt.Sprintf("/servers")
	resp, err := openstack.OpenstackCall(conn, "POST", path, body)
	if err != nil {
		return fmt.Errorf("create server: post: %v", err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("create server : read body: %v", err)
	}

	var resData map[string]interface{}
	if err := json.Unmarshal([]byte(resBody), &resData); err != nil {
		panic(err)
	}
	var newServer novaCreateProp
	mapstructure.DecodePath(resData, &newServer)
	c := os.newNovaClient(newServer.Id)

	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: c})
	return nil
}

func (os *openstackBackend) newNovaClient(id string) libswarm.Sender {
	c := &novaClient{openstackBackend: os, id: id}
	instance := libswarm.NewServer()
	instance.OnVerb(libswarm.Get, libswarm.Handler(c.get))
	instance.OnVerb(libswarm.Start, libswarm.Handler(c.start))
	instance.OnVerb(libswarm.Stop, libswarm.Handler(c.stop))
	instance.OnVerb(libswarm.Delete, libswarm.Handler(c.delete))
	return instance
}

type novaClient struct {
	openstackBackend *openstackBackend
	id               string
}

func (osCli *novaClient) get(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.PublicDomain)
	path := fmt.Sprintf("/servers/%s", osCli.id)
	resp, err := openstack.OpenstackCall(conn, "GET", path, "")
	if err != nil {
		return fmt.Errorf("ls server: get: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ls server : read body: %v", err)
	}
	var resData map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resData); err != nil {
		panic(err)
	}
	var novaServer novaServerProp
	mapstructure.DecodePath(resData, &novaServer)
	isRunning := false
	exitCode := 1
	if novaServer.Status == "ACTIVE" {
		isRunning = true
		exitCode = 0
	}
	var containerProp = &containerProp{
		Id:      novaServer.ID,
		Created: novaServer.Created,
		Name:    novaServer.Name,
		State: &State{
			Running:  isRunning,
			ExitCode: exitCode,
		},
		Config: &Config{
			Image: novaServer.Image,
		},
	}
	outArgs, _ := json.Marshal(containerProp)
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Set, Args: []string{string(outArgs)}})
	return nil
}

func (osCli *novaClient) start(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.PublicDomain)
	path := fmt.Sprintf("/servers/%s/action", osCli.id)
	body := "{\"os-start\":null}"
	_, err := openstack.OpenstackCall(conn, "POST", path, body)
	if err != nil {
		return fmt.Errorf("start server: post: %v", err)
	}
	if _, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack}); err != nil {
		return err
	}
	return nil
}

func (osCli *novaClient) stop(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.PublicDomain)
	path := fmt.Sprintf("/servers/%s/action", osCli.id)
	body := "{\"os-stop\":null}"
	_, err := openstack.OpenstackCall(conn, "POST", path, body)
	if err != nil {
		return fmt.Errorf("stop server: post: %v", err)
	}
	if _, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack}); err != nil {
		return err
	}
	return nil
}

func (osCli *novaClient) delete(ctx *libswarm.Message) error {
	conn, _ := openstack.GetOpenstackConnection("nova", openstack.PublicDomain)
	path := fmt.Sprintf("/servers/%s", osCli.id)
	_, err := openstack.OpenstackCall(conn, "DELETE", path, "")
	if err != nil {
		return fmt.Errorf("stop server: post: %v", err)
	}
	if _, err := ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack}); err != nil {
		return err
	}
	return nil
}
