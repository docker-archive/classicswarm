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

	// Retrieve an existing ID.
	config = BuildContainerConfig(dockerclient.ContainerConfig{Labels: map[string]string{namespace + ".id": "test"}})
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
