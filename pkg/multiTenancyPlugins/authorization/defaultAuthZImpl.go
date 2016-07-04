package authorization

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/samalba/dockerclient"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
)

type DefaultAuthZImpl struct {
	nextHandler pluginAPI.Handler
}

func NewAuthorization(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	authZ := &DefaultAuthZImpl{
		nextHandler: handler,
	}
	return authZ
}

func (defaultauthZ *DefaultAuthZImpl) Handle(command utils.CommandEnum, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin AuthZ got command: " + command)
	switch command {
	case "containerscreate":

		defer r.Body.Close()
		if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {
			var containerConfig dockerclient.ContainerConfig
			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
				return err
			}
			//Disallow a user to create the special labels we inject : headers.TenancyLabel
			res := strings.Contains(string(reqBody), headers.TenancyLabel)
			if res == true {
				errorMessage := "Error, special label " + headers.TenancyLabel + " disallowed!"
				return errors.New(errorMessage)
			}
			containerConfig.Labels[headers.TenancyLabel] = r.Header.Get(headers.AuthZTenantIdHeaderName)

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
				return err
			}

			r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")

		}
		return defaultauthZ.nextHandler(command, cluster, w, r, swarmHandler)
		log.Debug("Returned from Swarm")
		//In case of container json - should record and clean - consider seperating..
	case "containerstart", "containerstop", "containerrestart", "containerdelete", "containerwait", "containerarchive", "containerkill", "containerpause", "containerunpause", "containerupdate", "containercopy", "containerchanges", "containerattach", "containerlogs", "containertop":
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
		return defaultauthZ.nextHandler(command, cluster, w, r, swarmHandler)
	case "containerjson":
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}

		rec := httptest.NewRecorder()
		if err := defaultauthZ.nextHandler(command, cluster, rec, r, swarmHandler); err != nil {
			return err
		}
		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case "containersjson", "containersps":
		//TODO - clean up code
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
		if err := defaultauthZ.nextHandler(command, cluster, rec, newReq, swarmHandler); err != nil {
			return err
		}
		//TODO - May decide to overrideSwarms handlers.getContainersJSON - this is Where to do it.
		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case "listNetworks":
		rec := httptest.NewRecorder()
		if err := defaultauthZ.nextHandler(command, cluster, rec, r, swarmHandler); err != nil {
			return err
		}
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := utils.FilterNetworks(r, rec)
		w.Write(newBody)

	case "info", "createNetwork":

		return defaultauthZ.nextHandler(command, cluster, w, r, swarmHandler)

	//Always allow or not?
	default:
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
	}
	return nil
}
