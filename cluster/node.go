package cluster

type Node interface {
	ID() string
	Name() string

	IP() string   //to inject the actual IP of the machine in docker ps (hostname:port or ip:port)
	Addr() string //to know where to connect with the proxy

	Images() []*Image                     //used by the API
	Image(IdOrName string) *Image         //used by the filters
	Containers() []*Container             //used by the filters
	Container(IdOrName string) *Container //used by the filters

	TotalCpus() int64   //used by the strategy
	UsedCpus() int64    //used by the strategy
	TotalMemory() int64 //used by the strategy
	UsedMemory() int64  //used by the strategy

	Labels() map[string]string //used by the filters

	IsHealthy() bool
}
