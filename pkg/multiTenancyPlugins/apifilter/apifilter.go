package apifilter

import (
    log "github.com/Sirupsen/logrus"
	"errors"
	"encoding/json"
	"net/http"
	"io/ioutil"
	"os"
	"gopkg.in/yaml.v2"
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


type apiDisabledType struct {
	Attach bool
	Build bool
	Commit bool
	Cp bool
	Create bool
	Diff bool
	Events bool
	Exec bool
	Export bool
	History bool
	Import bool
	Info bool
	Inspect bool
	Kill bool
	Load bool
	Login bool
	Logout bool
	Logs bool
	Network_connect bool
	Network_create bool
	Network_disconnect bool
	Network_ls bool
	Network_rm bool
	Pause bool
	Port bool
	Ps bool
	Pull bool
	Push bool
	Rename bool
	Rmi bool
	Rm bool
	Run bool
	Save bool
	Search bool
	Start bool
	Stat bool
	Stop bool
	Tag bool
	Top bool
	Unpause bool
	Update bool
	Version bool
	Volume_create bool
	Volume_inspect bool
	Volume_ls bool
	Volume_rm bool
	Wait bool
}

var apiDisabled apiDisabledType

var apiDisabledMap map[string]bool





func init() {
	log.Info("apifliter.init()")
	//readApiSupportedFile()
	readApiFilter()
}

func readApiFilter() {
	log.Info("apifilter.readApiFilterFile() ..........")
	type Apifilteryaml struct {
		DisableAPI []string	
    }
	var config Apifilteryaml
	var f = os.Getenv("SWARM_API_FILTER_FILE")
	if f == "" {
		log.Warn("Missing SWARM_API_FILTER_FILE environment variable, using locate default ./apifiler.json")
		f = "apifilter.yaml"
	}
	log.Info("SWARM_API_FILTER_FILE: ",f)

	//file, err := os.Open(f)
	source, err := ioutil.ReadFile(f)
	if err != nil {
		log.Info("NO SWARM_API_FILTER_FILE")
		return
	}
	err = yaml.Unmarshal(source, &config)
    if err != nil {
        panic(err)
    }
	//log.Infof("DisableAPI %+v",config.DisableAPI)
	apiDisabledMap = make(map[string]bool)
}

func readApiSupportedFile() {
	log.Info("apifilter.readApiSupportedFile() ..........")
	var f = os.Getenv("SWARM_API_SUPPORTED_FILE")
	if f == "" {
		log.Warn("Missing SWARM_API_SUPPORTED_FILE environment variable, using locate default ./apisupported.json")
		f = "apisupported.json"
	}
	log.Info("SWARM_API_SUPPORTED_FILE: ",f)

	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
		panic("Error: could not open api supported file ")
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&apiDisabled)
	if err != nil {
		log.Fatal("Error in apiSupported decode:", err)
		panic("Error: could not decode apiSupported ")
	}
	log.Infof("apiDisabled %+v",apiDisabled)
	apiDisabledMap = make(map[string]bool)
	/*
	apiDisabledMap["containerattach"] = apiDisabled.Attach
	apiDisabledMap["containerbuild"] = apiDisabled.Build
	apiDisabledMap["imagecommit"] = apiDisabled.Commit
	apiDisabledMap["containercreate"] = apiDisabled.Create
	apiDisabledMap["containercopy"] = apiDisabled.Cp
	apiDisabledMap["containerdiff"] = apiDisabled.Diff
	apiDisabledMap["containerevents"] = apiDisabled.Events
	apiDisabledMap["containerexec"] = apiDisabled.Exec
	apiDisabledMap["containerexport"] = apiDisabled.Export
	apiDisabledMap["imagehistory"] = apiDisabled.History
	apiDisabledMap["imageimport"] = apiDisabled.Import
	apiDisabledMap["clusterInfo"] = apiDisabled.Info
	apiDisabledMap["containerjson"] = apiDisabled.Inspect
	apiDisabledMap["containerkill"] = apiDisabled.Kill
	apiDisabledMap["imageload"] = apiDisabled.Load	
	apiDisabledMap["serverlogin"] = apiDisabled.Login
	apiDisabledMap["serverlogout"] = apiDisabled.Logout
	apiDisabledMap["containerlogs"] = apiDisabled.Logs
	apiDisabledMap["networkconnect"] = apiDisabled.Network_connect
	apiDisabledMap["networkcreate"] = apiDisabled.Network_create
	apiDisabledMap["networkdisconnect"] = apiDisabled.Network_disconnect
	apiDisabledMap["listNetworks"] = apiDisabled.Network_ls
	apiDisabledMap["networkremove"] = apiDisabled.Network_rm
	apiDisabledMap["containerpause"] = apiDisabled.Pause	
	apiDisabledMap["containertport"] = apiDisabled.Port
	apiDisabledMap["listContainers"] = apiDisabled.Ps
	apiDisabledMap["imagepull"] = apiDisabled.Pull
	apiDisabledMap["imagepush"] = apiDisabled.Push
	apiDisabledMap["containerrename"] = apiDisabled.Rename
	apiDisabledMap["imageremove"] = apiDisabled.Rmi
	apiDisabledMap["containerdelete"] = apiDisabled.Rm
	apiDisabledMap["containerrun"] = apiDisabled.Run
	apiDisabledMap["imagesave"] = apiDisabled.Save
	apiDisabledMap["imagesearch"] = apiDisabled.Search
	apiDisabledMap["containerstart"] = apiDisabled.Start
	apiDisabledMap["containerstop"] = apiDisabled.Stop
	apiDisabledMap["imagetag"] = apiDisabled.Tag
	apiDisabledMap["containertop"] = apiDisabled.Top
	apiDisabledMap["containerunpause"] = apiDisabled.Unpause
	apiDisabledMap["containerupdate"] = apiDisabled.Update
	apiDisabledMap["version"] = apiDisabled.Version	
	apiDisabledMap["volumecreate"] = apiDisabled.Volume_create
	apiDisabledMap["volumeinspect"] = apiDisabled.Volume_inspect
	apiDisabledMap["volumelist"] = apiDisabled.Volume_ls
	apiDisabledMap["volumeremove"] = apiDisabled.Volume_rm
	apiDisabledMap["containerwait"] = apiDisabled.Wait
	*/
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

 