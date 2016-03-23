package cluster

import (
	"testing"

	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/stretchr/testify/assert"
)

func TestBuildContainerConfig(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Env)
	assert.Empty(t, config.Labels)

	config = BuildContainerConfig(container.Config{Env: []string{"test=true"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Env, 1)
	assert.Empty(t, config.Labels)

	config = BuildContainerConfig(container.Config{Env: []string{"constraint:test==true"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Env)
	assert.Len(t, config.Labels, 1)

	config = BuildContainerConfig(container.Config{Env: []string{"affinity:container==test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Env)
	assert.Len(t, config.Labels, 1)

	config = BuildContainerConfig(container.Config{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Env, 1)
	assert.Len(t, config.Labels, 2)
}

func TestSwarmID(t *testing.T) {
	// Getter / Setter
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.SwarmID())
	config.SetSwarmID("foo")
	assert.Equal(t, config.SwarmID(), "foo")
	assert.Equal(t, config.Labels[SwarmLabelNamespace+".id"], "foo")

	// Retrieve an existing ID.
	config = BuildContainerConfig(container.Config{Labels: map[string]string{SwarmLabelNamespace + ".id": "test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Equal(t, config.SwarmID(), "test")
}

func TestConstraints(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Constraints())

	config = BuildContainerConfig(container.Config{Env: []string{"constraint:test==true"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Constraints(), 1)

	config = BuildContainerConfig(container.Config{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Constraints(), 1)
}

func TestAffinities(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Affinities())

	config = BuildContainerConfig(container.Config{Env: []string{"affinity:container==test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Affinities(), 1)

	config = BuildContainerConfig(container.Config{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Len(t, config.Affinities(), 1)
	assert.Equal(t, len(config.Affinities()), 1)
}

func TestConsolidateResourceFields(t *testing.T) {

	config := BuildContainerConfig(container.Config{}, container.HostConfig{Resources: container.Resources{Memory: 4242, MemorySwap: 4343, CPUShares: 4444, CpusetCpus: "1-2"}}, network.NetworkingConfig{})
	assert.Equal(t, config.Memory, int64(4242))
	assert.Equal(t, config.MemorySwap, int64(4343))
	assert.Equal(t, config.CPUShares, int64(4444))
	assert.Equal(t, config.CpusetCpus, "1-2")
}

func TestAddAffinity(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Affinities())

	config.AddAffinity("image==~testimage")
	assert.Len(t, config.Affinities(), 1)
}

func TestRemoveAffinity(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.Empty(t, config.Affinities())

	config.AddAffinity("image==~testimage1")
	config.AddAffinity("image==~testimage2")
	assert.Len(t, config.Affinities(), 2)

	config.RemoveAffinity("image==~testimage1")
	assert.Len(t, config.Affinities(), 1)

	assert.Equal(t, config.Affinities()[0], "image==~testimage2")
}

func TestHaveNodeConstraint(t *testing.T) {
	config := BuildContainerConfig(container.Config{}, container.HostConfig{}, network.NetworkingConfig{})
	assert.False(t, config.HaveNodeConstraint())

	config = BuildContainerConfig(container.Config{Env: []string{"constraint:node==node1"}}, container.HostConfig{}, network.NetworkingConfig{})
	assert.True(t, config.HaveNodeConstraint())
}
