package mesos

import (
	"testing"
	"time"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/cluster/mesos/task"
	"github.com/mesos/mesos-go/mesosutil"
	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestNewAgent(t *testing.T) {
	s := newAgent("SID", nil)

	assert.Equal(t, s.id, "SID")
	assert.Empty(t, s.offers)
	assert.Empty(t, s.tasks)
	assert.True(t, s.empty())
}

func TestAddOffer(t *testing.T) {
	s := newAgent("SID", nil)

	assert.Empty(t, s.offers)
	assert.True(t, s.empty())

	s.addOffer(mesosutil.NewOffer(mesosutil.NewOfferID("ID1"), nil, nil, "hostname1"))
	s.addOffer(mesosutil.NewOffer(mesosutil.NewOfferID("ID2"), nil, nil, "hostname1"))
	assert.Equal(t, len(s.offers), 2)
	assert.Equal(t, len(s.getOffers()), 2)
	assert.False(t, s.empty())

	s.addOffer(mesosutil.NewOffer(mesosutil.NewOfferID("ID1"), nil, nil, "hostname1"))
	assert.Equal(t, len(s.offers), 2)
	assert.Equal(t, len(s.getOffers()), 2)
}

func TestAddTask(t *testing.T) {
	s := newAgent("SID", nil)

	assert.Empty(t, s.tasks)
	assert.True(t, s.empty())

	t1, err := task.NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), "task1", 5*time.Second)
	assert.NoError(t, err)
	s.addTask(t1)

	t2, err := task.NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), "task1", 5*time.Second)
	assert.NoError(t, err)
	s.addTask(t2)
	assert.Equal(t, len(s.tasks), 2)
	assert.Equal(t, len(s.getTasks()), 2)
	assert.False(t, s.empty())

	s.addTask(t1)
	assert.Equal(t, len(s.tasks), 2)
	assert.Equal(t, len(s.getTasks()), 2)
}

func TestRemoveOffer(t *testing.T) {
	s := newAgent("SID", nil)

	assert.Empty(t, s.offers)

	s.addOffer(mesosutil.NewOffer(mesosutil.NewOfferID("ID1"), nil, nil, "hostname1"))
	s.addOffer(mesosutil.NewOffer(mesosutil.NewOfferID("ID2"), nil, nil, "hostname1"))
	assert.Equal(t, len(s.offers), 2)
	assert.Equal(t, len(s.getOffers()), 2)

	assert.True(t, s.removeOffer("ID1"))
	assert.Equal(t, len(s.offers), 1)
	assert.Equal(t, len(s.getOffers()), 1)

	assert.False(t, s.removeOffer("ID1"))
}

func TestRemoveTask(t *testing.T) {
	s := newAgent("SID", nil)

	assert.Empty(t, s.tasks)

	t1, err := task.NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), "task1", 5*time.Second)
	assert.NoError(t, err)
	s.addTask(t1)

	t2, err := task.NewTask(cluster.BuildContainerConfig(dockerclient.ContainerConfig{}), "task1", 5*time.Second)
	assert.NoError(t, err)
	s.addTask(t2)
	assert.Equal(t, len(s.tasks), 2)
	assert.Equal(t, len(s.getTasks()), 2)

	assert.True(t, s.removeTask(t1.TaskId.GetValue()))
	assert.Equal(t, len(s.tasks), 1)
	assert.Equal(t, len(s.getTasks()), 1)

	assert.False(t, s.removeTask(t1.TaskId.GetValue()))
}
