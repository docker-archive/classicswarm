package quota

import (
	"errors"
	"os"
	log "github.com/Sirupsen/logrus"
)

type QuotaService struct {
	limit ResourceList
	used ResourceList
}

type ResourceList struct { 
	memory  int64 
	network int64 
	cpu     float64 
}

var enforceQuotaService = os.Getenv("SWARM_ENFORCE_QUOTA")
var quotasMgmt = make(map[string]QuotaService)
var DEFAULT_MEMORY_QUOTA int64 = 1024 * 1024 * 300 //300MB for all tenants
var DEFAULT_MEMORY int64 = 1024 * 1024 * 64        //64MB

//Server will check if RQTenant.used+delta > RQTenant.limit, and if not, it will update RQShard.used and RQTenant.used using CAS 
//to update tenant quota in the backend storage
func (quota *QuotaService) UpdateRQUsed(TenantID string,FrameworkID string, ShardID string, delta ResourceList)error{
	tenantQuota, ok := quotasMgmt[TenantID]
	if ok {		
		if (tenantQuota.used.memory + delta.memory <= tenantQuota.limit.memory ) {
			log.Debug("Quota Service::UpdateRQUsed tenant ", TenantID," current quota usage= ",tenantQuota.used.memory," quota limit= ", tenantQuota.limit.memory )
    		// Existing tenant, increase usage
    		tenantQuota.used.memory = tenantQuota.used.memory + delta.memory
    		quotasMgmt[TenantID] = tenantQuota
			log.Debug("Quota Service::UpdateRQUsed tenant ", TenantID," after increase/decrease quota usage= ",tenantQuota.used.memory)
			return nil
		}else{ //not enough memory
			return errors.New("Quota Service::UpdateRQUsed Tenant memory quota limit reached!")
		}
	}else{// New tenant
		if (delta.memory <= DEFAULT_MEMORY_QUOTA ){
			tenantQuota.limit.memory = DEFAULT_MEMORY_QUOTA
			log.Debug("Quota Service::UpdateRQUsed New Tenant ", TenantID," quota limit= ", tenantQuota.limit.memory )
			tenantQuota.used.memory = tenantQuota.used.memory + delta.memory
        	quotasMgmt[TenantID] = tenantQuota
        	log.Debug("Quota Service::UpdateRQUsed tenant ", TenantID," after increase/decrease quota usage= ",tenantQuota.used.memory)
        	return nil
		}else{
			return errors.New("Quota Service::UpdateRQUsed New Tenant memory quota limit reached!")
		}
	}	
}

//Returns RQTenant.limit, RQTenant.used, and RQTenant[FrameworkID][ShardID].used
func (quota *QuotaService) GetRQ(TenantID string, FrameworkID string, ShardID string) (limit_ret ResourceList, used_ret ResourceList,shardUsed *ResourceList, err error){
	var limit,used ResourceList
	if tenantQuota, ok := quotasMgmt[TenantID]; ok {
		limit.memory = tenantQuota.limit.memory

		used.memory = tenantQuota.used.memory
	}else {
		log.Debug("Quota Service::GetRQ tenant Not exist")
		limit.memory = -1
		used.memory = -1
		return limit, used, nil, nil
	}
	return limit, used, nil, nil
} 

//Server will overwrite RQTenant.limit with limit. If there are concurrent requests, last one wins
func (quota *QuotaService) SetRQLimit(TenantID string, limit *ResourceList){
	if tenantQuota, ok := quotasMgmt[TenantID]; ok {
		tenantQuota.limit.memory = limit.memory
		log.Debug("Quota Service::SetRQLimit tenant ",TenantID,"memory limit: ", limit.memory)
		quotasMgmt[TenantID] = tenantQuota
	}else {// New tenant
		tenantQuota.limit.memory = limit.memory
		log.Debug("Quota Service::SetRQLimit New tenant ",TenantID,"memory limit: ", limit.memory)
        quotasMgmt[TenantID] = tenantQuota
	}	
}

//Init - Any required initialization
func (quota *QuotaService) Init() error {
	if enforceQuotaService != "true" {
		log.Debug("Tenant quota is not enforced.")
		return nil
	}
	return nil
}

