package backends

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	azure "github.com/MSOpenTech/azure-sdk-for-go"
	"github.com/MSOpenTech/azure-sdk-for-go/clients/vmClient"
	"github.com/docker/libswarm"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"
)

type azureClient struct {
	Server         *libswarm.Server
	dockerInstance *libswarm.Client
	config         *AzureConfig
}

type AzureConfig struct {
	DnsName                 string
	UserName                string
	UserPassword            string
	Location                string
	ImageName               string
	RoleSize                string
	SshCert                 string
	SshPort                 int
	DockerPort              int
	DockerCertDir           string
	SubscriptionID          string
	SubscriptionCert        string
	PublishSettingsFilePath string
}

func Azure() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnVerb(libswarm.Spawn, libswarm.Handler(azureOnSpawn))
	return backend
}

func azureOnSpawn(ctx *libswarm.Message) error {
	var config, err = createAzureConfig(ctx.Args)
	if err != nil {
		return err
	}
	client := &azureClient{libswarm.NewServer(), nil, config}
	client.Server.OnVerb(libswarm.Ls, libswarm.Handler(client.ls))
	client.Server.OnVerb(libswarm.Attach, libswarm.Handler(client.attach))
	client.Server.OnVerb(libswarm.Spawn, libswarm.Handler(client.spawn))
	client.Server.OnVerb(libswarm.Start, libswarm.Handler(client.start))
	client.Server.OnVerb(libswarm.Stop, libswarm.Handler(client.stop))
	client.Server.OnVerb(libswarm.Get, libswarm.Handler(client.get))
	_, err = ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: client.Server})
	return err
}

func (client *azureClient) ls(ctx *libswarm.Message) error {
	output, err := client.dockerInstance.Ls()
	if err != nil {
		return err
	}
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Set, Args: output})
	return nil
}

func (client *azureClient) attach(ctx *libswarm.Message) error {
	if ctx.Args[0] == "" {
		ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: client.Server})
		<-make(chan struct{})
	} else {
		_, out, err := client.dockerInstance.Attach(ctx.Args[0])
		if err != nil {
			return err
		}
		ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: out})
	}
	return nil
}

func (client *azureClient) get(ctx *libswarm.Message) error {
	output, err := client.dockerInstance.Get()
	if err != nil {
		return err
	}
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Set, Args: []string{output}})
	return nil
}

func (client *azureClient) start(ctx *libswarm.Message) error {
	fmt.Printf("Starting azure client %s... \n", client.config.DnsName)
	err := getOrCreateAzureInstance(client.config)
	if err != nil {
		PrintErrorAndExit(err)
	}
	url := fmt.Sprintf("tcp://%s:%v", client.config.DnsName+".cloudapp.net", client.config.DockerPort)
	dockerClient, err := createDockerClient(client.config)
	if err != nil {
		PrintErrorAndExit(err)
	}
	forwardBackend := libswarm.AsClient(dockerClient)
	forwardInstance, err := forwardBackend.Spawn(url)
	if err != nil {
		return err
	}
	client.dockerInstance = forwardInstance
	fmt.Printf("Azure client up and running: name: %s\n", client.config.DnsName)
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: client.Server})
	return nil
}

func (client *azureClient) spawn(ctx *libswarm.Message) error {
	out, err := client.dockerInstance.Spawn(ctx.Args...)
	if err != nil {
		return err
	}
	ctx.Ret.Send(&libswarm.Message{Verb: libswarm.Ack, Ret: out})
	return nil
}

func (client *azureClient) stop(ctx *libswarm.Message) error {
	client.dockerInstance.Stop()
	return nil
}

func waitForDocker(config *AzureConfig) error {
	fmt.Println("Waiting for docker daemon on remote machine to be available.")
	maxRepeats := 24
	url := fmt.Sprintf("http://%s:%v", config.DnsName+".cloudapp.net", config.DockerPort)
	success := waitForDockerEndpoint(url, maxRepeats)
	if !success {
		fmt.Println("Restarting docker daemon on remote machine.")
		err := vmClient.RestartRole(config.DnsName, config.DnsName, config.DnsName)
		if err != nil {
			return err
		}
		success = waitForDockerEndpoint(url, maxRepeats)
		if !success {
			fmt.Println("Error: Can not run docker daemon on remote machine. Please check docker daemon at " + url)
		}
	}
	fmt.Println()
	fmt.Println("Docker daemon is ready.")
	return nil
}

func waitForDockerEndpoint(url string, maxRepeats int) bool {
	counter := 0
	for {
		resp, err := http.Get(url)
		error := err.Error()
		if strings.Contains(error, "malformed HTTP response") || len(error) == 0 {
			break
		}
		fmt.Print(".")
		if resp != nil {
			fmt.Println(resp)
		}
		time.Sleep(10 * time.Second)
		counter++
		if counter == maxRepeats {
			return false
		}
	}
	return true
}

func createDockerClient(options *AzureConfig) (libswarm.Sender, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	certDir := path.Join(usr.HomeDir, options.DockerCertDir)
	certPath := path.Join(certDir, "cert.pem")
	keyPath := path.Join(certDir, "key.pem")
	caPath := path.Join(certDir, "ca.pem")
	certDirExists := fileOrFolderExists(certDir)
	certExists := fileOrFolderExists(certPath)
	keyExists := fileOrFolderExists(keyPath)
	caExists := fileOrFolderExists(caPath)
	if !certDirExists || !certExists || !keyExists || !caExists {
		return nil, errors.New("Can not find some of the docker certificates. Please create them first. Info can be found here: https://docs.docker.com/articles/https/")
	}
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	caBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{cert}
	cas := x509.NewCertPool()
	cas.AppendCertsFromPEM(caBytes)
	tlsConfig.RootCAs = cas
	dockerClient := DockerClientWithConfig(&DockerClientConfig{
		Scheme:          "https",
		URLHost:         options.DnsName,
		TLSClientConfig: tlsConfig,
	})
	return dockerClient, nil
}

func fileOrFolderExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("No such file or directory: %s \n", path)
		return false
	}
	return true
}

func setUserSubscription(config *AzureConfig) error {
	if len(config.PublishSettingsFilePath) != 0 {
		err := azure.ImportPublishSettingsFile(config.PublishSettingsFilePath)
		if err != nil {
			return err
		}
		return nil
	}
	err := azure.ImportPublishSettings(config.SubscriptionID, config.SubscriptionCert)
	if err != nil {
		return err
	}
	return nil
}

func getOrCreateAzureInstance(config *AzureConfig) error {
	err := setUserSubscription(config)
	if err != nil {
		return err
	}
	dockerVM, err := vmClient.GetVMDeployment(config.DnsName, config.DnsName)
	if err != nil {
		return err
	}
	if dockerVM == nil {
		err = createDockerVM(config)
		if err != nil {
			return err
		}
	} else {
		if dockerVM.Status != "Running" {
			fmt.Printf("Starting existing azure service: name : %s \n", dockerVM.Name)
			err = vmClient.StartRole(config.DnsName, config.DnsName, config.DnsName)
			if err != nil {
				return err
			}
		}
	}
	err = waitForDocker(config)
	if err != nil {
		return err
	}
	return nil
}

func createAzureConfig(args []string) (options *AzureConfig, err error) {
	options = createDefaultOptions()
	parseUserSpecifiedOptions(args, options)
	if (len(options.SubscriptionID) == 0 || len(options.SubscriptionCert) == 0) && len(options.PublishSettingsFilePath) == 0 {
		options.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
		options.SubscriptionCert = os.Getenv("AZURE_SUBSCRIPTION_CERT")
		options.PublishSettingsFilePath = os.Getenv("AZURE_PUBLISH_SETTINGS_FILE")
	} else {
		return options, nil
	}
	if (len(options.SubscriptionID) == 0 || len(options.SubscriptionCert) == 0) && len(options.PublishSettingsFilePath) == 0 {
		PrintErrorAndExit(errors.New("Please specify azure subscription params using \n Environment variables: AZURE_SUBSCRIPTION_ID and AZURE_SUBSCRIPTION_CERT or AZURE_PUBLISH_SETTINGS_FILE \n OR \n Libswarm options: --azure-subscription-id and --azure-subscription-cert or --azure-publish-settings-file"))
	}
	return options, nil
}

func createDefaultOptions() *AzureConfig {
	options := new(AzureConfig)
	options.DnsName = "docker-azure-swarm"
	options.UserName = "tcuser"
	options.UserPassword = "Docker123"
	options.Location = "West US"
	options.ImageName = "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"
	options.RoleSize = "Small"
	options.SshPort = 22
	options.DockerPort = 4243
	options.DockerCertDir = ".docker"
	return options
}

func parseUserSpecifiedOptions(args []string, options *AzureConfig) {
	var optValPair []string
	var opt, val string
	for _, value := range args {
		optValPair = strings.Split(value, "=")
		opt, val = optValPair[0], optValPair[1]
		switch opt {
		case "--name":
			options.DnsName = val
		case "--username":
			if val == "docker" {
				PrintErrorAndExit(errors.New("You can not use username 'docker'. Please specify another username."))
			}
			options.UserName = val
		case "--password":
			options.UserPassword = val
		case "--image-name":
			options.ImageName = val
		case "--location":
			options.Location = val
		case "--size":
			if val != "ExtraSmall" && val != "Small" && val != "Medium" &&
				val != "Large" && val != "ExtraLarge" &&
				val != "A5" && val != "A6" && val != "A7" {
				PrintErrorAndExit(errors.New("Invalid VM size specified with --size"))
			}
			options.RoleSize = val
		case "--ssh":
			sshPort, err := strconv.Atoi(val)
			if err != nil {
				PrintErrorAndExit(err)
			}
			options.SshPort = sshPort
		case "--docker-port":
			dockerPort, err := strconv.Atoi(val)
			if err != nil {
				PrintErrorAndExit(err)
			}
			options.DockerPort = dockerPort
		case "--docker-cert-dir":
			options.DockerCertDir = val
		case "--ssh-cert":
			options.SshCert = val
		case "--azure-subscription-id":
			options.SubscriptionID = val
		case "--azure-subscription-cert":
			options.SubscriptionCert = val
		case "--azure-publish-settings-file":
			options.PublishSettingsFilePath = val
		default:
			PrintErrorAndExit(errors.New(fmt.Sprintf("Unrecognizable option: %s value: %s", opt, val)))
		}
	}
}

func createDockerVM(options *AzureConfig) error {
	vmConfig, err := vmClient.CreateAzureVMConfiguration(options.DnsName, options.RoleSize, options.ImageName, options.Location)
	if err != nil {
		return err
	}
	vmConfig, err = vmClient.AddAzureLinuxProvisioningConfig(vmConfig, options.UserName, options.UserPassword, options.SshCert)
	if err != nil {
		return err
	}
	vmConfig, err = vmClient.SetAzureDockerVMExtension(vmConfig, options.DockerCertDir, options.DockerPort, "0.3")
	if err != nil {
		return err
	}
	err = vmClient.CreateAzureVM(vmConfig, options.DnsName, options.Location)
	if err != nil {
		return err
	}
	return nil
}

func PrintErrorAndExit(err error) {
	fmt.Println("Error: ")
	fmt.Println(err)
	os.Exit(2)
}
