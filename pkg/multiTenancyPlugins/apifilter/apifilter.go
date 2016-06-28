package apifilter

import (
	"errors"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
)

type DefaultApiFilterImpl struct {
	nextHandler pluginAPI.Handler
}

func NewPlugin(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	apiFilterPlugin := &DefaultApiFilterImpl{
		nextHandler: handler,
	}
	return apiFilterPlugin
}

type Apifilter struct{}

func (apiFilterImpl *DefaultApiFilterImpl) Handle(command utils.CommandEnum, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin apiFilter Got command: " + command)
	if supportedAPIsMap[command] {
		return apiFilterImpl.nextHandler(command, cluster, w, r, swarmHandler)
	} else {
		return errors.New("Command Not Supported!")
	}

}
