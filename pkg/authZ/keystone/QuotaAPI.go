package keystone

import (
	"github.com/docker/swarm/cluster"
)

//QuotaAPI - API for quota management. Currently only memory support
type QuotaAPI interface {

	//Do we need any kind of init? Like load defaults from config file?
	Init() error

	//Validates if the tenant quota valid. Should return error?
	ValidateQuota(myCluster cluster.Cluster, reqBody []byte, tenant string) error
}
