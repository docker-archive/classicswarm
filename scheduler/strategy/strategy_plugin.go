package strategy

import (
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os/exec"
	"path"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/strategy/plugin"
	"github.com/samalba/dockerclient"
)

type StrategyPlugin struct {
	Name   string
	Cmd    *exec.Cmd
	Client *rpc.Client
}

var PLUGIN_DIR string = "."

func InitializePlugins(dir string) []string {
	PLUGIN_DIR = dir

	// reset the plugin map first
	resetStrategyMap()

	// scan plugin files next to `swarm`
	files, err := ioutil.ReadDir(PLUGIN_DIR)
	if err != nil {
		log.Fatal(err)
	}

	// when found, register each to the `strategies` registry
	for _, file := range files {
		if strings.HasPrefix(file.Name(), plugin.STRATEGY_PLUGIN_PREFIX) {
			name := strings.TrimPrefix(file.Name(), plugin.STRATEGY_PLUGIN_PREFIX)
			if runtime.GOOS == "windows" {
				name = strings.TrimSuffix(name, ".exe")
			}
			strategies[name] = &StrategyPlugin{Name: name}
		}
	}

	keys := make([]string, 0, len(strategies))
	for k := range strategies {
		keys = append(keys, k)
	}
	return keys
}

func StopPlugins() error {
	for _, strategy := range strategies {
		if st, ok := strategy.(interface {
			StopPlugin() error
		}); ok {
			st.StopPlugin()
		}
	}
	return nil
}

func (s *StrategyPlugin) startPlugin() error {
	exeName := plugin.STRATEGY_PLUGIN_PREFIX + s.Name
	s.Cmd = exec.Command(path.Join(PLUGIN_DIR, exeName))
	err := s.Cmd.Start()
	if err != nil {
		return err
	}
	// TODO handle this properly
	go s.Cmd.Wait()
	return nil
}

func (s *StrategyPlugin) StopPlugin() error {
	if s.Cmd != nil {
		err := s.Cmd.Process.Kill()
		s.Cmd = nil
		return err
	}
	return nil
}

func (s *StrategyPlugin) Initialize() error {
	s.startPlugin()

	client, err := plugin.NewClient(s.Name)
	if err != nil {
		return err
	}

	r := new(int)
	err = client.Call("Rpc.Initialize", r, r)
	s.Client = client
	return err
}

func (s *StrategyPlugin) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*cluster.Node) (*cluster.Node, error) {
	outs := []*plugin.Node{}
	nodeMap := make(map[string]*cluster.Node)
	for _, node := range nodes {
		nodeMap[node.ID] = node
		info := &plugin.Node{
			node.ID,
			node.IP,
			node.Addr,
			node.Name,
			node.Cpus,
			node.Memory,
			node.Labels,
			node.UsableMemory(),
			node.UsableCpus(),
			node.ReservedMemory(),
			node.ReservedCpus(),
			len(node.Containers()),
		}
		outs = append(outs, info)
	}
	request := &plugin.StrategyPluginRequest{config, outs}
	reply := &plugin.Node{}
	err := s.Client.Call("Rpc.PlaceContainer", request, reply)
	if err != nil {
		return nil, err
	}

	if node, exist := nodeMap[reply.ID]; exist {
		return node, nil
	}

	return nil, fmt.Errorf("Plugin %s: cannot find a node to place the new container", s.Name)
}
