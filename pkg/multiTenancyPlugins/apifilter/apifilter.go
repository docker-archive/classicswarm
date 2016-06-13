package apifilter

import (
    log "github.com/Sirupsen/logrus"
	"errors"
	"encoding/json"
	"net/http"
	"os"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/cluster"

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

var apiDisabledMap map[string]bool

func init() {
	log.Info("apifliter.init()")
	readApiFilter()
}

func readApiFilter() {
	log.Info("apifilter.readApiFilter() ..........")
	type Filter struct {
		Disableapi []string		
	}
	var filter Filter
	var f = os.Getenv("SWARM_APIFILTER_FILE")
	if f == "" {
		log.Info("Missing SWARM_API_SUPPORTED_FILE environment variable, using locate default ./apifilter.json")
		f = "apifilter.json"
	}
	log.Info("SWARM_APIFILTER_FILE: ",f)

	file, err := os.Open(f)
	if err != nil {
		log.Info("no API FILTER")
		return
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&filter)
	if err != nil {
		log.Fatal("Error in apifilter decode:", err)
		panic("Error: could not decode apifilter.json")
	}
	log.Infof("filter %+v",filter)
	apiDisabledMap = make(map[string]bool)
	for _,e := range filter.Disableapi {
		if apiImplementedMap[e] {
			apiDisabledMap[e] = true
		}
		
	}
	log.Infof("apiDisabledMap %+v",apiDisabledMap)
}

func (apiFilterImpl *DefaultApiFilterImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin apiFilter Got command: " + command)
	if apiImplementedMap[command] && !apiDisabledMap[command] {
		return apiFilterImpl.nextHandler(command, cluster, w, r, swarmHandler)		
	} else {
		return errors.New("Command Not Supported!")		
	}
	
}

 