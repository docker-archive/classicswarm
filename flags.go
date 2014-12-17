package main

import "github.com/codegangsta/cli"

var (
	flDiscovery = cli.StringFlag{
		Name:   "discovery",
		Value:  "",
		Usage:  "DiscoveryService to use [token://<token>, etcd://<ip1>,<ip2>/<path>, file://path/to/file]",
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
		Value:  &cli.StringSlice{"tcp://127.0.0.1:4243"},
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
		Usage: "Use TLS; implied by --tlsverify=true",
	}
	flTlsCaCert = cli.StringFlag{
		Name:  "tlscacert",
		Usage: "Trust only remotes providing a certificate signed by the CA given here",
	}
	flTlsCert = cli.StringFlag{
		Name:  "tlscert",
		Usage: "Path to TLS certificate file",
	}
	flTlsKey = cli.StringFlag{
		Name:  "tlskey",
		Usage: "Path to TLS key file",
	}
	flTlsVerify = cli.BoolFlag{
		Name:  "tlsverify",
		Usage: "Use TLS and verify the remote",
	}
	flScheduler = cli.StringFlag{
		Name:  "scheduler",
		Usage: "Scheduler to use [swarm, api]",
		Value: "swarm",
	}
	flSchedulerOpt = cli.StringSliceFlag{
		Name:  "scheduler-option",
		Usage: "Scheduler option. For swarm scheduler, default options: strategy:binpacking:0.05 filters:health,label,port",
		Value: &cli.StringSlice{"strategy:binpacking:0.05", "filters:health,label,port"},
	}
)
