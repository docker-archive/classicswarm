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
/*
func (*DefaultImp) HandleEvent(eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, dto *utils.ValidationOutPutDTO, reqBody []byte) {
	switch eventType {
	case states.ContainerCreate:
		log.Debug("In create...")
		log.Debug("Old body: " + string(reqBody))

		//TODO - Here we just use the token for the tenant name for now
		newBody := bytes.Replace(reqBody, []byte("{"), []byte("{\"Labels\": {\""+headers.TenancyLabel+"\":\""+r.Header.Get(headers.AuthZTenantIdHeaderName)+"\"},"), 1)
		for cId, _ := range dto.Links {
			log.Debug(cId)
			cName := dto.Links[cId]
			log.Debug("cName:" + cName + "#")
			log.Debug("cId:" + cId + "#")
			newBody = bytes.Replace(newBody, []byte(cName+" :"), []byte(cId+":"), -1)
			newBody = bytes.Replace(newBody, []byte(cName+":"), []byte(cId+":"), -1)
		}

//		for cId, _ := range dto.VolumesFrom {
//			log.Debug(cId)
//			cName := dto.VolumesFrom[cId]
//			log.Debug("2cName:" + cName + "#")
//			log.Debug("2cId:" + cId + "#")
//			newBody = bytes.Replace(newBody, []byte(cName+" :"), []byte(cId+":"), -1)
//			newBody = bytes.Replace(newBody, []byte(cName+":"), []byte(cId+":"), -1)
//		}

		log.Debug("New body: " + string(newBody))

		var newQuery string
		if "" != r.URL.Query().Get("name") {
			log.Debug("Postfixing name with Label...")
			newQuery = strings.Replace(r.RequestURI, r.URL.Query().Get("name"), r.URL.Query().Get("name")+r.Header.Get(headers.AuthZTenantIdHeaderName), 1)
			log.Debug(newQuery)
		}

		newReq, e1 := utils.ModifyRequest(r, bytes.NewReader(newBody), newQuery, "")
		if e1 != nil {
			log.Error(e1)
		}
		next.ServeHTTP(w, newReq)

	case states.ContainerInspect:
		log.Debug("In inspect...")
		rec := httptest.NewRecorder()

		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], dto.ContainerID, 1)
		mux.Vars(r)["name"] = dto.ContainerID
		next.ServeHTTP(rec, r)

		//POST Swarm
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

		//POST Swarm
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case states.ContainerOthers:
		log.Debug("In others...")
		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], dto.ContainerID, 1)
		mux.Vars(r)["name"] = dto.ContainerID

		next.ServeHTTP(w, r)

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
*/

func (*DefaultImp) HandleEvent(eventType states.EventEnum, w http.ResponseWriter, r *http.Request, next http.Handler, dto *utils.ValidationOutPutDTO, reqBody []byte, containerConfig dockerclient.ContainerConfig) {
	log.Debugf("HandleEvent %+v\n", eventType)
	switch eventType {
	case states.ContainerCreate:
		log.Debug("In create...")
		log.Debugf("containerConfig In: %+v\n", containerConfig)
		containerConfig.Labels[headers.TenancyLabel] = r.Header.Get(headers.AuthZTenantIdHeaderName)
		containerConfig.HostConfig.VolumesFrom = dto.VolumesFrom
		containerConfig.HostConfig.Links = dto.Links
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
		next.ServeHTTP(w, newReq)

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
			next.ServeHTTP(w, r)
		}
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
