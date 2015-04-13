package mesos

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
	"github.com/samalba/dockerclient"
)

type slave struct {
	cluster.Engine

	slaveID  *mesosproto.SlaveID
	offers   []*mesosproto.Offer
	statuses map[string]chan *mesosproto.TaskStatus
}

// NewSlave creates mesos slave agent
func newSlave(addr string, overcommitRatio float64, offer *mesosproto.Offer) *slave {
	slave := &slave{Engine: *cluster.NewEngine(addr, overcommitRatio)}
	slave.offers = []*mesosproto.Offer{offer}
	slave.statuses = make(map[string]chan *mesosproto.TaskStatus)
	slave.slaveID = offer.SlaveId
	return slave
}

func (s *slave) toNode() *node.Node {
	return &node.Node{
		ID:          s.slaveID.GetValue(),
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

func generateTaskID() (string, error) {
	id := make([]byte, 6)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	return hex.EncodeToString(id), nil
}

func (s *slave) create(driver *mesosscheduler.MesosSchedulerDriver, config *dockerclient.ContainerConfig, name string, pullImage bool) (*cluster.Container, error) {
	ID, err := generateTaskID()
	if err != nil {
		return nil, err
	}

	s.statuses[ID] = make(chan *mesosproto.TaskStatus)

	resources := []*mesosproto.Resource{}

	if cpus := config.CpuShares; cpus > 0 {
		resources = append(resources, mesosutil.NewScalarResource("cpus", float64(cpus)))
	}

	if mem := config.Memory; mem > 0 {
		resources = append(resources, mesosutil.NewScalarResource("mem", float64(mem/1024/1024)))
	}

	taskInfo := &mesosproto.TaskInfo{
		Name: &name,
		TaskId: &mesosproto.TaskID{
			Value: &ID,
		},
		SlaveId:   s.slaveID,
		Resources: resources,
		Command:   &mesosproto.CommandInfo{},
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

	taskStatus := <-s.statuses[ID]
	delete(s.statuses, ID)

	switch taskStatus.GetState() {
	case mesosproto.TaskState_TASK_STAGING:
	case mesosproto.TaskState_TASK_STARTING:
	case mesosproto.TaskState_TASK_RUNNING:
	case mesosproto.TaskState_TASK_FINISHED:
	case mesosproto.TaskState_TASK_FAILED:
		return nil, errors.New(taskStatus.GetMessage())
	case mesosproto.TaskState_TASK_KILLED:
	case mesosproto.TaskState_TASK_LOST:
		return nil, errors.New(taskStatus.GetMessage())
	case mesosproto.TaskState_TASK_ERROR:
		return nil, errors.New(taskStatus.GetMessage())
	}

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
