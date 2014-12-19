package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

type ApiScheduler struct {
	cluster *cluster.Cluster
	url     string
}

func (s *ApiScheduler) Initialize(cluster *cluster.Cluster, opts map[string]string) error {
	url, exists := opts["url"]
	if !exists {
		log.Fatal("You have to provide --scheduler-option=url:<scheduler_api_host>")
	}
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	s.url = url
	s.cluster = cluster
	return nil
}

func (s *ApiScheduler) CreateContainer(config *dockerclient.ContainerConfig, name string) (*cluster.Container, error) {

	url := s.url + "/containers/create"

	// Request
	reqData := map[string]interface{}{
		"Name":      name,
		"Container": config,
		"Nodes":     s.cluster.Nodes(),
	}
	reqBytes, err := json.Marshal(reqData)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp.StatusCode != 200 {
		log.Errorf("Scheduler API error: %d - %q", resp.StatusCode, err)
		return nil, fmt.Errorf("%q", err)
	}
	defer resp.Body.Close()

	// Response
	body, _ := ioutil.ReadAll(resp.Body)
	// Format: `{"Node": "http://<node_addr>", "Id": "<container_id>"}`
	var respData map[string]string
	_ = json.Unmarshal(body, &respData)

	// Add container to cluster
	node := s.cluster.Node(respData["Node"])
	container := node.Container(respData["Id"])
	return container, nil
}

func (s *ApiScheduler) RemoveContainer(container *cluster.Container, force bool) error {

	node := container.Node()
	url := s.url + "/containers/" + container.Id

	// Request
	reqData := map[string]interface{}{
		"Node":  node,
		"Force": force,
	}
	reqBytes, _ := json.Marshal(reqData)
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	// Response
	client := &http.Client{}
	resp, err := client.Do(req)
	if resp.StatusCode != 204 {
		log.Errorf("Scheduler API error: %d - %q", resp.StatusCode, err)
		return fmt.Errorf("%q", err)
	}
	defer resp.Body.Close()

	// Remove container from cluster
	err = node.RemoveContainer(container)
	return err
}
