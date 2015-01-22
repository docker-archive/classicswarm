package main

import (
	"github.com/codegangsta/cli"
	"os"
	"path/filepath"
	"runtime"
)

func homepath(p string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), p)
	}
	return filepath.Join(os.Getenv("HOME"), p)
}

func getDiscovery(c *cli.Context) string {
	if len(c.Args()) == 1 {
		return c.Args()[0]
	}
	return os.Getenv("SWARM_DISCOVERY")
}

var (
	flStore = cli.StringFlag{
		Name:  "rootdir",
		Value: homepath(".swarm"),
		Usage: "",
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
	flOverCommit = cli.Float64Flag{
		Name:  "overcommit, oc",
		Usage: "overcommit to apply on resources",
		Value: 0.05,
	}
	flStrategy = cli.StringFlag{
		Name:  "strategy",
		Usage: "placement strategy to use [binpacking, random]",
		Value: "binpacking",
	}
	flFilter = cli.StringSliceFlag{
		Name:  "filter, f",
		Usage: "filter to use [constraint, affinity, health, port]",
		Value: &cli.StringSlice{"constraint", "affinity", "health", "port"},
	}
)
