package cluster

type FakeNode struct {
	name string
}

func (fn *FakeNode) ID() string                    { return "" }
func (fn *FakeNode) Name() string                  { return fn.name }
func (fn *FakeNode) IP() string                    { return "" }
func (fn *FakeNode) Addr() string                  { return "" }
func (fn *FakeNode) Images() []*Image              { return nil }
func (fn *FakeNode) Image(_ string) *Image         { return nil }
func (fn *FakeNode) Containers() []*Container      { return nil }
func (fn *FakeNode) Container(_ string) *Container { return nil }
func (fn *FakeNode) TotalCpus() int64              { return 0 }
func (fn *FakeNode) UsedCpus() int64               { return 0 }
func (fn *FakeNode) TotalMemory() int64            { return 0 }
func (fn *FakeNode) UsedMemory() int64             { return 0 }
func (fn *FakeNode) Labels() map[string]string     { return nil }
func (fn *FakeNode) IsHealthy() bool               { return true }
