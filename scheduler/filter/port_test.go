package filter

import (
	"fmt"
	"testing"

	"github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
	"github.com/stretchr/testify/assert"
)

func makeBinding(ip, port string) nat.PortMap {
	binding := nat.PortBinding{
		HostIP:   ip,
		HostPort: port,
	}
	bindingMap := map[nat.Port][]nat.PortBinding{nat.Port(fmt.Sprintf("%s/tcp", port)): {binding}}
	return nat.PortMap(bindingMap)
}

func TestPortFilterNoConflicts(t *testing.T) {
	var (
		p     = PortFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
	)

	// Request no ports.
	config := &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{},
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	// Make sure we don't filter anything out.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Request port 80.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("", "80"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}

	// Since there are no other containers in the cluster, this shouldn't
	// filter anything either.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// Add a container taking a different (4242) port.
	container := &cluster.Container{Container: types.Container{ID: "c1"}, Info: types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			HostConfig: &containertypes.HostConfig{
				PortBindings: makeBinding("", "4242"),
			},
		},
	}}
	assert.NoError(t, nodes[0].AddContainer(container))

	// Since no node is using port 80, there should be no filter
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)
}

func TestPortFilterSimple(t *testing.T) {
	var (
		p     = PortFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
	)

	// Add a container taking away port 80 to nodes[0].
	container := &cluster.Container{Container: types.Container{ID: "c1"}, Info: types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			HostConfig: &containertypes.HostConfig{
				PortBindings: makeBinding("", "80"),
			},
		},
	}}
	assert.NoError(t, nodes[0].AddContainer(container))

	// Request port 80.
	config := &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("", "80"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}

	// nodes[0] should be excluded since port 80 is taken away.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[0])
}

func TestPortFilterDifferentInterfaces(t *testing.T) {
	var (
		p     = PortFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
	)

	// Add a container taking away port 80 on every interface to nodes[0].
	container := &cluster.Container{Container: types.Container{ID: "c1"}, Info: types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			HostConfig: &containertypes.HostConfig{
				PortBindings: makeBinding("", "80"),
			},
		},
	}}
	assert.NoError(t, nodes[0].AddContainer(container))

	// Request port 80 for the local interface.
	config := &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("127.0.0.1", "80"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}

	// nodes[0] should be excluded since port 80 is taken away for every
	// interface.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[0])

	// Add a container taking away port 4242 on the local interface of
	// nodes[1].
	container = &cluster.Container{Container: types.Container{ID: "c1"}, Info: types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			HostConfig: &containertypes.HostConfig{
				PortBindings: makeBinding("127.0.0.1", "4242"),
			},
		},
	}}
	assert.NoError(t, nodes[1].AddContainer(container))
	// Request port 4242 on the same interface.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("127.0.0.1", "4242"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	// nodes[1] should be excluded since port 4242 is already taken on that
	// interface.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[1])

	// Request port 4242 on every interface.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("0.0.0.0", "4242"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	// nodes[1] should still be excluded since the port is not available on the same interface.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[1])

	// Request port 4242 on every interface using an alternative syntax.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("", "4242"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	// nodes[1] should still be excluded since the port is not available on the same interface.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[1])

	// Finally, request port 4242 on a different interface.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("192.168.1.1", "4242"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	// nodes[1] should be included this time since the port is available on the
	// other interface.
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Contains(t, result, nodes[1])
}

func TestPortFilterRandomAssignment(t *testing.T) {
	var (
		p     = PortFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-0-id",
				Name: "node-0-name",
				Addr: "node-0",
			},
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
		}
		result []*node.Node
		err    error
	)

	// Simulate a container that requested to map 80 to a random port.
	// In this case, HostConfig.PortBindings should contain a binding with no
	// HostPort defined and NetworkSettings.Ports should contain the actual
	// mapped port.
	container := &cluster.Container{
		Container: types.Container{ID: "c1"},
		Info: types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					PortBindings: nat.PortMap(
						map[nat.Port][]nat.PortBinding{
							nat.Port("80/tcp"): {
								{
									HostIP:   "",
									HostPort: "",
								},
							},
						},
					),
				}}}}

	container.Info.NetworkSettings = &types.NetworkSettings{
		NetworkSettingsBase: types.NetworkSettingsBase{
			Ports: nat.PortMap(
				map[nat.Port][]nat.PortBinding{
					nat.Port("80/tcp"): {
						{
							HostIP:   "127.0.0.1",
							HostPort: "1234",
						},
					},
				},
			),
		}}

	assert.NoError(t, nodes[0].AddContainer(container))

	// Request port 80.
	config := &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("", "80"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}

	// Since port "80" has been mapped to "1234", we should be able to request "80".
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, result, nodes)

	// However, we should not be able to request "1234" since it has been used for a random assignment.
	config = &cluster.ContainerConfig{Config: containertypes.Config{}, HostConfig: containertypes.HostConfig{
		PortBindings: makeBinding("", "1234"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.NotContains(t, result, nodes[0])
}

func TestPortFilterForHostMode(t *testing.T) {
	var (
		p     = PortFilter{}
		nodes = []*node.Node{
			{
				ID:   "node-1-id",
				Name: "node-1-name",
				Addr: "node-1",
			},
			{
				ID:   "node-2-id",
				Name: "node-2-name",
				Addr: "node-2",
			},
			{
				ID:   "node-3-id",
				Name: "node-3-name",
				Addr: "node-3",
			},
		}
		result []*node.Node
		err    error
	)

	// Add a container taking away port 80 in the host mode to nodes[0].
	container := &cluster.Container{
		Container: types.Container{ID: "c1"},
		Info: types.ContainerJSON{
			Config: &containertypes.Config{
				ExposedPorts: map[nat.Port]struct{}{nat.Port("80"): {}},
			},
			ContainerJSONBase: &types.ContainerJSONBase{
				HostConfig: &containertypes.HostConfig{
					NetworkMode: containertypes.NetworkMode("host"),
				},
			},
		}}

	assert.NoError(t, nodes[0].AddContainer(container))

	// Request port 80 in the host mode
	config := &cluster.ContainerConfig{Config: containertypes.Config{
		ExposedPorts: map[nat.Port]struct{}{nat.Port("80"): {}},
	}, HostConfig: containertypes.HostConfig{
		NetworkMode: containertypes.NetworkMode("host"),
	}, NetworkingConfig: networktypes.NetworkingConfig{}}

	// nodes[0] should be excluded since port 80 is taken away
	result, err = p.Filter(config, nodes, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.NotContains(t, result, nodes[0])
}
