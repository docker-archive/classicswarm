package strategy

import (
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

func createNode(ID string, memory int64, cpus int64) *cluster.Node {
	node := cluster.NewNode(ID, 0.05)
	node.ID = ID
	node.Memory = memory * 1024 * 1024 * 1024
	node.Cpus = cpus
	return node
}

func createConfig(memory int64, cpus int64) *dockerclient.ContainerConfig {
	return &dockerclient.ContainerConfig{Memory: memory * 1024 * 1024 * 1024, CpuShares: cpus}
}

func createContainer(ID string, config *dockerclient.ContainerConfig) *cluster.Container {
	return &cluster.Container{Container: dockerclient.Container{Id: ID}, Info: dockerclient.ContainerInfo{Config: config}}
}
