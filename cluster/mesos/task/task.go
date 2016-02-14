package task

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/swarm/cluster"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
)

// Task struct inherits from TaskInfo and represents a mesos task
type Task struct {
	mesosproto.TaskInfo

	updates chan *mesosproto.TaskStatus

	config    *cluster.ContainerConfig
	Error     chan error
	container chan *cluster.Container
	done      bool
}

// GetContainer returns the container channel from the task
// where the Swarm API sends the created container
func (t *Task) GetContainer() chan *cluster.Container {
	return t.container
}

// SetContainer writes on the container channel from the task
func (t *Task) SetContainer(container *cluster.Container) {
	t.container <- container
}

// GetConfig returns the container configuration of the task
func (t *Task) GetConfig() *cluster.ContainerConfig {
	return t.config
}

// ID method returns the taskId
func (t *Task) ID() string {
	return t.TaskId.GetValue()
}

// Stopped method returns a boolean determining if the task
// is done
func (t *Task) Stopped() bool {
	return t.done
}

// Stop method sets the boolean determining if the task is done
func (t *Task) Stop() {
	t.done = true
}

// Build method builds the task
func (t *Task) Build(slaveID string, offers map[string]*mesosproto.Offer) {
	t.Command = &mesosproto.CommandInfo{Shell: proto.Bool(false)}

	t.Container = &mesosproto.ContainerInfo{
		Type: mesosproto.ContainerInfo_DOCKER.Enum(),
		Docker: &mesosproto.ContainerInfo_DockerInfo{
			Image: &t.config.Image,
		},
	}

	if t.config.Hostname != "" {
		t.Container.Hostname = proto.String(t.config.Hostname)
		if t.config.Domainname != "" {
			t.Container.Hostname = proto.String(t.config.Hostname + "." + t.config.Domainname)
		}
	}

	switch t.config.HostConfig.NetworkMode {
	case "none":
		t.Container.Docker.Network = mesosproto.ContainerInfo_DockerInfo_NONE.Enum()
	case "host":
		t.Container.Docker.Network = mesosproto.ContainerInfo_DockerInfo_HOST.Enum()
	case "default", "bridge", "":
		var ports []uint64

		for _, offer := range offers {
			ports = append(ports, getPorts(offer)...)
		}

		for containerProtoPort, bindings := range t.config.HostConfig.PortBindings {
			for _, binding := range bindings {
				containerInfo := strings.SplitN(containerProtoPort, "/", 2)
				containerPort, err := strconv.ParseUint(containerInfo[0], 10, 32)
				if err != nil {
					log.Warn(err)
					continue
				}

				var hostPort uint64

				if binding.HostPort != "" {
					hostPort, err = strconv.ParseUint(binding.HostPort, 10, 32)
					if err != nil {
						log.Warn(err)
						continue
					}
				} else if len(ports) > 0 {
					hostPort = ports[0]
					ports = ports[1:]
				}

				if hostPort == 0 {
					log.Warn("cannot find port to bind on the host")
					continue
				}

				protocol := "tcp"
				if len(containerInfo) == 2 {
					protocol = containerInfo[1]
				}
				t.Container.Docker.PortMappings = append(t.Container.Docker.PortMappings, &mesosproto.ContainerInfo_DockerInfo_PortMapping{
					HostPort:      proto.Uint32(uint32(hostPort)),
					ContainerPort: proto.Uint32(uint32(containerPort)),
					Protocol:      proto.String(protocol),
				})
				t.Resources = append(t.Resources, mesosutil.NewRangesResource("ports", []*mesosproto.Value_Range{mesosutil.NewValueRange(hostPort, hostPort)}))
			}
		}
		// TODO handle -P here
		t.Container.Docker.Network = mesosproto.ContainerInfo_DockerInfo_BRIDGE.Enum()
	default:
		log.Errorf("Unsupported network mode %q", t.config.HostConfig.NetworkMode)
		t.Container.Docker.Network = mesosproto.ContainerInfo_DockerInfo_BRIDGE.Enum()
	}

	if cpus := t.config.CpuShares; cpus > 0 {
		t.Resources = append(t.Resources, mesosutil.NewScalarResource("cpus", float64(cpus)))
	}

	if mem := t.config.Memory; mem > 0 {
		t.Resources = append(t.Resources, mesosutil.NewScalarResource("mem", float64(mem/1024/1024)))
	}

	if len(t.config.Cmd) > 0 && t.config.Cmd[0] != "" {
		t.Command.Value = &t.config.Cmd[0]
	}

	if len(t.config.Cmd) > 1 {
		t.Command.Arguments = t.config.Cmd[1:]
	}

	for key, value := range t.config.Labels {
		t.Container.Docker.Parameters = append(t.Container.Docker.Parameters, &mesosproto.Parameter{Key: proto.String("label"), Value: proto.String(fmt.Sprintf("%s=%s", key, value))})
	}

	for _, value := range t.config.Env {
		t.Container.Docker.Parameters = append(t.Container.Docker.Parameters, &mesosproto.Parameter{Key: proto.String("env"), Value: proto.String(value)})
	}

	if !t.config.AttachStdin && !t.config.AttachStdout && !t.config.AttachStderr {
		t.Container.Docker.Parameters = append(t.Container.Docker.Parameters, &mesosproto.Parameter{Key: proto.String("label"), Value: proto.String(fmt.Sprintf("%s=true", cluster.SwarmLabelNamespace+".mesos.detach"))})
	}

	t.SlaveId = &mesosproto.SlaveID{Value: &slaveID}
}

// NewTask function creates a task
func NewTask(config *cluster.ContainerConfig, name string, timeout time.Duration) (*Task, error) {
	id := stringid.TruncateID(stringid.GenerateRandomID())

	if name != "" {
		id = name + "." + id
	}
	// save the name in labels as the mesos containerizer will override it
	config.Labels[cluster.SwarmLabelNamespace+".mesos.name"] = name
	// FIXME: once Mesos changes merged no need to save the task id to know which container we launched
	config.Labels[cluster.SwarmLabelNamespace+".mesos.task"] = id

	task := &Task{
		config:    config,
		container: make(chan *cluster.Container),
		Error:     make(chan error),
		updates:   make(chan *mesosproto.TaskStatus),
	}

	task.Name = &name
	task.TaskId = &mesosproto.TaskID{Value: &id}
	task.Labels = &mesosproto.Labels{Labels: []*mesosproto.Label{{Key: proto.String("SWARM_CONTAINER_NAME"), Value: &name}}}

	go task.suicide(timeout)

	return task, nil
}

func (t *Task) suicide(timeout time.Duration) {
	<-time.After(timeout)
	if !t.Stopped() && t.SlaveId == nil {
		t.Error <- fmt.Errorf("container failed to start after %s", timeout)
	}
}

// SendStatus method writes the task status in the updates channel
func (t *Task) SendStatus(status *mesosproto.TaskStatus) {
	t.updates <- status
}

// GetStatus method reads the task status on the updates channel
func (t *Task) GetStatus() *mesosproto.TaskStatus {
	return <-t.updates
}

// Monitor method monitors task statuses
func (t *Task) Monitor() (bool, []byte, error) {
	taskStatus := t.GetStatus()

	switch taskStatus.GetState() {
	case mesosproto.TaskState_TASK_STAGING:
	case mesosproto.TaskState_TASK_STARTING:
	case mesosproto.TaskState_TASK_RUNNING:
	case mesosproto.TaskState_TASK_FINISHED:
		return true, taskStatus.Data, nil
	case mesosproto.TaskState_TASK_FAILED:
		errorMessage := taskStatus.GetMessage()
		if strings.Contains(errorMessage, "Abnormal executor termination") {
			errorMessage += " : please verify your SWARM_MESOS_USER is correctly set"
		}
		return true, nil, errors.New(errorMessage)
	case mesosproto.TaskState_TASK_KILLED:
		return true, taskStatus.Data, nil
	case mesosproto.TaskState_TASK_LOST:
		return true, nil, errors.New(taskStatus.GetMessage())
	case mesosproto.TaskState_TASK_ERROR:
		return true, nil, errors.New(taskStatus.GetMessage())
	}

	return false, taskStatus.Data, nil
}
