package plugin

import (
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
)

var (
	STRATEGY_PLUGIN_PREFIX = "swarm-strategy-"
)

// This Node struct must be kept up with cluster.Node whenever possible.
type Node struct {
	ID     string
	IP     string
	Addr   string
	Name   string
	Cpus   int64
	Memory int64
	Labels map[string]string

	UsableMemory   int64
	UsableCpus     int64
	ReservedMemory int64
	ReservedCpus   int64
	Containers     int
}

type StrategyPluginRequest struct {
	Config *dockerclient.ContainerConfig
	Nodes  []*Node
}

type StrategyPluginResponse struct {
	Result *Node
	Error  string
}

// This interface is similar to PlacementStrategy
// except it depends only on package common and dockerclient
type StrategyPluginApi interface {
	Name() string
	Initialize() error
	PlaceContainer(config *dockerclient.ContainerConfig, nodes []*Node) (*Node, error)
}

type Rpc struct {
	plugin StrategyPluginApi
}

func (r *Rpc) Initialize(req *int, reply *int) error {
	return r.plugin.Initialize()
}

func (r *Rpc) PlaceContainer(req *StrategyPluginRequest, reply *Node) error {
	node, err := r.plugin.PlaceContainer(req.Config, req.Nodes)
	// copy
	*reply = *node
	return err
}

func Run(plugin StrategyPluginApi) {
	ln, err := Listen(plugin.Name())
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Infof("Caught signal %s: shutting down.", sig)
		ln.Close()
		os.Exit(0)
	}(sigc)

	server := rpc.NewServer()
	service := &Rpc{plugin}
	server.Register(service)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error(err)
			break
		}
		go server.ServeConn(conn)
	}
}
