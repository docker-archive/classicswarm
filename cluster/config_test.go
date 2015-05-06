package cluster

import (
	"testing"

	"github.com/samalba/dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestBuildContainerConfig(t *testing.T) {
	config := BuildContainerConfig(&dockerclient.ContainerConfig{})
	assert.Equal(t, len(config.Env), 0)
	assert.Equal(t, len(config.Labels), 0)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"test=true"}})
	assert.Equal(t, len(config.Env), 1)
	assert.Equal(t, len(config.Labels), 0)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"constraint:test==true"}})
	assert.Equal(t, len(config.Env), 0)
	assert.Equal(t, len(config.Labels), 1)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"affinity:container==test"}})
	assert.Equal(t, len(config.Env), 0)
	assert.Equal(t, len(config.Labels), 1)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Equal(t, len(config.Env), 1)
	assert.Equal(t, len(config.Labels), 2)
}

func TestConstraints(t *testing.T) {
	config := BuildContainerConfig(&dockerclient.ContainerConfig{})
	assert.Equal(t, len(config.Constraints()), 0)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"constraint:test==true"}})
	assert.Equal(t, len(config.Constraints()), 1)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Equal(t, len(config.Constraints()), 1)
}

func TestAffinities(t *testing.T) {
	config := BuildContainerConfig(&dockerclient.ContainerConfig{})
	assert.Equal(t, len(config.Affinities()), 0)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"affinity:container==test"}})
	assert.Equal(t, len(config.Affinities()), 1)

	config = BuildContainerConfig(&dockerclient.ContainerConfig{Env: []string{"test=true", "constraint:test==true", "affinity:container==test"}})
	assert.Equal(t, len(config.Affinities()), 1)
}
