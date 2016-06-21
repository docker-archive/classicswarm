package quota

import (
	"errors"
	"os"
	"time"
	"sync"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"github.com/samalba/dockerclient"
	"bytes"
)

type QuotaMgmt struct {
	sync.RWMutex
	containers map[string]ContainerInfo //map of containers by id
}

type State int64
// The list of of all possible "enum" values
const (
  NONE State = iota
  PENDING_DELETED  
)

type ContainerInfo struct {
	Status State
	Memory int64
}

var quotaService QuotaService
var quotas = make(map[string]QuotaMgmt)
var dec int64 = -1

//check limit and increase quota
func (quota *QuotaMgmt) CheckAndIncreaseQuota(tenant string, memory int64) error{
	if enforceQuota != "true" {
		return nil
	}
	var qp = ResourceList{memory:memory}
	_, used_ret , _,_ := quotaService.GetRQ(tenant,"","")
	log.Debug("Quota::CheckAndIncreaseQuota tenant ",tenant," used quota was= ",used_ret.memory)
	//check limit and increase quota
	err := quotaService.UpdateRQUsed(tenant,"","",qp)
	if err != nil {
			return err
	}
	_, used_ret , _,_ = quotaService.GetRQ(tenant,"","")
	log.Debug("Quota::CheckAndIncreaseQuota tenant ",tenant," used quota now= ",used_ret.memory)
	return nil
}

//if create container succeeded add container to swarm containers map. if failed decrease quota
func (quota *QuotaMgmt) HandleCreateResponse(returnCode int, body []byte, tenant string, memory int64) error{
	if enforceQuota != "true" {
		return nil
	}
	decreaseMem := memory*dec //only decrease in this method
	var qp = ResourceList{memory:decreaseMem}
	if returnCode!=201 {
		err := quotaService.UpdateRQUsed(tenant,"","",qp)	//Decrease quota
		_, used_ret , _,_ := quotaService.GetRQ(tenant,"","")
		log.Debug("Quota::HandleCreateResponse failed tenant ",tenant,"decreased used quota = ",used_ret.memory)
		if err != nil {
			return err
		}
		log.Error(err)
		return errors.New("Swarm Quota::HandleCreateResponse Create Container failed")
	}
	var containerConfig dockerclient.ContainerInfo
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&containerConfig); err != nil {
		//create response failed - container was NOT created on engine	-> decrease quota
	    err2 := quotaService.UpdateRQUsed(tenant,"","",qp)	//Decrease quota
	    _, used_ret , _,_ := quotaService.GetRQ(tenant,"","")
		log.Debug("Quota::HandleCreateResponse failed tenant ",tenant,"decreased used quota = ",used_ret.memory)
	    if err2 != nil {
				return err2
		}
	}
	
	id := containerConfig.Id
	
	if tenantQuota, ok := quotas[tenant]; ok {
		if contInfo, ok := tenantQuota.containers[id]; ok{	//in case of delete event before create response arrived
			if contInfo.Status == PENDING_DELETED {		//if container exists in swarm containers map with status PENDING_DELETED, decrease quota and delete quota container
				quotaMgmt.Lock()
				err := quotaService.UpdateRQUsed(tenant,"","",qp) 	//Decrease quota
				if err != nil {
					return err
				}
				_, used_ret , _,_ := quotaService.GetRQ(tenant,"","")
				log.Debug("Quota::HandleCreateResponse (container with PENDING_DELETED)  tenant ",tenant," decreased used quota = ",used_ret.memory)
				quotaMgmt.DeleteContainer(tenant, memory, id)		//delete container
				quotaMgmt.Unlock()
				return nil
			}
		}
	}
	quotaMgmt.AddContainer(tenant , memory, id, NONE) // add container to map
	
	for key, value := range quotas {
		quotaLimits, resUsage, _,_ := quotaService.GetRQ(tenant,"","")
	    log.Debug("Quota::HandleCreateResponse Tenant: ", key, " Quota Limit: ", quotaLimits.memory," resUsage: ", resUsage.memory)
	    for keycont, valuecont := range value.containers{
	    	log.Debug("Quota::HandleCreateResponse container id: ",keycont," memory cont: ",valuecont.Memory, " status= ",valuecont.Status)
	    }
	}
	return nil
}

//add swarm quota container
func (quota *QuotaMgmt) AddContainer(tenant string, memory int64, container string, status State){
	var contInfo ContainerInfo
	if tenantQuota, ok := quotas[tenant]; ok {
		contInfo.Memory = memory
		contInfo.Status = status	
		tenantQuota.containers[container] = contInfo
		log.Debug("Swarm Quota::AddContainer id= ",container," memory= ", memory," status= ",status," to tenant ",tenant)
		quotas[tenant] = tenantQuota
	    
	}else{ //crete new tenant
		contInfo.Memory = memory
		contInfo.Status = status
		tenantQuota.containers = make(map[string]ContainerInfo)
		tenantQuota.containers[container] = contInfo
		log.Debug("Swarm Quota::AddContainer id= ",container," memory= ", memory," status= ",status," to tenant ",tenant)
		quotas[tenant] = tenantQuota
	}	
}

//delete swarm quota container
func (quota *QuotaMgmt) DeleteContainer(tenant string, memory int64, container string){
	if tenantQuota, ok := quotas[tenant]; ok {
			delete(tenantQuota.containers, container)
			log.Debug("Swarm Quota::DeleteContainer id= ",container," memory= ", memory," tenant ",tenant)
	}	
}

func (quota *QuotaMgmt) IsSwarmContainer(cluster cluster.Cluster,id string, tenant string) error{
	containers := (cluster).Containers()
	for _, container := range containers{	
		if container.Info.ID == id{
			swarm := container.Config.Labels[headers.SwarmLabel]
			if swarm !="" {	
				return nil
			}
		}
		return errors.New("Not Swarm container!")
	}
	return errors.New("container Not exists in cluster!")
}


//on delete request - decrease resource usage for the tenant in quotaService and set quota container status to PENDING_DELETED
func (quota *QuotaMgmt) DecreaseQuota(id string, tenant string) bool{
	if enforceQuota != "true" {
		return false
	}
	var qp ResourceList
	if tenantQuota, ok := quotas[tenant]; ok {
		if contInfo, ok := tenantQuota.containers[id]; ok{	
			qp.memory = tenantQuota.containers[id].Memory*dec //decrease memory
			quotaMgmt.Lock()
			_, used_ret , _,_ := quotaService.GetRQ(tenant,"","")
				log.Debug("Quota::DecreaseQuota tenant ",tenant,"used quota was = ",used_ret.memory)
			quotaService.UpdateRQUsed(tenant,"","",qp)
			_, used_ret , _,_ = quotaService.GetRQ(tenant,"","")
				log.Debug("Quota::DecreaseQuota tenant ",tenant,"used quota is = ",used_ret.memory)
			contInfo.Status = PENDING_DELETED
			tenantQuota.containers[id] = contInfo
			quotas[tenant] = tenantQuota	
			quotaMgmt.Unlock()
			log.Debug("Swarm Quota::DecreaseQuota set container id = ",id," to status PENDING_DELETED memory= ",qp.memory)
			return true
		}
	}
	//if no such id-add it with status PENDING_DELETED - for case of delete request in the middle of periodic loop
	//quotaMgmt.AddContainer(tenant , memory, id, PENDING_DELETED)
		
	return true
}

//if delete response is OK delete the container
func (quota *QuotaMgmt) HandleDeleteResponse(returnCode int, id string, tenant string) {
	if 200 <= returnCode && returnCode < 300 { //response OK
		if tenantQuota, ok := quotas[tenant]; ok {
			memory := tenantQuota.containers[id].Memory
			quotaMgmt.DeleteContainer(tenant, memory, id)
		}
	}
	//sanaty check
	for key, value := range quotas {
		quotaLimits, resUsage, _,_ := quotaService.GetRQ(key,"","")
	    log.Debug("Quota::HandleDeleteResponse Tenant: ", key, " Quota Limit: ", quotaLimits.memory," resUsage: ", resUsage.memory)
	    for keycont, valuecont := range value.containers{
	    	log.Debug("Quota::HandleDeleteResponse container id: ",keycont," memory cont: ",valuecont.Memory, " status= ",valuecont.Status)
	    }
	}
}

//quota periodic refresh from cluster
func (quota *QuotaMgmt) refreshLoop(cluster cluster.Cluster) {
	for {
		time.Sleep(time.Second *30)
		quotaMgmt.Lock()
		var clusterContInfo ContainerInfo
		//containers represents a list a containers
		containers := (cluster).Containers()
		
		var clusterQuotas = make(map[string]QuotaMgmt) // map of cluster containers
		var tenantsDeltaMem = make(map[string]int64) // map of tenants and delta memory to add/subtruct
		//add missing cluster containers to quoata
		for _, container := range containers{	
			
			tenant := container.Config.Labels[headers.TenancyLabel]		
			if tenant !="" {
				//populating clusterQuotas with cluster containers for later use
				if tenantClusterQuota, ok := clusterQuotas[tenant]; ok {
					clusterContInfo.Memory = container.Config.HostConfig.Memory
					tenantClusterQuota.containers[container.Info.ID] = clusterContInfo
					clusterQuotas[tenant] = tenantClusterQuota
				}else{//create new tenant for cluster containers map
					tenantClusterQuota.containers = make(map[string]ContainerInfo)
					clusterContInfo.Memory = container.Config.HostConfig.Memory
					tenantClusterQuota.containers[container.Info.ID] = clusterContInfo
					clusterQuotas[tenant] = tenantClusterQuota
				}
				
				if tenantQuota, ok := quotas[tenant]; ok {
					//if cluster container id with no status doesn't exist on quota containers add container id to quota and add container memory to quotaContMemory
					_, okId := tenantQuota.containers[container.Info.ID]
					//if container id not exist in quota
					if !okId { //not in PENDING_DELETED state
						quotaMgmt.AddContainer(tenant , container.Config.HostConfig.Memory, container.Info.ID, NONE)
						tenantsDeltaMem[tenant] = tenantsDeltaMem[tenant] + container.Config.HostConfig.Memory
					}
				}else{ //New tenant
					quotaMgmt.AddContainer(tenant , container.Config.HostConfig.Memory, container.Info.ID, NONE)
					tenantsDeltaMem[tenant] = tenantsDeltaMem[tenant] + container.Config.HostConfig.Memory
				}	
			}
		}	
		//delete quota containers which are missing on cluster containers
		quotaMgmt.HandleMissingQuotaCont(clusterQuotas, tenantsDeltaMem)
		quotaMgmt.Unlock()
	}
}

//remove quotas containers with NONE status which are in quotas but not in cluster containers or tenant
func (quota *QuotaMgmt) HandleMissingQuotaCont(clusterQuotas map[string]QuotaMgmt, tenantsDeltaMem map[string]int64) {
	var qp ResourceList
	var tenantClusterMemory int64
	//remove quotas containers with NONE status which are in quotas but not in cluster containers or tenant
	for key, value := range quotas {	//for all quota tenants
		for keycont, valuecont := range value.containers{	//for all quota tenant containers
			if valuecont.Status == NONE { //with status NONE - not PENDING_DELETED
				if tenantclusterQuota, ok := clusterQuotas[key]; ok {
					if _, ok := tenantclusterQuota.containers[keycont]; !ok{
						tenantsDeltaMem[key] = tenantsDeltaMem[key] - valuecont.Memory
						quotaMgmt.DeleteContainer(key, valuecont.Memory, keycont)
					}	
				}else{ //cluster tenant doesn't exist - delete quota containers and quota tenant
					tenantsDeltaMem[key] = tenantsDeltaMem[key] - valuecont.Memory
					quotaMgmt.DeleteContainer(key, valuecont.Memory, keycont)
					//should i delete tenant???
				}
			}else {		//delete quota containers with PENDING_DELETED status
				quotaMgmt.DeleteContainer(key, valuecont.Memory, keycont) //delete container id with PENDING_DELETED status
			}
		}
	}
	//updating quota service tenant memory according to clusters containers
	if len(clusterQuotas) == 0 {
		for key, _ := range quotas {
			_, used_ret , _,_ := quotaService.GetRQ(key,"","")
			if used_ret.memory != 0 {
				qp.memory = used_ret.memory*-1
				err := quotaService.UpdateRQUsed(key,"","",qp)
				if err != nil {
					log.Error(err)
				}	
			}
		}
	}
	
	for key, _ := range clusterQuotas {
		if tenantclusterQuota, ok := clusterQuotas[key]; ok {
			for _, valcont := range tenantclusterQuota.containers {
				tenantClusterMemory = tenantClusterMemory + valcont.Memory
			}
			_, used_ret , _,_ := quotaService.GetRQ(key,"","")
			if tenantClusterMemory != used_ret.memory { 
				qp.memory = tenantClusterMemory - used_ret.memory //increase or decrease delta memory
				err := quotaService.UpdateRQUsed(key,"","",qp)
				if err != nil {
					log.Error(err)
				}	
			}
			tenantClusterMemory = 0	
		}
	}
}

//Init - Any required initialization
func (quota *QuotaMgmt) Init(cluster cluster.Cluster) error {	
	if enforceQuota != "true" {
 		return nil
 	}
	go quotaMgmt.refreshLoop(cluster) //periodic refresh loop
	
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
