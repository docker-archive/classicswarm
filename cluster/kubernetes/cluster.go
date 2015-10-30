package kubernetes

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/units"
	"github.com/samalba/dockerclient"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/docker/swarm/cluster"
)

// Cluster struct for Kubernetes.
type Cluster struct {
	sync.RWMutex

	dockerEnginePort string
	kubeClient       unversioned.Interface
	TLSConfig        *tls.Config
	options          *cluster.DriverOpts
	engines          map[string]*cluster.Engine
}

const (
	serviceName                = "swarm"
	defaultDockerEnginePort    = "2375"
	defaultDockerEngineTLSPort = "2376"
)

var (
	errNotSupported   = errors.New("not supported with kubernetes")
	errNotImplemented = errors.New("not implemented in the kubernetes cluster")
)

func newKubeClient() (*unversioned.Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	log.Infof("Using %s for kubernetes master", config.Host)
	log.Infof("Using kubernetes API %s", config.Version)

	return unversioned.New(config)
}

// NewCluster create a new Kubernetes cluster based on the given cluster options.
func NewCluster(TLSConfig *tls.Config, master string, kubeClient unversioned.Interface, options cluster.DriverOpts) (cluster.Cluster, error) {
	log.WithFields(log.Fields{"name": "kubernetes"}).Debug("Initializing cluster")

	var err error
	if kubeClient == nil {
		kubeClient, err = newKubeClient()
		if err != nil {
			return nil, err
		}
	}

	engines, err := dockerEngines(kubeClient)
	if err != nil {
		return nil, err
	}

	cluster := &Cluster{
		dockerEnginePort: defaultDockerEnginePort,
		kubeClient:       kubeClient,
		TLSConfig:        TLSConfig,
		options:          &options,
		engines:          engines,
	}

	if cluster.TLSConfig != nil {
		cluster.dockerEnginePort = defaultDockerEngineTLSPort
	}

	log.Debugf("Kubernetes driver started")
	return cluster, nil
}

// CreateContainer creates a pod in a Kubernetes cluster running a single container
// based on the given container config.
func (c *Cluster) CreateContainer(config *cluster.ContainerConfig, name string) (*cluster.Container, error) {
	pods := c.kubeClient.Pods(api.NamespaceDefault)

	if name == "" {
		name = strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
	}

	pod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"swarm": "true"},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: name, Image: config.Image},
			},
		},
	}

	_, err := pods.Create(pod)
	if err != nil {
		return nil, err
	}

	pod, err = pods.Get(name)
	if err != nil {
		return nil, err
	}

	for {
		if api.IsPodReady(pod) {
			break
		}
		pod, err = pods.Get(name)
		if err != nil {
			return nil, err
		}
		time.Sleep(1 * time.Second)
	}

	containerStatus := pod.Status.ContainerStatuses[0]
	for {
		if containerStatus.Ready {
			log.Infof("Container %s is now ready", formatContainerID(containerStatus.ContainerID))
			break
		}
		time.Sleep(1 * time.Second)
	}

	return c.Container(formatContainerID(containerStatus.ContainerID)), nil
}

// RemoveContainer removes a pod from a Kubernetes cluster based on value of the
// container's io.kubernetes.pod.name label.
func (c *Cluster) RemoveContainer(container *cluster.Container, force, volumes bool) error {
	podLabel := strings.Split(container.Config.Labels["io.kubernetes.pod.name"], "/")
	namespace := podLabel[0]
	podName := podLabel[1]
	return c.kubeClient.Pods(namespace).Delete(podName, api.NewDeleteOptions(0))
}

// RANDOMENGINE returns a random engine.
func (c *Cluster) RANDOMENGINE() (*cluster.Engine, error) {
	return nil, nil
}

// Containers returns all containers running in the Kubernetes cluster.
func (c *Cluster) Containers() cluster.Containers {
	cs := make(cluster.Containers, 0)

	for _, engine := range c.engines {
		err := engine.Connect(nil)
		if err != nil {
			log.Error(err)
			return cs
		}
		err = engine.RefreshContainers(true)
		if err != nil {
			log.Error(err)
			return cs
		}
	}

	for _, engine := range c.engines {
		for _, container := range engine.Containers() {
			cs = append(cs, container)
		}
	}

	return cs
}

// Container returns the container with IdOrName in the cluster.
// Pods, not containers, are the smallest level of compute in a Kubernetes
// cluster. If a pod exisits with the same name as the given container ID
// or name, then the returned container returned will be the frist container
// in the pod.
func (c *Cluster) Container(IDOrName string) *cluster.Container {
	if len(IDOrName) == 0 {
		return nil
	}

	containerID, err := c.containerIDFromPod(IDOrName)
	if err != nil {
		log.Error(err)
	}
	if containerID != "" {
		IDOrName = containerID
	}

	c.RLock()
	defer c.RUnlock()
	return cluster.Containers(c.Containers()).Get(IDOrName)
}

// Image returns an image with IdOrName in the cluster.
func (c *Cluster) Image(IDOrName string) *cluster.Image {
	if len(IDOrName) == 0 {
		return nil
	}
	c.RLock()
	defer c.RUnlock()
	for _, engine := range c.engines {
		if image := engine.Image(IDOrName); image != nil {
			return image
		}
	}
	return nil
}

// Images returns all the images in the cluster.
func (c *Cluster) Images() cluster.Images {
	c.RLock()
	defer c.RUnlock()
	images := []*cluster.Image{}
	for _, engine := range c.engines {
		images = append(images, engine.Images()...)
	}
	return images
}

// Info gives minimal information about containers and resources on the kubernetes cluster
func (c *Cluster) Info() [][]string {
	info := [][]string{
		{"\bKubernetes Version", ""},
		{"\bNodes", fmt.Sprintf("%d", len(c.engines))},
	}

	for _, engine := range c.engines {
		info = append(info, []string{engine.Name, engine.Addr})
		info = append(info, []string{" └ Containers", fmt.Sprintf("%d", len(engine.Containers()))})
		info = append(info, []string{" └ Reserved CPUs", fmt.Sprintf("%d / %d", engine.UsedCpus(), engine.TotalCpus())})
		info = append(info, []string{" └ Reserved Memory", fmt.Sprintf("%s / %s", units.BytesSize(float64(engine.UsedMemory())), units.BytesSize(float64(engine.TotalMemory())))})
		labels := make([]string, 0, len(engine.Labels))
		for k, v := range engine.Labels {
			labels = append(labels, k+"="+v)
		}
		sort.Strings(labels)
		info = append(info, []string{" └ Labels", fmt.Sprintf("%s", strings.Join(labels, ", "))})
	}
	return info
}

// TotalCpus return the total memory of the Kubernetes cluster.
func (c *Cluster) TotalCpus() int64 {
	c.RLock()
	defer c.RUnlock()
	nodes, err := c.kubeClient.Nodes().List(labels.Everything(), fields.Everything())
	if err != nil {
		log.Error(err)
		return 0
	}

	total := int64(0)
	for _, node := range nodes.Items {
		total += node.Status.Capacity.Cpu().Value()
	}
	return total
}

// TotalMemory return the total memory of the Kubernetes cluster.
func (c *Cluster) TotalMemory() int64 {
	c.RLock()
	defer c.RUnlock()
	nodes, err := c.kubeClient.Nodes().List(labels.Everything(), fields.Everything())
	if err != nil {
		log.Error(err)
		return 0
	}

	total := int64(0)
	for _, node := range nodes.Items {
		total += node.Status.Capacity.Memory().Value()
	}
	return total
}

// BuildImage build an image.
func (c *Cluster) BuildImage(buildImage *dockerclient.BuildImage, out io.Writer) error {
	return errNotImplemented
}

// TagImage tag an image.
func (c *Cluster) TagImage(IDOrName string, repo string, tag string, force bool) error {
	return errNotSupported
}

// CreateVolume creates a volume in the cluster.
func (c *Cluster) CreateVolume(request *dockerclient.VolumeCreateRequest) (*cluster.Volume, error) {
	return nil, errNotSupported
}

// RemoveVolumes removes volumes from the cluster.
func (c *Cluster) RemoveVolumes(name string) (bool, error) {
	return false, errNotSupported
}

// Volume returns the volume name in the cluster.
func (c *Cluster) Volume(name string) *cluster.Volume {
	return nil
}

// Volumes returns all the volumes in the cluster.
func (c *Cluster) Volumes() []*cluster.Volume {
	return nil
}

// Pull will pull images on the cluster nodes.
func (c *Cluster) Pull(name string, authConfig *dockerclient.AuthConfig, callback func(where, status string, err error)) {
}

// Load images loads an images on the cluster nodes.
func (c *Cluster) Load(imageReader io.Reader, callback func(where, status string, err error)) {
}

// Import image imports an image on the cluster nodes.
func (c *Cluster) Import(source string, repository string, tag string, imageReader io.Reader, callback func(what, status string, err error)) {
}

// Handle callbacks for the events.
func (c *Cluster) Handle(e *cluster.Event) error {
	return nil
}

// RegisterEventHandler registers an event handler.
func (c *Cluster) RegisterEventHandler(h cluster.EventHandler) error {
	return nil
}

// RemoveImages removes images from the cluster.
func (c *Cluster) RemoveImages(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	return nil, errNotSupported
}

// RenameContainer renames a container.
func (c *Cluster) RenameContainer(container *cluster.Container, newName string) error {
	return errNotSupported
}

// CreateNetwork creates a network in the cluster.
func (c *Cluster) CreateNetwork(request *dockerclient.NetworkCreate) (*dockerclient.NetworkCreateResponse, error) {
	return nil, errNotSupported
}

// RemoveNetwork removes network from the cluster.
func (c *Cluster) RemoveNetwork(network *cluster.Network) error {
	return errNotSupported
}

// Networks returns all the networks in the cluster.
func (c *Cluster) Networks() cluster.Networks {
	return cluster.Networks{}
}

// containerIDFromPod extracts the Docker container ID from the Kubernetes pod
// identified by name.
func (c *Cluster) containerIDFromPod(name string) (string, error) {
	pods := c.kubeClient.Pods(api.NamespaceDefault)
	pod, err := pods.Get(name)
	if err != nil {
		return "", err
	}
	if pod != nil {
		return formatContainerID(pod.Status.ContainerStatuses[0].ContainerID), nil
	}
	return "", nil
}

// formatContainerID converts a Kubernetes container ID to an ID compatible with
// the Docker API.
func formatContainerID(id string) string {
	return strings.TrimPrefix(id, "docker://")
}

// dockerEngines returns the docker engines in the Kubernetes cluster.
func dockerEngines(client unversioned.Interface) (map[string]*cluster.Engine, error) {
	engines := make(map[string]*cluster.Engine)

	nodes, err := client.Nodes().List(labels.Everything(), fields.Everything())
	if err != nil {
		return engines, err
	}

	for _, node := range nodes.Items {
		var host string
		for _, address := range node.Status.Addresses {
			if address.Type == api.NodeInternalIP {
				host = address.Address
			}
		}
		engine := cluster.NewEngine(net.JoinHostPort(host, defaultDockerEnginePort), 0)
		engines[node.ObjectMeta.Name] = engine
	}
	return engines, nil
}
