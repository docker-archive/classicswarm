package authorization

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	//	"os"
	//	"errors"
	//"container/list"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/samalba/dockerclient"
	//	"github.com/docker/swarm/pkg/authZ/keystone"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/utils"
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
	case "containerjson", "containerstart", "containerstop", "containerdelete":

		if utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			swarmHandler.ServeHTTP(w, r)
			log.Debug("Returned from Swarm")
		}
	case "listContainers":
		//record to clean up host names and labeling etc..

	//Always allow or not?
	default:
		if !utils.IsOwner(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r) {
			return errors.New("Not Authorized!")
		}
	}
	return nil
}
