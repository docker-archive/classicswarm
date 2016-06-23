package flavors

import (
    log "github.com/Sirupsen/logrus"
	"encoding/json"
	"bytes"
	"os"
	"net/http"
	"io/ioutil"
	"github.com/samalba/dockerclient"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/pluginAPI"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
	"github.com/docker/swarm/cluster"

)
type DefaultFlavorsImpl struct {
	nextHandler pluginAPI.Handler
}
func NewPlugin(handler pluginAPI.Handler) pluginAPI.PluginAPI {
	flavorsPlugin := &DefaultFlavorsImpl{
		nextHandler: handler,
	}
	return flavorsPlugin
}
const MEGABYTE = 1048576  
type Flavor struct {
	Memory            int64
}
var flavors map[string]Flavor 
var flavorsEnforced = os.Getenv("SWARM_FLAVORS_ENFORCED")

func init() {
	log.Info("flavors.init()")
	readFlavorFile()

} 
func readFlavorFile() {
	log.Info("Flavors.ReadFlavorFile() ..........")
	if (flavorsEnforced != "true") {
		log.Info("Flavors not enforced")
		return
	}
	var flavorsFile = os.Getenv("SWARM_FLAVORS_FILE")
	if flavorsFile == "" {
		log.Warn("Missing SWARM_FLAVORS_FILE environment variable, using locate default ./flavors.json")
		flavorsFile = "flavors.json"
	}

	file, err := os.Open(flavorsFile)
	if err != nil {
		log.Fatal(err)
		panic("Error: could not open flavorsFile ")
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&flavors)
	if err != nil {
		log.Fatal("Error in flavors file decode:", err)
		panic("Error: could not decode flavors file ")
	}
	if _, ok := flavors["default"]; !ok {
        log.Fatal("Error flavors file does not contain default flavor")
		panic("Error: flavors file does not contain default flavor")		
	}
	// convert memory to megabytes
	for key, value := range flavors {
		flavors[key] = Flavor{value.Memory*MEGABYTE}
	}
	log.Infof("Flavors %+v",flavors)
}
func (flavorsImpl *DefaultFlavorsImpl) Handle(command utils.CommandEnum, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin flavors Got command: " + command)
	log.Debug("Flavors enforced: ",flavorsEnforced)
	if(flavorsEnforced != "true") {
		log.Debug("Flavors not enforced")
		return flavorsImpl.nextHandler(command, cluster, w, r, swarmHandler)
	}
	if(command != "containercreate") {
		log.Debug("Flavors not containercreate")
		return flavorsImpl.nextHandler(command, cluster, w, r, swarmHandler)
	}
	defer r.Body.Close()
	if reqBody, _ := ioutil.ReadAll(r.Body); len(reqBody) > 0 {
		var flavorIn Flavor
		var buf bytes.Buffer
		var containerConfig dockerclient.ContainerConfig
		if err := json.NewDecoder(bytes.NewReader(reqBody)).Decode(&containerConfig); err != nil {
			return err
		}
	   	flavorIn.Memory = containerConfig.HostConfig.Memory
		_key := "default"
	    for key, value := range flavors {
	      if(value == flavorIn) {
			_key = key
			break 			
	      }
	    }
        log.Debug("apply flavor: ",_key)
        containerConfig.HostConfig.Memory = flavors[_key].Memory
		if err := json.NewEncoder(&buf).Encode(containerConfig); err != nil {
			return err
		}
		r, _ = utils.ModifyRequest(r, bytes.NewReader(buf.Bytes()), "", "")
        return flavorsImpl.nextHandler(command, cluster, w, r, swarmHandler)
	}
	log.Debug("Flavors returning nil in create")
	return nil
}

