package network

import "github.com/docker/docker/pkg/nat"

// Address represents an IP address
type Address struct {
	Addr      string
	PrefixLen int
}

// IPAM represents IP Address Management
type IPAM struct {
	Driver string       `json:"driver"`
	Config []IPAMConfig `json:"config"`
}

// IPAMConfig represents IPAM configurations
type IPAMConfig struct {
	Subnet     string            `json:"subnet,omitempty"`
	IPRange    string            `json:"ip_range,omitempty"`
	Gateway    string            `json:"gateway,omitempty"`
	AuxAddress map[string]string `json:"auxiliary_address,omitempty"`
}

// Settings stores configuration details about the daemon network config
// TODO Windows. Many of these fields can be factored out.,
type Settings struct {
	Bridge                 string
	EndpointID             string
	SandboxID              string
	Gateway                string
	GlobalIPv6Address      string
	GlobalIPv6PrefixLen    int
	HairpinMode            bool
	IPAddress              string
	IPPrefixLen            int
	IPv6Gateway            string
	LinkLocalIPv6Address   string
	LinkLocalIPv6PrefixLen int
	MacAddress             string
	Networks               []string
	Ports                  nat.PortMap
	SandboxKey             string
	SecondaryIPAddresses   []Address
	SecondaryIPv6Addresses []Address
}
