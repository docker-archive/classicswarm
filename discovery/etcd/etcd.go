package etcd

import (
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/docker/swarm/discovery"
)

type EtcdDiscoveryService struct {
	ttl    uint64
	client *etcd.Client
	path   string
}

func init() {
	discovery.Register("etcd", Init)
}

func Init(uris string, heartbeat int) (discovery.DiscoveryService, error) {
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
	ttl := uint64(heartbeat * 3 / 2)
	client.CreateDir(path, ttl) // skip error check error because it might already exists
	return EtcdDiscoveryService{client: client, path: path, ttl: ttl}, nil
}
func (s EtcdDiscoveryService) Fetch() ([]*discovery.Node, error) {
	resp, err := s.client.Get(s.path, true, true)
	if err != nil {
		return nil, err
	}

	var nodes []*discovery.Node

	for _, n := range resp.Node.Nodes {
		nodes = append(nodes, discovery.NewNode(n.Value))
	}
	return nodes, nil
}

func (s EtcdDiscoveryService) Watch() <-chan time.Time {
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
	_, err := s.client.Set(path.Join(s.path, addr), addr, s.ttl)
	return err
}
