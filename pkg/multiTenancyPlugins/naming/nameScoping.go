package namescoping

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/headers"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/utils"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
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

//Handle authentication on request and call next plugin handler.
func (nameScoping *DefaultNameScopingImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin nameScoping Got command: " + command)
	switch command {
	case "containerCreate":
		//It is not optimal to do this without condition
		defer r.Body.Close()
		reqBody, _ := ioutil.ReadAll(r.Body)
		if "" != r.URL.Query().Get("name") && len(reqBody) > 0 {
			var newQuery string
			var buf bytes.Buffer
			var containerConfig dockerclient.ContainerConfig

			if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
				return err
			}

			if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
				return err
			}

			log.Debug("Postfixing name with tenantID...")
			newQuery = strings.Replace(r.RequestURI, r.URL.Query().Get("name"), r.URL.Query().Get("name")+r.Header.Get(headers.AuthZTenantIdHeaderName), 1)
			containerConfig.Labels[headers.OriginalNameLabel] = r.URL.Query().Get("name")

			r, _ := utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), newQuery, "")
			return nameScoping.nextHandler(command, cluster, w, r, swarmHandler)
		}

	//Find the container and replace the name with ID
	case "containerInspect":
		resourceName := mux.Vars(r)["name"]
		tenantId := r.Header.Get(headers.AuthZTenantIdHeaderName)
		for _, container := range cluster.Containers() {
			if container.Info.ID == resourceName {
				//Match by Full Id - Do nothing
				return nil
			}
			for _, name := range container.Names {
				if (resourceName == name || resourceName == container.Labels[headers.OriginalNameLabel]) && container.Labels[headers.TenancyLabel] == tenantId {
					//Match by Name - Replace to full ID
					mux.Vars(r)["name"] = container.Info.ID
					r.URL.Path = strings.Replace(r.URL.Path, resourceName, container.Info.ID, 1)
					return nil
				}
			}
			//Handle short Id - What if we do nothing?
		}
	default:

	}
	return nil
}
