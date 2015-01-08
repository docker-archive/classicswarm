package main

import "github.com/codegangsta/cli"

var (
	flDiscovery = cli.StringFlag{
		Name:   "discovery",
		Value:  "",
		Usage:  "discovery service to use [token://<token>,\n\t\t\t\t  etcd://<ip1>,<ip2>/<path>,\n\t\t\t\t  file://path/to/file,\n\t\t\t\t  consul://<addr>/<path>,\n\t\t\t\t  zk://<ip1>,<ip2>/<path>,\n\t\t\t\t  <ip1>,<ip2>]",
		EnvVar: "SWARM_DISCOVERY",
	}
	flAddr = cli.StringFlag{
		Name:   "addr",
		Value:  "127.0.0.1:4243",
		Usage:  "ip to advertise",
		EnvVar: "SWARM_ADDR",
	}
	flHosts = cli.StringSliceFlag{
		Name:   "host, H",
		Value:  &cli.StringSlice{"tcp://127.0.0.1:2375"},
		Usage:  "ip/socket to listen on",
		EnvVar: "SWARM_HOST",
	}
	flHeartBeat = cli.IntFlag{
		Name:  "heartbeat, hb",
		Value: 25,
		Usage: "time in second between each heartbeat",
	}
	flEnableCors = cli.BoolFlag{
		Name:  "api-enable-cors, cors",
		Usage: "enable CORS headers in the remote API",
	}
	flTls = cli.BoolFlag{
		Name:  "tls",
		Usage: "use TLS; implied by --tlsverify=true",
	}
	flTlsCaCert = cli.StringFlag{
		Name:  "tlscacert",
		Usage: "trust only remotes providing a certificate signed by the CA given here",
	}
	flTlsCert = cli.StringFlag{
		Name:  "tlscert",
		Usage: "path to TLS certificate file",
	}
	flTlsKey = cli.StringFlag{
		Name:  "tlskey",
		Usage: "path to TLS key file",
	}
	flTlsVerify = cli.BoolFlag{
		Name:  "tlsverify",
		Usage: "use TLS and verify the remote",
	}
	flOverCommit = cli.IntFlag{
		Name:  "overcommit, oc",
		Usage: "overcommit to apply on resources",
		Value: 105,
	}
	flStrategy = cli.StringFlag{
		Name:  "strategy",
		Usage: "placement strategy to use [binpacking, random]",
		Value: "binpacking",
	}
	flFilter = cli.StringSliceFlag{
		Name:  "filter, f",
		Usage: "filter to use [constraint, health, port]",
		Value: &cli.StringSlice{"constraint", "health", "port"},
	}
)
