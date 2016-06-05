package namescoping

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	apitypes "github.com/docker/engine-api/types"
	log "github.com/Sirupsen/logrus"
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

func uniqlyIdentfyResource(cluster cluster.Cluster, r *http.Request, w http.ResponseWriter) {
	resourceName := mux.Vars(r)["name"]
	tenantId := r.Header.Get(headers.AuthZTenantIdHeaderName)
Loop:
	for _, container := range cluster.Containers() {
		if container.Info.ID == resourceName {
			//Match by Full Id - Do nothing
			break
		} else {
			for _, name := range container.Names {
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
func (nameScoping *DefaultNameScopingImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin nameScoping Got command: " + command)
	switch command {
	case "containercreate":
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
				containerConfig.Labels[headers.OriginalNameLabel] = r.URL.Query().Get("name")

				if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
					return err
				}

				r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), newQuery, "")
			}
		}
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)

	//Find the container and replace the name with ID
	case "containerjson":
		if resourceName := mux.Vars(r)["name"]; resourceName != "" {
			uniqlyIdentfyResource(cluster, r, w)
			return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
		}
	case "containerstart", "containerstop", "containerdelete", "containerkill", "containerpause", "containerunpause", "containerupdate", "containercopy", "containerattach", "containerlogs":

		uniqlyIdentfyResource(cluster, r, w)
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
	case "createNetwork":		
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
	case "listContainers", "listNetworks", "clusterInfo":
		return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
	default:

	}
	return nil
}
