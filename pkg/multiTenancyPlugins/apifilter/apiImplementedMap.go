package apifilter

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
	"os"
)

var supportedAPIsMap map[utils.CommandEnum]bool

func initSupportedAPIsMap() {
	supportedAPIsMap = make(map[utils.CommandEnum]bool)
	//containers
	supportedAPIsMap["containerscreate"] = true
	supportedAPIsMap["containersjson"] = true
	supportedAPIsMap["containersps"] = true
	//container
	supportedAPIsMap["containerstart"] = true
	supportedAPIsMap["containerarchive"] = true
	supportedAPIsMap["containerattach"] = true
	supportedAPIsMap["containerbuild"] = true
	supportedAPIsMap["containercopy"] = true
	supportedAPIsMap["containerchanges"] = true
	supportedAPIsMap["events"] = true
	supportedAPIsMap["containerexec"] = false
	supportedAPIsMap["containerexport"] = false
	supportedAPIsMap["containerjson"] = true
	supportedAPIsMap["containerrestart"] = true
	supportedAPIsMap["containerkill"] = true
	supportedAPIsMap["containerlogs"] = true
	supportedAPIsMap["containerpause"] = true
	supportedAPIsMap["containertport"] = true
	supportedAPIsMap["containerrename"] = false
	supportedAPIsMap["containerdelete"] = true
	supportedAPIsMap["containerstop"] = true
	supportedAPIsMap["containertop"] = true
	supportedAPIsMap["containerunpause"] = true
	supportedAPIsMap["containerupdate"] = true
	supportedAPIsMap["containerwait"] = true
	supportedAPIsMap["listContainers"] = true
	supportedAPIsMap["containerstats"] = true
	//image
	supportedAPIsMap["imagecommit"] = false
	supportedAPIsMap["imagehistory"] = false
	supportedAPIsMap["imageimport"] = false
	supportedAPIsMap["imageload"] = false
	supportedAPIsMap["imagepull"] = false
	supportedAPIsMap["imagepush"] = false
	supportedAPIsMap["imageremove"] = false
	supportedAPIsMap["imagesave"] = false
	supportedAPIsMap["imagesearch"] = false
	supportedAPIsMap["imagetag"] = false
	supportedAPIsMap["imagesjson"] = true //listImages
	//server
	supportedAPIsMap["serverlogin"] = false
	supportedAPIsMap["serverlogout"] = false
	//Network
	supportedAPIsMap["connectNetwork"] = false
	supportedAPIsMap["createNetwork"] = true
	supportedAPIsMap["disconnectNetwork"] = false
	supportedAPIsMap["networkslist"] = true
	supportedAPIsMap["networkremove"] = false
	//Volume
	supportedAPIsMap["createVolume"] = false
	supportedAPIsMap["inspectVolume"] = false
	supportedAPIsMap["listVolume"] = false
	supportedAPIsMap["removeVolume"] = false

	//general
	supportedAPIsMap["info"] = true
	supportedAPIsMap["version"] = false

	//new
	supportedAPIsMap["ping"] = false                  //_ping
	supportedAPIsMap["imagesviz"] = false             //notImplementedHandler
	supportedAPIsMap["getRepositoriesImages"] = false //images/get	(Get a tarball containing all images)
	supportedAPIsMap["getRepositoryImages"] = false   //images/{name:.*}/get	(Get a tarball containing all images in a repository)
	supportedAPIsMap["inspectImage"] = false          //images/{name:.*}/json
	supportedAPIsMap["execjson"] = false              //exec/{execid:.*}/json
	supportedAPIsMap["networkinspect"] = false        //networks/{networkid:.*}"
	supportedAPIsMap["auth"] = false                  //auth
	supportedAPIsMap["commit"] = false                //commit
	supportedAPIsMap["build"] = false                 //build
	supportedAPIsMap["containerresize"] = false       //containers/{name:.*}/resize
	supportedAPIsMap["execstart"] = false             //exec/{execid:.*}/start
	supportedAPIsMap["execresize"] = false            //exec/{execid:.*}/resize
	//images/create:                     (Create an image) is it equal to imagepull??
}

func modifySupportedWithDisabledApi() {
	type Filter struct {
		Disableapi []utils.CommandEnum
	}
	var filter Filter
	var f = os.Getenv("SWARM_APIFILTER_FILE")
	if f == "" {
		f = "apifilter.json"
	}

	file, err := os.Open(f)
	if err != nil {
		log.Info("no API FILTER file")
		return
	}

	log.Info("SWARM_APIFILTER_FILE: ", f)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&filter)
	if err != nil {
		log.Fatal("Error in apifilter decode:", err)
		panic("Error: could not decode apifilter.json")
	}
	log.Infof("filter %+v", filter)
	for _, e := range filter.Disableapi {
		if supportedAPIsMap[e] {
			log.Infof("disable %+v", e)
			supportedAPIsMap[e] = false
		}

	}
}
