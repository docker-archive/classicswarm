package cluster

import (
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestBuildContainerConfig(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.Env)
	assert.Empty(t, config.Labels)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"test=true"}})
	assert.Len(t, config.Env, 1)
	assert.Empty(t, config.Labels)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:test==true"}})
	assert.Empty(t, config.Env)
	assert.Len(t, config.Labels, 1)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==test"}})
	assert.Empty(t, config.Env)
	assert.Len(t, config.Labels, 1)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Len(t, config.Env, 1)
	assert.Len(t, config.Labels, 2)
}

func TestSwarmID(t *testing.T) {
	// Getter / Setter
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.SwarmID())
	config.SetSwarmID("foo")
	assert.Equal(t, config.SwarmID(), "foo")
	assert.Equal(t, config.Labels[SwarmLabelNamespace+".id"], "foo")

	// Retrieve an existing ID.
	config = BuildContainerConfig(dockerclient.ContainerConfig{Labels: map[string]string{SwarmLabelNamespace + ".id": "test"}})
	assert.Equal(t, config.SwarmID(), "test")
}

func TestConstraints(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.Constraints())

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:test==true"}})
	assert.Len(t, config.Constraints(), 1)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Len(t, config.Constraints(), 1)
}

func TestAffinities(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.Affinities())

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"affinity:container==test"}})
	assert.Len(t, config.Affinities(), 1)

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Len(t, config.Affinities(), 1)
	assert.Equal(t, len(config.Affinities()), 1)
}

func TestConsolidateResourceFields(t *testing.T) {
	for _, config := range []*ContainerConfig{
		BuildContainerConfig(dockerclient.ContainerConfig{Memory: 4242, MemorySwap: 4343, CpuShares: 4444, Cpuset: "1-2"}),
		BuildContainerConfig(dockerclient.ContainerConfig{HostConfig: dockerclient.HostConfig{Memory: 4242, MemorySwap: 4343, CpuShares: 4444, CpusetCpus: "1-2"}}),
	} {
		assert.Equal(t, config.Memory, int64(4242))
		assert.Equal(t, config.MemorySwap, int64(4343))
		assert.Equal(t, config.CpuShares, int64(4444))
		assert.Equal(t, config.Cpuset, "1-2")
		assert.Equal(t, config.HostConfig.Memory, int64(4242))
		assert.Equal(t, config.HostConfig.MemorySwap, int64(4343))
		assert.Equal(t, config.HostConfig.CpuShares, int64(4444))
		assert.Equal(t, config.HostConfig.CpusetCpus, "1-2")
	}

}

func TestAddAffinity(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.Affinities())

	config.AddAffinity("image==~testimage")
	assert.Len(t, config.Affinities(), 1)
}

func TestRemoveAffinity(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.Empty(t, config.Affinities())

	config.AddAffinity("image==~testimage1")
	config.AddAffinity("image==~testimage2")
	assert.Len(t, config.Affinities(), 2)

	config.RemoveAffinity("image==~testimage1")
	assert.Len(t, config.Affinities(), 1)

	assert.Equal(t, config.Affinities()[0], "image==~testimage2")
}

func TestHaveNodeConstraint(t *testing.T) {
	config := BuildContainerConfig(dockerclient.ContainerConfig{})
	assert.False(t, config.HaveNodeConstraint())

	config = BuildContainerConfig(dockerclient.ContainerConfig{Env: []string{"constraint:node==node1"}})
	assert.True(t, config.HaveNodeConstraint())
}
