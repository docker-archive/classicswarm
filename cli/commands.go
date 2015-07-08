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
			Flags:     []cli.Flag{flTimeout},
			Action:    list,
		},
		{
			Name:      "manage",
			ShortName: "m",
			Usage:     "Manage a docker cluster",
			Flags: []cli.Flag{
				flStore,
				flStrategy, flFilter,
				flHosts,
				flLeaderElection, flManageAdvertise, flLeaderPathPrefix,
				flTLS, flTLSCaCert, flTLSCert, flTLSKey, flTLSVerify,
				flHeartBeat,
				flEnableCors,
				flCluster, flClusterOpt},
			Action: manage,
		},
		{
			Name:      "join",
			ShortName: "j",
			Usage:     "join a docker cluster",
			Flags:     []cli.Flag{flJoinAdvertise, flHeartBeat, flTTL},
			Action:    join,
		},
	}
)
