package quota

import (
	"os"
	"bytes"
	"encoding/json"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"	
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"net/http"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/samalba/dockerclient"
	"net/http/httptest"
	"github.com/gorilla/mux"
)

var enforceQuota = os.Getenv("SWARM_ENFORCE_QUOTA")
var quotaMgmt QuotaMgmt

type DefaultQuotaImpl struct {
	nextHandler pluginAPI.Handler
}

func NewQuota(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	log.Debug("Plugin Quota NewQuota")
	quotaPlugin := &DefaultQuotaImpl{
		nextHandler: handler,
	}
	return quotaPlugin
}

func (quotaImpl *DefaultQuotaImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin Quota got command: " + command)
	if(enforceQuota != "true") {
		log.Debug("Quota NOT enforced!")
		swarmHandler.ServeHTTP(w, r)
		return nil
	}
	
	//initialize quota once
	initQuota := false
	if initQuota == false{
		quotaMgmt.Init(cluster)
		initQuota = true
	}
	
	switch command {
	case "containercreate":
		defer r.Body.Close()
		if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {
			var containerConfig dockerclient.ContainerConfig
			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
				return err
			}
			memory := containerConfig.HostConfig.Memory
			tenant := r.Header.Get(headers.AuthZTenantIdHeaderName)
			// Increase tenant quota usage if quota limit isn't exceeded.
			err := quotaMgmt.CheckAndIncreaseQuota(tenant, memory) 
			if err != nil {
				log.Error(err)
				return err
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))
			rec := httptest.NewRecorder()
			
			swarmHandler.ServeHTTP(rec, r)	
			/*POST Swarm*/
			w.WriteHeader(rec.Code)
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.Write(rec.Body.Bytes())
			//only if create container succeeded - add container
			err = quotaMgmt.HandleCreateResponse(rec.Code,rec.Body.Bytes(), tenant, memory) //checks that createContainer succeeded
			if err != nil {
				log.Error(err)
				return err
			}	
		}
	case "containerdelete":
		resourceLongID := mux.Vars(r)["name"]
		tenant := r.Header.Get(headers.AuthZTenantIdHeaderName)
		//on delete request - decrease resource usage for the tenant in quotaService and set quota container status to PENDING_DELETED
		quotaMgmt.DecreaseQuota(resourceLongID, tenant)			
		rec := httptest.NewRecorder()
		swarmHandler.ServeHTTP(rec, r)	
		//if delete response is OK delete the container
		quotaMgmt.HandleDeleteResponse(rec.Code, resourceLongID, tenant)	
	default:	
		swarmHandler.ServeHTTP(w, r)	
	}
	return nil
}
























