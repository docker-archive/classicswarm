package keystone

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/authZ/states"
	"github.com/jeffail/gabs"
	//	"github.com/docker/swarm/pkg/authZ"

	"github.com/docker/swarm/pkg/authZ/headers"
	"github.com/docker/swarm/pkg/authZ/utils"
	"github.com/samalba/dockerclient"
)

type KeyStoneAPI struct{ quotaAPI QuotaAPI }

var cacheAPI *Cache

var configs *Configs

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

type AuthenticationResponse struct {
	access TokenResponseData
}

type TokenResponseData struct {
	issued_at string
	expires   string
	id        string
}

func (this *KeyStoneAPI) Init() error {
	cacheAPI = new(Cache)
	cacheAPI.Init()
	configs = new(Configs)
	configs.ReadConfigurationFormfile()
	this.quotaAPI = new(QuotaImpl)
	this.quotaAPI.Init()
	return nil
}

//TODO - May want to sperate concenrns
// 1- Validate Token
// 2- Get ACLs or Lable for your valid token
// 3- Set up cache to save Keystone call
func (this *KeyStoneAPI) ValidateRequest(cluster cluster.Cluster, eventType states.EventEnum, w http.ResponseWriter, r *http.Request, reqBody []byte, containerConfig dockerclient.ContainerConfig) (states.ApprovalEnum, *utils.ValidationOutPutDTO) {
//func (this *KeyStoneAPI) ValidateRequest(cluster cluster.Cluster, eventType states.EventEnum, r *http.Request, containerConfig dockerclient.ContainerConfig) (states.ApprovalEnum, *utils.ValidationOutPutDTO) {
	log.Debug("ValidateRequest Keystone")
	log.Debugf("%+v\n",containerConfig)
	tokenToValidate := r.Header.Get(headers.AuthZTokenHeaderName)
	tokenToValidate = strings.TrimSpace(tokenToValidate)
	tenantIdToValidate := strings.TrimSpace(r.Header.Get(headers.AuthZTenantIdHeaderName))

	log.Debugf("Going to validate token:  %v, for tenant Id: %v, ", tokenToValidate, tenantIdToValidate)
//	valid := queryKeystone(tenantIdToValidate, tokenToValidate)

//	if !valid {
//		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
//	}
	
	if os.Getenv("SWARM_MULTI_TENANT") == "KEYSTONE_AUTH" && !(queryKeystone(tenantIdToValidate, tokenToValidate)) {
		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	}

	if isAdminTenant(tenantIdToValidate) {
		return states.Admin, nil
	}

	switch eventType {
	case states.ContainerCreate:
		err := this.quotaAPI.ValidateQuota(cluster, reqBody, tenantIdToValidate)
		if err != nil {
			return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: err.Error()}
		}
		valid, dto := utils.CheckLinksOwnerShip(cluster, tenantIdToValidate, containerConfig)
		log.Debug(valid)
		log.Debug(dto)
		log.Debug("-----------------")
		if !valid {
			return states.NotApproved, dto			
		} 
		return states.Approved, dto
	case states.ContainersList:
		return states.ConditionFilter, nil
	case states.Unauthorized:
		return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
	default:
		//CONTAINER_INSPECT / CONTAINER_OTHERS / STREAM_OR_HIJACK / PASS_AS_IS
		isOwner, dto := utils.CheckOwnerShip(cluster, tenantIdToValidate, r)
		if isOwner {
			return states.Approved, dto
		}
	}
	log.Debug("SHOULD NOT BE HERE....")
	return states.NotApproved, &utils.ValidationOutPutDTO{ErrorMessage: "Not Authorized!"}
}

func isAdminTenant(tenantIdToValidate string) bool {
	log.Info("isAdminTenant(" + tenantIdToValidate + ")")
	swarmAdminTenantId := os.Getenv("SWARM_ADMIN_TENANT_ID")
	log.Debug("SWARM_ADMIN_TENANT_ID: " + swarmAdminTenantId)
	log.Debug("isAdminTenant: ", swarmAdminTenantId == tenantIdToValidate)
	return swarmAdminTenantId == tenantIdToValidate
}

//SHORT CIRCUIT KEYSTONE
//func queryKeystone(tenantIdToValidate string, tokenToValidate string) bool {
//	return true
//}

func queryKeystone(tenantIdToValidate string, tokenToValidate string) bool {
	var headers = map[string]string{
		headers.AuthZTokenHeaderName: tokenToValidate,
	}
	resp := doHTTPreq("GET", configs.GetConf().KeystoneUrl+"tenants", "", headers)
	defer resp.Body.Close()
	log.Debug("response Status:", resp.Status)
	log.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("response Body:", string(body))
	if 200 != resp.StatusCode {
		return false
	}

	jsonParsed, _ := gabs.ParseJSON(body)
	children, _ := jsonParsed.S("tenants").Children()
	var isSwarmMember bool = false
	var swarmMembersTenantId string = os.Getenv("SWARM_MEMBERS_TENANT_ID")
	if swarmMembersTenantId == "" {
		log.Info("SWARM_MEMBERS_TENANT_ID is blank")
		isSwarmMember = true
	}
	var isTenantFound bool = false
	if tenantIdToValidate != swarmMembersTenantId {
		for i := 0; i < len(children); i++ {
			if children[i].Path("id").Data().(string) == tenantIdToValidate {
				isTenantFound = true
			} else if !isSwarmMember {
				if children[i].Path("id").Data().(string) == swarmMembersTenantId {
					isSwarmMember = true
				}
			}
			if isTenantFound && isSwarmMember {
				log.Info("isTenantFound and isSwarmMember are true")
				break
			}
		}
	} else {
		log.Debug("error: Tenant trying use SWARM_MEMBERS_TENANT_ID")
	}
	if !(isTenantFound && isSwarmMember) {
		log.Debug("error: Tenant not eligible")
	}
	return isTenantFound && isSwarmMember
}
