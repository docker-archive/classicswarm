package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/docker/swarm/scheduler/filter"
	"github.com/docker/swarm/scheduler/strategy"
)

func homepath(p string) string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, p)
}

func getDiscovery(c *cli.Context) string {
	if len(c.Args()) == 1 {
		return c.Args()[0]
	}
	return os.Getenv("SWARM_DISCOVERY")
}

var (
	flJoinAdvertise = cli.StringFlag{
		Name:   "advertise, addr",
		Usage:  "Address of the Docker Engine joining the cluster. Swarm manager(s) MUST be able to reach the Docker Engine at this address.",
		EnvVar: "SWARM_ADVERTISE",
	}
	flJoinRandomDelay = cli.StringFlag{
		Name:  "delay",
		Value: "0s",
		Usage: "add a random delay in [0s,delay] to avoid synchronized registration",
	}
	flManageAdvertise = cli.StringFlag{
		Name:   "advertise, addr",
		Usage:  "Address of the swarm manager joining the cluster. Other swarm manager(s) MUST be able to reach the swarm manager at this address.",
		EnvVar: "SWARM_ADVERTISE",
	}
	// hack for go vet
	flHostsValue = cli.StringSlice([]string{"tcp://127.0.0.1:2375"})

	flHosts = cli.StringSliceFlag{
		Name:   "host, H",
		Value:  &flHostsValue,
		Usage:  "ip/socket to listen on",
		EnvVar: "SWARM_HOST",
	}
	flHeartBeat = cli.StringFlag{
		Name:  "heartbeat",
		Value: "60s",
		Usage: "period between each heartbeat",
	}
	flTTL = cli.StringFlag{
		Name:  "ttl",
		Value: "180s",
		Usage: "set the expiration of an ephemeral node",
	}
	flTimeout = cli.StringFlag{
		Name:  "timeout",
		Value: "10s",
		Usage: "timeout period",
	}
	flRefreshIntervalMin = cli.StringFlag{
		Name:  "engine-refresh-min-interval",
		Value: "30s",
		Usage: "set engine refresh minimum interval",
	}
	flRefreshIntervalMax = cli.StringFlag{
		Name:  "engine-refresh-max-interval",
		Value: "60s",
		Usage: "set engine refresh maximum interval",
	}
	flRefreshRetry = cli.IntFlag{
		Name:  "engine-refresh-retry",
		Value: 3,
		Usage: "deprecated; replaced by --engine-failure-retry",
	}
	flFailureRetry = cli.IntFlag{
		Name:  "engine-failure-retry",
		Value: 3,
		Usage: "set engine failure retry count",
	}
	flEnableCors = cli.BoolFlag{
		Name:  "api-enable-cors, cors",
		Usage: "enable CORS headers in the remote API",
	}
	flTLS = cli.BoolFlag{
		Name:  "tls",
		Usage: "use TLS; implied by --tlsverify=true",
	}
	flTLSCaCert = cli.StringFlag{
		Name:  "tlscacert",
		Usage: "trust only remotes providing a certificate signed by the CA given here",
	}
	flTLSCert = cli.StringFlag{
		Name:  "tlscert",
		Usage: "path to TLS certificate file",
	}
	flTLSKey = cli.StringFlag{
		Name:  "tlskey",
		Usage: "path to TLS key file",
	}
	flTLSVerify = cli.BoolFlag{
		Name:  "tlsverify",
		Usage: "use TLS and verify the remote",
	}
	flStrategy = cli.StringFlag{
		Name:  "strategy",
		Usage: "placement strategy to use [" + strings.Join(strategy.List(), ", ") + "]",
		Value: strategy.List()[0],
	}

	// hack for go vet
	flFilterValue = cli.StringSlice(filter.List())
	// DefaultFilterNumber is exported
	DefaultFilterNumber = len(flFilterValue)

	flFilter = cli.StringSliceFlag{
		Name:  "filter, f",
		Usage: "filter to use [" + strings.Join(filter.List(), ", ") + "]",
		Value: &flFilterValue,
	}

	flCluster = cli.StringFlag{
		Name:  "cluster-driver, c",
		Usage: "cluster driver to use [swarm, mesos-experimental]",
		Value: "swarm",
	}
	flClusterOpt = cli.StringSliceFlag{
		Name:  "cluster-opt",
		Usage: "cluster driver options",
		Value: &cli.StringSlice{},
	}
	flDiscoveryOpt = cli.StringSliceFlag{
		Name:  "discovery-opt",
		Usage: "discovery options",
		Value: &cli.StringSlice{},
	}
	flLeaderElection = cli.BoolFlag{
		Name:  "replication",
		Usage: "Enable Swarm manager replication",
	}
	flLeaderTTL = cli.StringFlag{
		Name:  "replication-ttl",
		Value: "20s",
		Usage: "Leader lock release time on failure",
	}
)
