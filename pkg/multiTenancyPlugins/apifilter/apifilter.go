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


type apiSupportedType struct {
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

var apiSupported apiSupportedType

var apiSupportedMap map[string]bool



func init() {
	log.Info("apifliter.init()")
	readApiSupportedFile()
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
	err = decoder.Decode(&apiSupported)
	if err != nil {
		log.Fatal("Error in apiSupported decode:", err)
		panic("Error: could not decode apiSupported ")
	}
	log.Infof("apiSupported %+v",apiSupported)
	apiSupportedMap = make(map[string]bool)
	apiSupportedMap["containerattach"] = apiSupported.Attach
	apiSupportedMap["containerbuild"] = apiSupported.Build
	apiSupportedMap["imagecommit"] = apiSupported.Commit
	apiSupportedMap["containercreate"] = apiSupported.Create
	apiSupportedMap["containercopy"] = apiSupported.Cp
	apiSupportedMap["containerdiff"] = apiSupported.Diff
	apiSupportedMap["containerevents"] = apiSupported.Events
	apiSupportedMap["containerexec"] = apiSupported.Exec
	apiSupportedMap["containerexport"] = apiSupported.Export
	apiSupportedMap["imagehistory"] = apiSupported.History
	apiSupportedMap["imageimport"] = apiSupported.Import
	apiSupportedMap["clusterInfo"] = apiSupported.Info
	apiSupportedMap["containerjson"] = apiSupported.Inspect
	apiSupportedMap["containerkill"] = apiSupported.Kill
	apiSupportedMap["imageload"] = apiSupported.Load	
	apiSupportedMap["serverlogin"] = apiSupported.Login
	apiSupportedMap["serverlogout"] = apiSupported.Logout
	apiSupportedMap["containerlogs"] = apiSupported.Logs
	apiSupportedMap["networkconnect"] = apiSupported.Network_connect
	apiSupportedMap["networkcreate"] = apiSupported.Network_create
	apiSupportedMap["networkdisconnect"] = apiSupported.Network_disconnect
	apiSupportedMap["listNetworks"] = apiSupported.Network_ls
	apiSupportedMap["networkremove"] = apiSupported.Network_rm
	apiSupportedMap["containerpause"] = apiSupported.Pause	
	apiSupportedMap["containertport"] = apiSupported.Port
	apiSupportedMap["listContainers"] = apiSupported.Ps
	apiSupportedMap["imagepull"] = apiSupported.Pull
	apiSupportedMap["imagepush"] = apiSupported.Push
	apiSupportedMap["containerrename"] = apiSupported.Rename
	apiSupportedMap["imageremove"] = apiSupported.Rmi
	apiSupportedMap["containerdelete"] = apiSupported.Rm
	apiSupportedMap["containerrun"] = apiSupported.Run
	apiSupportedMap["imagesave"] = apiSupported.Save
	apiSupportedMap["imagesearch"] = apiSupported.Search
	apiSupportedMap["containerstart"] = apiSupported.Start
	apiSupportedMap["containerstop"] = apiSupported.Stop
	apiSupportedMap["imagetag"] = apiSupported.Tag
	apiSupportedMap["containertop"] = apiSupported.Top
	apiSupportedMap["containerunpause"] = apiSupported.Unpause
	apiSupportedMap["containerupdate"] = apiSupported.Update
	apiSupportedMap["version"] = apiSupported.Version	
	apiSupportedMap["volumecreate"] = apiSupported.Volume_create
	apiSupportedMap["volumeinspect"] = apiSupported.Volume_inspect
	apiSupportedMap["volumelist"] = apiSupported.Volume_ls
	apiSupportedMap["volumeremove"] = apiSupported.Volume_rm
	apiSupportedMap["containerwait"] = apiSupported.Wait
	log.Infof("apiSupportedMap %+v",apiSupportedMap)
}

func (apiFilterImpl *DefaultApiFilterImpl) Handle(command string, cluster cluster.Cluster, w http.ResponseWriter, r *http.Request, swarmHandler http.Handler) error {
	log.Debug("Plugin apiFilter Got command: " + command)
	if apiSupportedMap[command] {
		return apiFilterImpl.nextHandler(command, cluster, w, r, swarmHandler)		
	} else {
		return errors.New("Command Not Supported!")		
	}
	
}

 