package keystone

import (
	"encoding/json"

	"os"

	log "github.com/Sirupsen/logrus"
)

type Configs struct {
	AuthTokenHeader    string
	TenancyLabel       string
	KeystoneUrl        string
	KeyStoneXAuthToken string
}

const defaultConfigurationFileCreationPath = "/tmp/authHookConf.json"

var Configuration *Configs
var swarmConfig = os.Getenv("SWARM_CONFIG")

func (*Configs) ReadConfigurationFormfile() {
	if swarmConfig == "" {
		log.Warn("Missing SWARM_CONFIG environment variable, trying to locate deafult authHookConf.json")
		swarmConfig = "authHookConf.json"
	}

	file, err := os.Open(swarmConfig)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	decoder := json.NewDecoder(file)
	Configuration = new(Configs)
	err = decoder.Decode(&Configuration)
	if err != nil {
		log.Println("error:", err)
	}
	log.Debug("*************************")
	log.Debug(Configuration)
	log.Debug("*************************")
}

func (*Configs) GetConf() *Configs {
	return Configuration
}

func (*Configs) CreateDefaultsConfigurationfile() {
	confs := Configs{
		AuthTokenHeader:    "X-Auth-Token",
		TenancyLabel:       "com.ibm.tenant.0",
		KeystoneUrl:        "http://127.0.0.1:5000/v2.0/",
		KeyStoneXAuthToken: "ADMIN",
	}

	bytesConfigurationData, e0 := json.Marshal(&confs)

	if e0 != nil {
		log.Fatal(e0)
	}
	f, e1 := os.Create(defaultConfigurationFileCreationPath)
	if e1 != nil {
		log.Fatal(e1)
	}

	defer f.Close()
	n2, e2 := f.Write(bytesConfigurationData)
	if e2 != nil {
		log.Fatal(e2)
	}
	log.Println(n2)
	f.Sync()

}
