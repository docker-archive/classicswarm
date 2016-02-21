package flavors

import (
    log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
	"encoding/json"
	"os"
)
type Flavor struct {
	Memory            int64
	MemoryReservation int64
	MemorySwap        int64
	KernelMemory      int64
	CpuShares         int64
	CpuPeriod         int64
	CpuQuota          int64
	BlkioWeight       int64
	OomKillDisable    bool
	MemorySwappiness  int64
	Privileged        bool
	ReadonlyRootfs    bool
}
var flavors map[string]Flavor 
var flavorsEnforced = os.Getenv("SWARM_FLAVORS_ENFORCED")

func init() {
	log.Info("flavors.init()")
	readFlavorFile()

} 
func readFlavorFile() {
	log.Info("Flavors.ReadFlavorFile() ..........")
	if (flavorsEnforced == "false") {
		log.Info("Flavors not enforced")
		return
	}
	var flavorsFile = os.Getenv("SWARM_FLAVORS_FILE")
	if flavorsFile == "" {
		log.Warn("Missing SWARM_FLAVORS_FILE environment variable, using locate default ./flavors.json")
		flavorsFile = "flavors.json"
	}

	file, err := os.Open(flavorsFile)
	if err != nil {
		log.Fatal(err)
		panic("Error: could not open flavorsFile ")
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&flavors)
	if err != nil {
		log.Fatal("Error in flavorsFile decode:", err)
		panic("Error: could not decode flavorsFile ")
	}
	log.Infof("Flavors %+v",flavors)
}

func IsFlavorValid(containerConfig dockerclient.ContainerConfig) (bool) {
	log.Debug("isFlavorValid")
	//log.Debugf("flavors: %+v",flavors)
	if(flavorsEnforced == "false") {
		return true
	}
	var flavorIn Flavor
	flavorIn.Memory = containerConfig.HostConfig.Memory
	flavorIn.MemorySwap = containerConfig.HostConfig.MemorySwap
	flavorIn.KernelMemory = containerConfig.HostConfig.KernelMemory
	flavorIn.CpuShares = containerConfig.HostConfig.CpuShares
	flavorIn.CpuPeriod = containerConfig.HostConfig.CpuPeriod
	flavorIn.CpuQuota = containerConfig.HostConfig.CpuQuota
	flavorIn.BlkioWeight = containerConfig.HostConfig.BlkioWeight
	flavorIn.OomKillDisable = containerConfig.HostConfig.OomKillDisable
	flavorIn.MemorySwappiness = containerConfig.HostConfig.MemorySwappiness
	flavorIn.Privileged = containerConfig.HostConfig.Privileged
	flavorIn.ReadonlyRootfs = containerConfig.HostConfig.ReadonlyRootfs
	for key, value := range flavors {
		log.Debugf("key: %s value: %+v",key,value)
		if(value == flavorIn) {
			log.Debugf("flavor: %s found",key)
			return true			
		}
	}
	return false
	
}
