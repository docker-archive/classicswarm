package strategy

import (
	"errors"

	"github.com/docker/swarm/cluster"
)

type FakeNode struct {
	id         string
	name       string
	addr       string
	memory     int64
	usedmemory int64
	cpus       int64
	usedcpus   int64
	containers []*cluster.Container
}

func (fn *FakeNode) ID() string                            { return fn.id }
func (fn *FakeNode) Name() string                          { return fn.name }
func (fn *FakeNode) IP() string                            { return "" }
func (fn *FakeNode) Addr() string                          { return fn.addr }
func (fn *FakeNode) Images() []*cluster.Image              { return nil }
func (fn *FakeNode) Image(_ string) *cluster.Image         { return nil }
func (fn *FakeNode) Containers() []*cluster.Container      { return fn.containers }
func (fn *FakeNode) Container(_ string) *cluster.Container { return nil }
func (fn *FakeNode) TotalCpus() int64                      { return fn.cpus }
func (fn *FakeNode) UsedCpus() int64                       { return fn.usedcpus }
func (fn *FakeNode) TotalMemory() int64                    { return fn.memory }
func (fn *FakeNode) UsedMemory() int64                     { return fn.usedmemory }
func (fn *FakeNode) Labels() map[string]string             { return nil }
func (fn *FakeNode) IsHealthy() bool                       { return true }

func (fn *FakeNode) AddContainer(container *cluster.Container) error {
	memory := container.Info.Config.Memory
	cpus := container.Info.Config.CpuShares
	if fn.memory-memory < 0 || fn.cpus-cpus < 0 {
		return errors.New("not enough resources")
	}
	fn.usedmemory = fn.usedmemory + memory
	fn.usedcpus = fn.usedcpus + cpus

	fn.containers = append(fn.containers, container)
	return nil
}

func AddContainer(node cluster.Node, container *cluster.Container) error {
	if n, ok := node.(*FakeNode); ok {
		return n.AddContainer(container)
	}
	return errors.New("Not a FakeNode")
}
