package authZ

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"net/http/httptest"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/pkg/authZ/states"
	//	"github.com/docker/swarm/pkg/authZ/keystone"
	"regexp"
       "github.com/docker/swarm/pkg/quota"
	"github.com/docker/swarm/pkg/authZ/headers"
	"github.com/docker/swarm/pkg/authZ/utils"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

//DefaultImp - Default basic label based implementation of ACLs & tenancy enforcment
type DefaultImp struct{}

//Init - Any required initialization
func (*DefaultImp) Init() error {
	return nil
}

//HandleEvent - Implement approved operation - Default labels based implmentation
func (*DefaultImp) HandleEvent(eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, dto *utils.ValidationOutPutDTO, reqBody []byte, containerConfig dockerclient.ContainerConfig, volumeCreateRequest dockerclient.VolumeCreateRequest) {
	log.Debugf("defaultAuthZImpl.HandleEvent %+v\n", eventType)
	quota := new(quota.Quota)
	switch eventType {
	case states.ContainerCreate:
		log.Debug("In create...")

        memory := containerConfig.HostConfig.Memory
		tenant := r.Header.Get(headers.AuthZTenantIdHeaderName)
		// validate that quota limit isn't exceeded.
		err := quota.ValidateQuota(memory, tenant)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debugf("containerConfig In: %+v\n", containerConfig)
		containerConfig.Labels[headers.TenancyLabel] = r.Header.Get(headers.AuthZTenantIdHeaderName)
		containerConfig.HostConfig.VolumesFrom = dto.VolumesFrom
		containerConfig.HostConfig.Links = dto.Links
		containerConfig.HostConfig.Binds = dto.Binds
		containerConfig.Env = dto.Env
		log.Debugf("containerConfig Out: %+v\n", containerConfig)

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
			log.Error(err)
			return
		}

		var newQuery string
		if "" != r.URL.Query().Get("name") {
			log.Debug("Postfixing name with Label...")
			newQuery = strings.Replace(r.RequestURI, r.URL.Query().Get("name"), r.URL.Query().Get("name")+r.Header.Get(headers.AuthZTenantIdHeaderName), 1)
			log.Debug(newQuery)
		}

		newReq, e1 := utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), newQuery, "")
		if e1 != nil {
			log.Error(e1)
		}
		//next.ServeHTTP(w, newReq)
              rec := httptest.NewRecorder()
		next.ServeHTTP(rec, newReq)	
		freeResources(quota, rec.Body.Bytes(), tenant, memory, "", 0)		
		// copy everything from recorder to writer
              w.WriteHeader(rec.Code)
        	for k, v := range rec.HeaderMap {
            		w.Header()[k] = v
        	}
        	rec.Body.WriteTo(w)

	case states.ContainerInspect:
		log.Debug("In inspect...")
		rec := httptest.NewRecorder()

		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], dto.ContainerID, 1)
		mux.Vars(r)["name"] = dto.ContainerID
		next.ServeHTTP(rec, r)

		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := utils.CleanUpLabeling(r, rec)
		w.Write(newBody)

	case states.ContainersList:
		log.Debug("In list...")
		var v = url.Values{}
		mapS := map[string][]string{"label": {headers.TenancyLabel + "=" + r.Header.Get(headers.AuthZTenantIdHeaderName)}}
		filterJSON, _ := json.Marshal(mapS)
		v.Set("filters", string(filterJSON))
		var newQuery string
		if strings.Contains(r.URL.RequestURI(), "?") {
			newQuery = r.URL.RequestURI() + "&" + v.Encode()
		} else {
			newQuery = r.URL.RequestURI() + "?" + v.Encode()
		}
		log.Debug("New Query: ", newQuery)

		newReq, e1 := utils.ModifyRequest(r, nil, newQuery, "")
		if e1 != nil {
			log.Error(e1)
		}
		rec := httptest.NewRecorder()

		//TODO - May decide to overrideSwarms handlers.getContainersJSON - this is Where to do it.
		next.ServeHTTP(rec, newReq)

		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case states.ContainerOthers:
		log.Debug("In others...")
		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], dto.ContainerID, 1)

		if strings.Contains(r.URL.Path, "rename") {
			re := regexp.MustCompile(`/containers/(.*)/rename\?name=(.*)`)
			arr := re.FindStringSubmatch(r.URL.RequestURI())
			nameParam := arr[2]
			newQuery := strings.Replace(r.URL.RequestURI(), nameParam, nameParam+r.Header.Get(headers.AuthZTenantIdHeaderName), 1)
			newReq, e1 := utils.ModifyRequest(r, nil, newQuery, "")
			if e1 != nil {
				log.Error(e1)
			}
			mux.Vars(r)["name"] = dto.ContainerID
			next.ServeHTTP(w, newReq)
		} else {
			mux.Vars(r)["name"] = dto.ContainerID
			//next.ServeHTTP(w, r)
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
		    	// tenant resources update for quota enforcement
			if strings.Contains(r.Method, "DELETE") { 
				freeResources(quota, nil, r.Header.Get(headers.AuthZTenantIdHeaderName), 0, dto.ContainerID, rec.Code)
			}
			// copy everything from recorder to writer
              		w.WriteHeader(rec.Code)
        		for k, v := range rec.HeaderMap {
            			w.Header()[k] = v
        		}
        		rec.Body.WriteTo(w)
		}
	case states.VolumeCreate:
		log.Debug("event: VolumeCreate...")
		log.Debugf("volumeCreateRequest In: %+v\n", volumeCreateRequest)
		if volumeCreateRequest.Name == "" {
			volumeCreateRequest.Name = utils.RandStringBytesRmndr(20)
		}
		volumeCreateRequest.Name = volumeCreateRequest.Name + r.Header.Get(headers.AuthZTenantIdHeaderName) 
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(volumeCreateRequest); err != nil {
			log.Error(err)
			return
		}
		newReq, e1 := utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")
		if e1 != nil {
			log.Error(e1)
		}

		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, newReq)
		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)
		

	case states.VolumesList:
		log.Debug("event: VolumesList...")
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		/*POST Swarm*/
		log.Debug("after swarm")
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		var volumesListResponse dockerclient.VolumesListResponse 
		var volumesListResponseOut dockerclient.VolumesListResponse
		var volumesOut []*dockerclient.Volume
		//responseBody, _ := ioutil.ReadAll(rec.Body)
		responseBody := rec.Body.Bytes()
		if len(responseBody)!= 0 {
			if err := json.NewDecoder(bytes.NewReader(responseBody)).Decode(&volumesListResponse); err != nil {
               log.Error(err)
                return
        		}				
			log.Debugf("volumesListResponse: %+v",volumesListResponse)
			volumes := volumesListResponse.Volumes
			tenantId := r.Header.Get(headers.AuthZTenantIdHeaderName)
			for _, v := range volumes {
				if strings.Contains(v.Name,tenantId) {
					v.Name = strings.Replace(v.Name,tenantId,"",1)
					volumesOut = append(volumesOut,v)					
				}
			}
			for _, v := range volumesOut {
				log.Debugf("volumeOut volume : %+v",v)
			}
			volumesListResponseOut.Volumes = volumesOut
			log.Debugf("volumesListResponseOut: %+v",volumesListResponseOut)
 
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(volumesListResponseOut); err != nil {
			log.Error(err)
			return
		}
		w.Write(buf.Bytes())

	case states.VolumeInspect:
		log.Debug("event: VolumeInspect...")
		volumeNameIndex := strings.Index(r.RequestURI,"/volumes/")+len("/volumes/")
		volumeName := r.RequestURI[volumeNameIndex:len(r.RequestURI)] + r.Header.Get(headers.AuthZTenantIdHeaderName)
		log.Debugf(" volumeName *%s*", volumeName)
		r.URL.Path = r.RequestURI[0:volumeNameIndex] + volumeName
		mux.Vars(r)["volumename"] =  volumeName
		log.Debugf(" mux.Vars(r) %+v", mux.Vars(r))
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := utils.CleanUpLabeling(r, rec)
		w.Write(newBody)

	case states.VolumeRemove:
		log.Debug("event: VolumeRemove...")
		volumeNameIndex := strings.LastIndex(r.RequestURI,"/")+1
		volumeName := r.RequestURI[volumeNameIndex:len(r.RequestURI)] + r.Header.Get(headers.AuthZTenantIdHeaderName)
		newPath := r.RequestURI[0:volumeNameIndex] + volumeName
		log.Debug(" newReq: ", newPath)
		r.URL.Path = newPath
		log.Debug(" url: ", r.URL)
		mux.Vars(r)["name"] =  volumeName
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		/*POST Swarm*/
		log.Debug("Post Swarm")
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := utils.CleanUpLabeling(r, rec)
		w.Write(newBody)
		
		
		//TODO - hijack and others are the same because we handle no post and no stream manipulation and no handler override yet
	case states.StreamOrHijack:
		log.Debug("In stream/hijack...")
		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], dto.ContainerID, 1)
		mux.Vars(r)["name"] = dto.ContainerID
		next.ServeHTTP(w, r)

	case states.PassAsIs:
		log.Debug("Forwarding the request AS IS...")
		next.ServeHTTP(w, r)
	case states.Unauthorized:
		log.Debug("In UNAUTHORIZED...")
	default:
		log.Debug("In default...")
	}
}

/*
freeResources - free resources for tenant quota enforcement.
*/
func freeResources(quota *quota.Quota, body []byte, tenant string, memory int64, containerId string, returnCode int) {
	if (containerId == "") && (body != nil){
		id, err := utils.ParseID(body)
		if err != nil {
			log.Error(err)
			// free resources
			quota.UpdateQuota(tenant, true, "", memory)
			return
		}
		log.Info("Add to quota container ID: ", id)
		quota.AddContainer(memory, id, tenant)
		return
	}	
	if 200 <= returnCode && returnCode < 300 {
		log.Debugf("containerId: %+v\n", containerId)
		// free resources
		quota.UpdateQuota(tenant, true, containerId, 0)
	}			
}
