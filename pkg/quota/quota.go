package quota

import (
	"errors"
	"os"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

type Quota struct {
	tenantMemoryLimit int64
	tenantMemoryAvailable int64
	containers map[string]int64
}

var quotas = make(map[string]Quota)
var enforceQuota = os.Getenv("SWARM_ENFORCE_QUOTA")
var DEFAULT_MEMORY_QUOTA int64 = 1024 * 1024 * 300 //300MB for all tenants
var DEFAULT_MEMORY int64 = 1024 * 1024 * 64        //64MB


/*
ValidateQuota - checks if tenant quota satisfies container create request
*/
func (*Quota) ValidateQuota(resource int64, tenant string) error {
	if enforceQuota != "true" {
		log.Debug("Tenant quota is not enforced.")
		return nil
	}
	if resource == 0{
		log.Debugf("Parsed memory limit is 0, using default memory limit of %vB", DEFAULT_MEMORY)
		resource = DEFAULT_MEMORY
	}
	if tenantQuota, ok := quotas[tenant]; ok {
    	// Existing tenant
    	log.Debug("Existing tenant")
    	log.Debug("Current action memory add: ", int64(resource))
		tenantQuota.tenantMemoryAvailable = tenantQuota.tenantMemoryAvailable - resource
		quotas[tenant] = tenantQuota
		log.Debug("New available: ", tenantQuota.tenantMemoryAvailable)
	}else{// New tenant
		log.Debug("New tenant")
		tenantQuota.tenantMemoryLimit = DEFAULT_MEMORY_QUOTA
		tenantQuota.tenantMemoryAvailable = DEFAULT_MEMORY_QUOTA - resource
		tenantQuota.containers = make(map[string]int64)
        quotas[tenant] = tenantQuota
	}
	
	//sanaty check
	for key, value := range quotas {
	    log.Debug("Tenant: ", key, " Quota Limit: ", value.tenantMemoryLimit," Available: ", value.tenantMemoryAvailable)
	}

	if (quotas[tenant].tenantMemoryAvailable < 0) {
		// need temp var. bug in go. https://github.com/golang/go/issues/3117
		revertQuota := quotas[tenant]
		revertQuota.tenantMemoryAvailable = revertQuota.tenantMemoryAvailable + resource
		quotas[tenant]= revertQuota
		return errors.New("Tenant memory quota limit reached!")
	}
	return nil
}

/*
UpdateQuota - consum or free resources
*/
func (*Quota) UpdateQuota(tenant string, toFree bool, id string, memory int64) error {
	log.Debug("Update resources allocation for tenant: ", tenant)
	if tenantQuota, ok := quotas[tenant]; ok {
		//sanaty
		for key, value := range tenantQuota.containers {
	    	log.Debug("Container: ", key, " Memory: ", value)
		}
		if id != ""{
			// get memory
			memory = tenantQuota.containers[id]
		}
		if toFree{
			log.Debug("Free resources. Current availible: ", tenantQuota.tenantMemoryAvailable)
			tenantQuota.tenantMemoryAvailable = tenantQuota.tenantMemoryAvailable + memory
			removeContainer(id, tenant)
		} else{
			tenantQuota.tenantMemoryAvailable = tenantQuota.tenantMemoryAvailable - memory
		}
		
		log.Debug("New availible: ", tenantQuota.tenantMemoryAvailable)
		quotas[tenant] = tenantQuota
	} else {
		log.Debug("No quota exists for this tenant")
	}
	return nil
}

/*
AddContainer - add new container to tenant list of container
*/
func (*Quota) AddContainer(resource int64, id string, tenant string) {
	log.Debug("Add container to quota containers list")
	if tenantQuota, ok := quotas[tenant]; ok {
		tenantQuota.containers[id] = resource
	}
}

/*
removeContainer - remove container from tenant list of container
*/
func removeContainer(id string, tenant string) {
	if tenantQuota, ok := quotas[tenant]; ok {
		delete(tenantQuota.containers, id)
	}
}

//Init - Any required initialization
func (*Quota) Init() error {
	return nil
}

func init() {
 	log.Debug("Init quota")
 	setDefault()
}

/*
setDefault - set default quota limit
*/
func setDefault() {
 	if enforceQuota != "true" {
 		log.Info("Tenant quota is not enforced.")
 		return
 	}
 	var quotaFile = os.Getenv("SWARM_QUOTA_FILE")
 	if quotaFile == "" {
 		quotaFile = "quota.json"
 	}
 	file, err := os.Open(quotaFile)
 	if err != nil {
 		return // using hardcoded values
 	}
 	//parse json to get the value
 	type DefaultQuota struct {
		Memory int64
	}
	decoder := json.NewDecoder(file)
	var defaultQuota DefaultQuota
 	err = decoder.Decode(&defaultQuota)
 	if err != nil {
 		return // using hardcoded values
 	}
 	DEFAULT_MEMORY_QUOTA = 1024 * 1024 * defaultQuota.Memory
 	log.Info("Memory from file: ", DEFAULT_MEMORY_QUOTA)	
}

