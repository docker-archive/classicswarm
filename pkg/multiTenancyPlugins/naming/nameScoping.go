package namescoping

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	apitypes "github.com/docker/engine-api/types"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

//AuthenticationImpl - implementation of plugin API
type DefaultNameScopingImpl struct {
	nextHandler pluginAPI.Handler
}

func NewNameScoping(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	nameScoping := &DefaultNameScopingImpl{
		nextHandler: handler,
	}
	return nameScoping
}

func uniquelyIdentifyContainer(cluster cluster.Cluster, r *http.Request, w http.ResponseWriter) {
	resourceName := mux.Vars(r)["name"]
	tenantId := r.Header.Get(headers.AuthZTenantIdHeaderName)
Loop:
	for _, container := range cluster.Containers() {
		if container.Info.ID == resourceName {
			//Match by Full Id - Do nothing
			break
		} else {
			for _, name := range container.Names {
				name := strings.TrimPrefix(name, "/")
				if (resourceName == name || resourceName == container.Labels[headers.OriginalNameLabel]) && container.Labels[headers.TenancyLabel] == tenantId {
					//Match by Name - Replace to full ID
					mux.Vars(r)["name"] = container.Info.ID
					r.URL.Path = strings.Replace(r.URL.Path, resourceName, container.Info.ID, 1)
					break Loop
				}
			}
		}
		if strings.HasPrefix(container.Info.ID, resourceName) {
			mux.Vars(r)["name"] = container.Info.ID
			r.URL.Path = strings.Replace(r.URL.Path, resourceName, container.Info.ID, 1)
			break
		}
	}
}

//Handle authentication on request and call next plugin handler.
func (nameScoping *DefaultNameScopingImpl) Handle(command utils.CommandEnum, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin nameScoping Got command: " + command)
	switch command {
	case utils.CONTAINER_CREATE:
		if "" != r.URL.Query().Get("name") {
			defer r.Body.Close()
			if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {
				var newQuery string
				var buf bytes.Buffer
				var containerConfig dockerclient.ContainerConfig

				if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
					return err
				}

				log.Debug("Postfixing name with tenantID...")
				newQuery = strings.Replace(r.RequestURI, r.URL.Query().Get("name"), r.URL.Query().Get("name")+r.Header.Get(headers.AuthZTenantIdHeaderName), 1)
				//Disallow a user to create the special labels we inject : headers.OriginalNameLabel
				res := strings.Contains(string(reqBody), headers.OriginalNameLabel)
				if res == true {
					errorMessage := "Error, special label " + headers.OriginalNameLabel + " disallowed!"
					return errors.New(errorMessage)
				}
				containerConfig.Labels[headers.OriginalNameLabel] = r.URL.Query().Get("name")

				if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
					return err
				}

				r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), newQuery, "")
			}
		}
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)

	//Find the container and replace the name with ID
	case utils.CONTAINER_JSON:
		if resourceName := mux.Vars(r)["name"]; resourceName != "" {
			uniquelyIdentifyContainer(cluster, r, w)
			return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
		} else {
			log.Debug("What now?")
		}
	case utils.CONTAINER_START, utils.CONTAINER_STOP, utils.CONTAINER_RESTART, utils.CONTAINER_DELETE, utils.CONTAINER_WAIT, utils.CONTAINER_ARCHIVE, utils.CONTAINER_KILL, utils.CONTAINER_PAUSE, utils.CONTAINER_UNPAUSE, utils.CONTAINER_UPDATE, utils.CONTAINER_COPY, utils.CONTAINER_CHANGES, utils.CONTAINER_ATTACH, utils.CONTAINER_LOGS, utils.CONTAINER_TOP, utils.CONTAINER_STATS:
		uniquelyIdentifyContainer(cluster, r, w)
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
	case utils.NETWORK_CREATE:
		defer r.Body.Close()
		if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {

			var request apitypes.NetworkCreate
			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&request); err != nil {
				log.Error(err)
				return nil
			}
			request.Name = r.Header.Get(headers.AuthZTenantIdHeaderName) + request.Name
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(request); err != nil {
				log.Error(err)
				return nil
			}
			r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")
		}
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
	case utils.PS, utils.JSON, utils.NETWORKS_LIST, utils.INFO, utils.EVENTS, utils.IMAGES_JSON:
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
	default:

	}
	return nil
}
