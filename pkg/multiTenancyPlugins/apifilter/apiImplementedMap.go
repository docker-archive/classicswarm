package apifilter

import "github.com/docker/swarm/pkg/multiTenancyPlugins/utils"

var supportedAPIsMap map[utils.CommandEnum]bool

func init() {
	supportedAPIsMap = make(map[utils.CommandEnum]bool)
	supportedAPIsMap["containerscreate"] = true
	supportedAPIsMap["containerstart"] = true
	supportedAPIsMap["containerattach"] = true
	supportedAPIsMap["containerbuild"] = true
	supportedAPIsMap["imagecommit"] = false
	supportedAPIsMap["containercopy"] = true
	supportedAPIsMap["containerchanges"] = true
	supportedAPIsMap["containerevents"] = false
	supportedAPIsMap["containerexec"] = false
	supportedAPIsMap["containerexport"] = false
	supportedAPIsMap["imagehistory"] = false
	supportedAPIsMap["imageimport"] = false
	supportedAPIsMap["info"] = true
	supportedAPIsMap["containerjson"] = true
	supportedAPIsMap["containerrestart"] = true
	supportedAPIsMap["containerkill"] = true
	supportedAPIsMap["imageload"] = false
	supportedAPIsMap["serverlogin"] = false
	supportedAPIsMap["serverlogout"] = false
	supportedAPIsMap["containerlogs"] = true
	supportedAPIsMap["connectNetwork"] = false
	supportedAPIsMap["createNetwork"] = true
	supportedAPIsMap["disconnectNetwork"] = false
	supportedAPIsMap["listNetworks"] = true
	supportedAPIsMap["networkremove"] = false
	supportedAPIsMap["containerpause"] = true
	supportedAPIsMap["containertport"] = true
	supportedAPIsMap["listContainers"] = true
	supportedAPIsMap["imagepull"] = false
	supportedAPIsMap["imagepush"] = false
	supportedAPIsMap["containerrename"] = false
	supportedAPIsMap["imageremove"] = false
	supportedAPIsMap["containerdelete"] = true
	supportedAPIsMap["imagesave"] = false
	supportedAPIsMap["imagesearch"] = false
	supportedAPIsMap["containerstart"] = true
	supportedAPIsMap["containerstop"] = true
	supportedAPIsMap["imagetag"] = false
	supportedAPIsMap["containertop"] = true
	supportedAPIsMap["containerunpause"] = true
	supportedAPIsMap["containerupdate"] = true
	supportedAPIsMap["version"] = false
	supportedAPIsMap["createVolume"] = false
	supportedAPIsMap["inspectVolume"] = false
	supportedAPIsMap["listVolume"] = false
	supportedAPIsMap["removeVolume"] = false
	supportedAPIsMap["containerwait"] = true
	supportedAPIsMap["containersjson"] = true
	supportedAPIsMap["containersps"] = true

}
