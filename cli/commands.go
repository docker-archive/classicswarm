package cli

import "github.com/codegangsta/cli"

var (
	commands = []cli.Command{
		{
			Name:      "create",
			ShortName: "c",
			Usage:     "Create a cluster",
			Action:    create,
		},
		{
			Name:      "list",
			ShortName: "l",
			Usage:     "List nodes in a cluster",
			Flags:     []cli.Flag{flTimeout, flDiscoveryOpt},
			Action:    list,
		},
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "Manage a docker cluster",
			Flags: []cli.Flag{
				flStrategy, flFilter,
				flHosts,
				flLeaderElection, flLeaderTTL, flManageAdvertise,
				flTLS, flTLSCaCert, flTLSCert, flTLSKey, flTLSVerify,
				flRefreshIntervalMin, flRefreshIntervalMax, flFailureRetry, flRefreshRetry,
				flHeartBeat,
				flEnableCors,
				flCluster, flDiscoveryOpt, flClusterOpt},
			Action: manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "Join a docker cluster",
			Flags:     []cli.Flag{flJoinAdvertise, flHeartBeat, flTTL, flJoinRandomDelay, flDiscoveryOpt},
			Action:    join,
		},
	}
)
