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

func (defaultauthZ *DefaultAuthZImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin AuthZ got command: " + command)
	switch command {
	case "containercreate":

		defer r.Body.Close()
		if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {
			var containerConfig dockerclient.ContainerConfig
			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
				return err
			}
			containerConfig.Labels[headers.TenancyLabel] = r.Header.Get(headers.AuthZTenantIdHeaderName)

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
				return err
			}

			r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")

			//			defaultauthZ.nextHandler("containerCreate", cluster, w, r, swarmHandler)
		}
		swarmHandler.ServeHTTP(w, r)
		log.Debug("Returned from Swarm")

		//In case of container json - should record and clean - consider seperating..
	case "containerstart", "containerstop", "containerdelete":
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
		swarmHandler.ServeHTTP(w, r)

	case "containerjson":
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
		rec := httptest.NewRecorder()
		swarmHandler.ServeHTTP(rec, r)
		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case "listContainers":
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

		//TODO - May decide to overrideSwarms handlers.getContainersJSON - this is Where to do it.
		swarmHandler.ServeHTTP(rec, newReq)

		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := utils.CleanUpLabeling(r, rec)

		w.Write(newBody)

	case "listNetworks":
		rec := httptest.NewRecorder()
		swarmHandler.ServeHTTP(rec, r)

		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := utils.FilterNetworks(r, rec)
		w.Write(newBody)

	//Always allow or not?
	default:
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
	}
	return nil
}
