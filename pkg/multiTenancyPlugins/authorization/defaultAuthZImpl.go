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
	log.Debug(command)
	switch command {
	case "containerCreate":
		defer r.Body.Close()
		reqBody, _ := ioutil.ReadAll(r.Body)
		if len(reqBody) != 0 {
			var containerConfig dockerclient.ContainerConfig
			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
				return err
			}
			containerConfig.Labels[headers.TenancyLabel] = r.Header.Get(headers.AuthZTenantIdHeaderName)

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
				return err
			}

			r, _ := utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")

			swarmHandler.ServeHTTP(w, r)
			log.Debug("Returned from Swarm")
			//			defaultauthZ.nextHandler("containerCreate", cluster, w, r, swarmHandler)
		}

	case "containerInspect":
		isOwner, _ := utils.CheckOwnerShip(cluster, r.Header.Get(headers.AuthZTenantIdHeaderName), r)
		if isOwner {
			swarmHandler.ServeHTTP(w, r)
			log.Debug("Returned from Swarm")
		}

	default:
		tenant := r.Header.Get(headers.AuthZTenantIdHeaderName)
		isOwner, _ := utils.CheckOwnerShip(cluster, tenant, r)
		if !isOwner {
			return errors.New("Not Authorized!")
		}
	}
	return nil
}
