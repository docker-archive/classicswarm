package etcd

import (
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/docker/swarm/discovery"
)

const DEFAULT_TTL = 30

type EtcdDiscoveryService struct {
	client *etcd.Client
	path   string
}

func init() {
	discovery.Register("etcd", Init)
}

func Init(uris string) (discovery.DiscoveryService, error) {
	var (
		// split here because uris can contain multiples ips
		// like `etcd://192.168.0.1,192.168.0.2,192.168.0.3/path`
		parts    = strings.SplitN(uris, "/", 2)
		ips      = strings.Split(parts[0], ",")
		machines []string
		path     = "/" + parts[1] + "/"
	)
	for _, ip := range ips {
		machines = append(machines, "http://"+ip)
	}

	client := etcd.NewClient(machines)
	client.CreateDir(path, DEFAULT_TTL) // skip error check error because it might already exists
	return EtcdDiscoveryService{client: client, path: path}, nil
}
func (s EtcdDiscoveryService) Fetch() ([]string, error) {
	resp, err := s.client.Get(s.path, true, true)
	if err != nil {
		return nil, err
	}
	nodes := []string{}

	for _, n := range resp.Node.Nodes {
		nodes = append(nodes, n.Value)
	}
	return nodes, nil
}

func (s EtcdDiscoveryService) Watch(heartbeat int) <-chan time.Time {
	watchChan := make(chan *etcd.Response)
	timeChan := make(chan time.Time)
	go s.client.Watch(s.path, 0, true, watchChan, nil)
	go func() {
		for {
			<-watchChan
			log.Debugf("[ETCD] Watch triggered")
			timeChan <- time.Now()
		}
	}()
	return timeChan
}

func (s EtcdDiscoveryService) Register(addr string) error {
	_, err := s.client.Set(path.Join(s.path, addr), addr, DEFAULT_TTL)
	return err
}
