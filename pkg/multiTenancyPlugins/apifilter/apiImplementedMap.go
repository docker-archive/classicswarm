package apifilter

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	c "github.com/docker/swarm/pkg/multiTenancyPlugins/utils"
	"os"
)

var supportedAPIsMap map[c.CommandEnum]bool

func initSupportedAPIsMap() {
	supportedAPIsMap = make(map[c.CommandEnum]bool)
	//containers
	supportedAPIsMap[c.CONTAINER_CREATE] = true
	supportedAPIsMap[c.CONTAINER_JSON] = true
	supportedAPIsMap[c.PS] = true
	//container
	supportedAPIsMap[c.CONTAINER_START] = true
	supportedAPIsMap[c.CONTAINER_ARCHIVE] = true
	supportedAPIsMap[c.CONTAINER_ATTACH] = true
	supportedAPIsMap["containerbuild"] = true
	supportedAPIsMap[c.CONTAINER_COPY] = true
	supportedAPIsMap[c.CONTAINER_CHANGES] = true
	supportedAPIsMap[c.EVENTS] = true
	supportedAPIsMap[c.CONTAINER_EXEC] = false
	supportedAPIsMap[c.CONTAINER_EXPORT] = false
	supportedAPIsMap[c.CONTAINER_JSON] = true
	supportedAPIsMap[c.CONTAINER_RESTART] = true
	supportedAPIsMap[c.CONTAINER_KILL] = true
	supportedAPIsMap[c.CONTAINER_LOGS] = true
	supportedAPIsMap[c.CONTAINER_PAUSE] = true
	supportedAPIsMap["containertport"] = true
	supportedAPIsMap[c.CONTAINER_RENAME] = false
	supportedAPIsMap[c.CONTAINER_DELETE] = true
	supportedAPIsMap[c.CONTAINER_STOP] = true
	supportedAPIsMap[c.CONTAINER_TOP] = true
	supportedAPIsMap[c.CONTAINER_UNPAUSE] = true
	supportedAPIsMap[c.CONTAINER_UPDATE] = true
	supportedAPIsMap[c.CONTAINER_WAIT] = true
	supportedAPIsMap[c.JSON] = true
	supportedAPIsMap[c.CONTAINER_STATS] = true
	supportedAPIsMap[c.CONTAINER_RESIZE] = false
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
	supportedAPIsMap[c.IMAGES_JSON] = true //listImages
	//server
	supportedAPIsMap["serverlogin"] = false
	supportedAPIsMap["serverlogout"] = false
	//Network
	supportedAPIsMap["connectNetwork"] = false
	supportedAPIsMap[c.NETWORK_CREATE] = true
	supportedAPIsMap["disconnectNetwork"] = false
	supportedAPIsMap[c.NETWORKS_LIST] = true
	supportedAPIsMap["networkremove"] = false
	//Volume
	supportedAPIsMap["createVolume"] = false
	supportedAPIsMap["inspectVolume"] = false
	supportedAPIsMap["listVolume"] = false
	supportedAPIsMap["removeVolume"] = false

	//general
	supportedAPIsMap[c.INFO] = true
	supportedAPIsMap[c.VERSION] = false

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
	supportedAPIsMap[c.CONTAINER_RESIZE] = false      //containers/{name:.*}/resize
	supportedAPIsMap["execstart"] = false             //exec/{execid:.*}/start
	supportedAPIsMap["execresize"] = false            //exec/{execid:.*}/resize
	//images/create:                    (Create an image) is it equal to imagepull??
}

func modifySupportedWithDisabledApi() {
	type Filter struct {
		Disableapi []c.CommandEnum
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
