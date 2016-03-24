package mesos

import (
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	mesosscheduler "github.com/mesos/mesos-go/scheduler"
)

// parseDriveConfig parses cluster opts to fill some fields of mesosscheduler.DriverConfig
func parseDriveConfig(options cluster.DriverOpts, driverConfig *mesosscheduler.DriverConfig) {
	if bindingPort, err := options.Uint("mesos.port", "SWARM_MESOS_PORT"); err != nil {
		if err == cluster.ErrNoKeyNorEnv {
			log.Debug("no mesos.port in cluster-opts nor SWARM_MESOS_PORT in env")
		} else {
			log.Fatalf("Failed to parse mesos.port in Uint (%v)", err)
		}
	} else {
		// validate bindingPort to be a valid BindingPort
		if uint16(bindingPort) == 0 {
			log.Fatal("mesos.port cannot be 0")
		}
		driverConfig.BindingPort = uint16(bindingPort)
	}

	if bindingAddress, err := options.String("mesos.address", "SWARM_MESOS_ADDRESS"); err != nil {
		log.Debug("no mesos.address in cluster-opts nor SWARM_MESOS_ADDRESS in env")
	} else {
		if ipAddress := net.ParseIP(bindingAddress); ipAddress == nil {
			log.Fatalf("invalid IP address for cluster-opt mesos.address: %s", bindingAddress)
		} else {
			driverConfig.BindingAddress = ipAddress
		}
	}

	if checkpointFailover, err := options.Bool("mesos.checkpointfailover", "SWARM_MESOS_CHECKPOINT_FAILOVER"); err != nil {
		if err == cluster.ErrNoKeyNorEnv {
			log.Debug("no mesos.checkpointfailover in cluster-opts nor SWARM_MESOS_CHECKPOINT_FAILOVER in env")
		} else {
			log.Fatalf("Failed to parse mesos.checkpointfailover in Bool (%v)", err)
		}
	} else {
		driverConfig.Framework.Checkpoint = &checkpointFailover
	}
}

// parseClusterOpts parses cluster-opt to fill some fields of mesos.Cluster
func parseClusterOpts(options cluster.DriverOpts, cl *Cluster) {
	if taskCreationTimeout, err := options.String("mesos.tasktimeout", "SWARM_MESOS_TASK_TIMEOUT"); err != nil {
		if err == cluster.ErrNoKeyNorEnv {
			log.Debug("no mesos.tasktimeout in cluster-opts nor SWARM_MESOS_TASK_TIMEOUT in env")
		}
	} else {
		d, err := time.ParseDuration(taskCreationTimeout)
		if err != nil {
			log.Fatalf("Failed to parse mesos.tasktimeout in Duration (%v)", err)
		}
		// validate d to be a valid taskCreationTimeout
		if d < time.Duration(0)*time.Second {
			log.Fatalf("mesos.taskCreationTimeout cannot be a negative number")
		}
		cl.taskCreationTimeout = d
	}

	if offerTimeout, err := options.String("mesos.offertimeout", "SWARM_MESOS_OFFER_TIMEOUT"); err != nil {
		if err == cluster.ErrNoKeyNorEnv {
			log.Debug("no mesos.offertimeout in cluster-opts nor SWARM_MESOS_OFFER_TIMEOUT in env")
		}
	} else {
		d, err := time.ParseDuration(offerTimeout)
		if err != nil {
			log.Fatalf("Failed to parse mesos.offertimeout in Duration (%v)", err)
		}
		// validate d to be a valid offerTimeout
		if d < time.Duration(0)*time.Second {
			log.Fatalf("mesos.offerTimeout cannot be a negative number")
		}
		cl.offerTimeout = d
	}

	if refuseTimeout, err := options.String("mesos.offerrefusetimeout", "SWARM_MESOS_OFFER_REFUSE_TIMEOUT"); err != nil {
		if err == cluster.ErrNoKeyNorEnv {
			log.Debug("no mesos.offerrefusetimeout in cluster-opts nor SWARM_MESOS_OFFER_REFUSE_TIMEOUT in env")
		}
	} else {
		d, err := time.ParseDuration(refuseTimeout)
		if err != nil {
			log.Fatalf("Failed to parse mesos.offerrefusetimeout in Duration (%v)", err)
		}
		// validate d to be a valid offerrefusetimeout
		if d < time.Duration(0)*time.Second {
			log.Fatalf("mesos.offerrefusetimeout cannot be a negative number")
		}
		cl.refuseTimeout = d
	}
}
