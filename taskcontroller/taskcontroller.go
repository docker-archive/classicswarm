package main 

import (
	"encoding/json"
	"flag"
	"github.com/docker/swarmd/controller"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var config = flag.String("config", "", "Path to the config file.")
var syncPeriod = flag.Duration("period", time.Second * 30, "Period between container synchronization")
var endpoint = flag.String("endpoint", "tcp://localhost:4243", "Endpoint for the Docker API")

func main() {
	data, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Printf("Failed to load file: %s (%#v)", *config, err)
		os.Exit(1)
	}
	var containers []controller.Container
	err = json.Unmarshal(data, &containers)
	if err != nil {
		log.Printf("Failed to load file: %s (%#v)", *config, err)
		os.Exit(1)
	}
	dockerClient, err := docker.NewClient(*endpoint)
	if err != nil {
		log.Panicf("Couldn't connnect to docker.")
	}
	taskController := controller.MakeTaskController(dockerClient, containers)
	for {
		taskController.SyncContainers()
		time.Sleep(*syncPeriod)
	}
}

