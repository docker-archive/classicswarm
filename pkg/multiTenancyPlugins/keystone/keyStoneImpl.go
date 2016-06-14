package keystone

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"errors"
	"encoding/json"


	log "github.com/Sirupsen/logrus"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/cluster"

)

type KeyStoneAPI struct{}

type DefaultApiFilterImpl struct {
	nextHandler pluginAPI.Handler
}
func NewPlugin(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	apiKeystonePlugin := &DefaultApiFilterImpl{
		nextHandler: handler,
	}
	return apiKeystonePlugin
}
var keystoneUrl string
func (apiKeystoneImpl *DefaultApiFilterImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin keystone got command: " + command)
	if os.Getenv("SWARM_AUTH_BACKEND") != "Keystone" {
		return apiKeystoneImpl.nextHandler(command, cluster, w, r, swarmHandler)
	}
	if keystoneUrl = os.Getenv("SWARM_KEYSTONE_URL"); keystoneUrl == "" {
		log.Fatal("Error in SWARM_KEYSTONE_URL not set")
		panic("Error: Error in SWARM_KEYSTONE_URL not set")
	}

	tokenToValidate := r.Header.Get(headers.AuthZTokenHeaderName)
	tokenToValidate = strings.TrimSpace(tokenToValidate)
	tenantIdToValidate := strings.TrimSpace(r.Header.Get(headers.AuthZTenantIdHeaderName))

	log.Debugf("Going to validate token:  %v, for tenant Id: %v, ", tokenToValidate, tenantIdToValidate)
	valid := queryKeystone(tenantIdToValidate, tokenToValidate)

	if !valid {
		return errors.New("Not Authorized!")
	}
	
	if isAdminTenant(tenantIdToValidate) {
		swarmHandler.ServeHTTP(w, r)
		return nil
	}
	return apiKeystoneImpl.nextHandler(command, cluster, w, r, swarmHandler)
	
}


func isAdminTenant(tenantIdToValidate string) bool {
	log.Info("isAdminTenant(" + tenantIdToValidate + ")")
	swarmAdminTenantId := os.Getenv("SWARM_ADMIN_TENANT_ID")
	log.Debug("SWARM_ADMIN_TENANT_ID: " + swarmAdminTenantId)
	log.Debug("isAdminTenant: ", swarmAdminTenantId == tenantIdToValidate)
	return swarmAdminTenantId == tenantIdToValidate
}

func doHTTPreq(reqType, url, jsonBody string, headers map[string]string) *http.Response {
	var req *http.Request = nil
	var err error = nil
	if "" != jsonBody {
		byteStr := []byte(jsonBody)
		data := bytes.NewBuffer(byteStr)
		req, err = http.NewRequest(reqType, url, data)
	} else {
		req, err = http.NewRequest(reqType, url, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		for k, v := range headers {
			log.Debug("k: " + k + ", v: " + v)
			req.Header.Set(k, v)
		}
		log.Debug("jsonBody: " + jsonBody)
		log.Debug("url: " + url)
		
		panic(err)
	}
	return resp
}


func queryKeystone(tenantIdToValidate string, tokenToValidate string) bool {
	log.Debug("in queryKeystone")
	var headers = map[string]string{
		headers.AuthZTokenHeaderName: tokenToValidate,
	}
	 
	var listTenantsResponse map[string]interface{}
	resp := doHTTPreq("GET", keystoneUrl+"tenants", "", headers)
	defer resp.Body.Close()
	log.Debug("response Status:", resp.Status)
	log.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("response Body:", string(body))
	if 200 != resp.StatusCode {
		return false
	}
	var isSwarmMember bool = false
	var swarmMembersTenantId string = os.Getenv("SWARM_MEMBERS_TENANT_ID")
	if swarmMembersTenantId == "" {
		log.Info("SWARM_MEMBERS_TENANT_ID is blank")
		isSwarmMember = true
	}
	var isTenantFound bool = false
	if tenantIdToValidate != swarmMembersTenantId {
		err := json.Unmarshal(body,&listTenantsResponse)
		if err != nil {
			log.Fatal("Error in Keystone List Tenant decode:", err)
			panic("Error: could not decode Keystone List Tenant ")
		}
		for _,v := range listTenantsResponse["tenants"].([]interface{}) {
			id := v.(map[string]interface{})["id"]
			log.Infof("tenants id",id)
			if id == tenantIdToValidate {
				isTenantFound = true
			} else if id == swarmMembersTenantId {
				isSwarmMember = true
			}
			if isTenantFound && isSwarmMember {
				log.Info("isTenantFound and isSwarmMember are true")
				break
			}
			
		}
	} else {
		log.Debug("error: Tenant trying to use SWARM_MEMBERS_TENANT_ID")
	}
	if !(isTenantFound && isSwarmMember) {
		log.Debug("error: Tenant not eligible")
	}
	return isTenantFound && isSwarmMember
}
