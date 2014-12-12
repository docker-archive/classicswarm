package discovery

type DiscoveryBackend interface {
	Supports(url string) (bool, error)
	FetchSlaves(url, token string) ([]string, error)
	RegisterSlave(url, addr, token string) error
	CreateCluster(url string) (string, error)
}
