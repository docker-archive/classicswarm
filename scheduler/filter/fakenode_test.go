package filter

import "github.com/docker/swarm/cluster"

type FakeNode struct {
	id         string
	name       string
	addr       string
	containers []*cluster.Container
	images     []*cluster.Image
	labels     map[string]string
}

func (fn *FakeNode) ID() string               { return fn.id }
func (fn *FakeNode) Name() string             { return fn.name }
func (fn *FakeNode) IP() string               { return "" }
func (fn *FakeNode) Addr() string             { return fn.addr }
func (fn *FakeNode) Images() []*cluster.Image { return fn.images }
func (fn *FakeNode) Image(id string) *cluster.Image {
	for _, image := range fn.images {
		if image.Id == id {
			return image
		}
	}
	return nil
}
func (fn *FakeNode) Containers() []*cluster.Container { return fn.containers }
func (fn *FakeNode) Container(id string) *cluster.Container {
	for _, container := range fn.containers {
		if container.Id == id {
			return container
		}
	}
	return nil
}
func (fn *FakeNode) TotalCpus() int64          { return 0 }
func (fn *FakeNode) UsedCpus() float64         { return 0 }
func (fn *FakeNode) TotalMemory() int64        { return 0 }
func (fn *FakeNode) UsedMemory() int64         { return 0 }
func (fn *FakeNode) Labels() map[string]string { return fn.labels }
func (fn *FakeNode) IsHealthy() bool           { return true }

func (fn *FakeNode) AddContainer(container *cluster.Container) error {
	fn.containers = append(fn.containers, container)
	return nil
}
