package mesos

import (
	"crypto/rand"
	"encoding/hex"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
	"github.com/samalba/dockerclient"
)

type slave struct {
	cluster.Engine

	slaveID string
	offers  []*mesosproto.Offer
	updates map[string]chan string
}

// NewSlave creates mesos slave agent
func NewSlave(addr string, overcommitRatio float64, offer *mesosproto.Offer) *slave {
	slave := &slave{Engine: *cluster.NewEngine(addr, overcommitRatio)}
	slave.offers = []*mesosproto.Offer{offer}
	slave.slaveID = offer.SlaveId.GetValue()
	slave.updates = make(map[string]chan string)
	return slave
}

func (s *slave) toNode() *node.Node {
	return &node.Node{
		ID:          s.slaveID,
		IP:          s.IP,
		Addr:        s.Addr,
		Name:        s.Name,
		Cpus:        s.Cpus,
		Labels:      s.Labels,
		Containers:  s.Containers(),
		Images:      s.Images(),
		UsedMemory:  s.UsedMemory(),
		UsedCpus:    s.UsedCpus(),
		TotalMemory: s.TotalMemory(),
		TotalCpus:   s.TotalCpus(),
		IsHealthy:   s.IsHealthy(),
	}
}

func (s *slave) addOffer(offer *mesosproto.Offer) {
	s.offers = append(s.offers, offer)
}

func (s *slave) scalarResourceValue(name string) float64 {
	var value float64
	for _, offer := range s.offers {
		for _, resource := range offer.Resources {
			if *resource.Name == name {
				value += *resource.Scalar.Value
			}
		}
	}
	return value
}

func (s *slave) UsedMemory() int64 {
	return s.TotalMemory() - int64(s.scalarResourceValue("mem"))*1024*1024
}

func (s *slave) UsedCpus() int64 {
	return s.TotalCpus() - int64(s.scalarResourceValue("cpus"))
}

func (s *slave) create(driver *mesosscheduler.MesosSchedulerDriver, config *dockerclient.ContainerConfig, name string, pullImage bool) (*cluster.Container, error) {

	id := make([]byte, 6)
	nn, err := rand.Read(id)
	if nn != len(id) || err != nil {
		return nil, err
	}
	ID := hex.EncodeToString(id)

	s.updates[ID] = make(chan string)

	cpus := "cpus"
	typ := mesosproto.Value_SCALAR
	val := 1.0

	taskInfo := &mesosproto.TaskInfo{
		Name: &name,
		TaskId: &mesosproto.TaskID{
			Value: &ID,
		},
		SlaveId: s.offers[0].SlaveId,
		Resources: []*mesosproto.Resource{
			{
				Name: &cpus,
				Type: &typ,
				Scalar: &mesosproto.Value_Scalar{
					Value: &val,
				},
			},
		},
		Command: &mesosproto.CommandInfo{},
	}

	if len(config.Cmd) > 0 && config.Cmd[0] != "" {
		taskInfo.Command.Value = &config.Cmd[0]
	}

	if len(config.Cmd) > 1 {
		taskInfo.Command.Arguments = config.Cmd[1:]
	}

	taskInfo.Container = &mesosproto.ContainerInfo{
		Type: mesosproto.ContainerInfo_DOCKER.Enum(),
		Docker: &mesosproto.ContainerInfo_DockerInfo{
			Image: &config.Image,
		},
	}

	taskInfo.Command.Shell = proto.Bool(false)

	offerIds := []*mesosproto.OfferID{}

	for _, offer := range s.offers {
		offerIds = append(offerIds, offer.Id)
	}

	status, err := driver.LaunchTasks(offerIds, []*mesosproto.TaskInfo{taskInfo}, &mesosproto.Filters{})
	if err != nil {
		return nil, err
	}
	log.Debugf("create %v: %v", status, err)

	s.offers = []*mesosproto.Offer{}

	// block until we get the container
	<-s.updates[ID]

	// Register the container immediately while waiting for a state refresh.
	// Force a state refresh to pick up the newly created container.
	s.RefreshContainers(true)

	s.RLock()
	defer s.RUnlock()

	// TODO: We have to return the right container that was just created.
	// Once we receive the ContainerID from the executor.
	for _, container := range s.Containers() {
		return container, nil
	}

	return nil, nil
}
