package swarm

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
)

// parseClusterOpts parses cluster opts to fill some fields of swarm.Cluster
func parseClusterOpts(options cluster.DriverOpts, cl *Cluster) {
	// parse swarm.overcommit in cluster-opt
	if val, err := options.Float("swarm.overcommit", ""); err != nil {
		// run into error when parsing swarm.overcommit
		if err == cluster.ErrNoKeyNorEnv {
			// swarm.overcommit is not provided, just move on
			log.Debug("swarm.overcommit is not provided in cluster options")
		} else {
			log.Fatalf("Failed to parse swarm.overcommit in Float (%v)", err)
		}
	} else {
		//validate val to be a valid overcommit
		if val <= float64(-1) {
			log.Fatalf("swarm.overcommit should be larger than -1, %f is invalid", val)
		} else if val < float64(0) {
			log.Warn("-1 < swarm.overcommit < 0 will make swarm take less resource than docker engine offers")
			cl.overcommitRatio = val
		} else {
			cl.overcommitRatio = val
		}
	}

	// parse swarm.createretry in cluster-opt
	if val, err := options.Int("swarm.createretry", ""); err != nil {
		// run into error when parsing swarm.createretry
		if err == cluster.ErrNoKeyNorEnv {
			// swarm.createretry is not provided, just move on
			log.Debug("swarm.createretry is not provided in cluster options")
		} else {
			log.Fatalf("Failed to parse swarm.createretry in Int (%v)", err)
		}
	} else {
		//validate val to be a valid createRetry
		if val < 0 {
			log.Fatalf("swarm.createretry can not be negative, %d is invalid", val)
		}
		cl.createRetry = val
	}
}
