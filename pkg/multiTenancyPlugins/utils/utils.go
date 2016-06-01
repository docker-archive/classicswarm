package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/headers"
	"github.com/gorilla/mux"
)

type ValidationOutPutDTO struct {
	ContainerID  string
	Links        []string
	VolumesFrom  []string
	Binds        []string
	Env          []string
	ErrorMessage string
	//Quota can live here too? Currently quota needs only raise error
	//What else
}

//UTILS

//Use something else...
func ParseCommand(r *http.Request) string {
	return commandParser(r)
}

func ModifyRequest(r *http.Request, body io.Reader, urlStr string, containerID string) (*http.Request, error) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
		r.Body = rc
	}
	if urlStr != "" {
		u, err := url.Parse(urlStr)

		if err != nil {
			return nil, err
		}
		r.URL = u
		mux.Vars(r)["name"] = containerID
	}
	return r, nil
}

func getResourceId(r *http.Request) string {
	return mux.Vars(r)["name"]
}

//Assumes ful ID was injected
func IsOwner(cluster cluster.Cluster, tenantId string, r *http.Request) bool {
	for _, container := range cluster.Containers() {
		if container.Info.ID == getResourceId(r) {
			return container.Labels[headers.TenancyLabel] == tenantId
		}
	}
	return false
}

//Expand / Refactor
func CleanUpLabeling(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	newBody := bytes.Replace(rec.Body.Bytes(), []byte(headers.TenancyLabel), []byte(" "), -1)
	//TODO - Here we just use the token for the tenant name for now so we remove it from the data before returning to user.
	newBody = bytes.Replace(newBody, []byte(r.Header.Get(headers.AuthZTenantIdHeaderName)), []byte(""), -1)
	newBody = bytes.Replace(newBody, []byte(",\" \":\" \""), []byte(""), -1)
	log.Debugf("Clean up labeling done.")
	//	log.Debug("Got this new body...", string(newBody))
	return newBody
}

// RandStringBytesRmndr used to generate a name for docker volume create when no name is supplied
// The tenant id is then appended to the name by the caller
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

//Maybe merge to one regExp
var containers = regexp.MustCompile(`/containers/(.*)`)
var containersWithIdentifier = regexp.MustCompile(`/containers/(.*)/(.*)`)

//TODO - Do the same for networks, images, and so on. What is not supported will fail because of the generic message will go to default

//TODO - Handle delete better
func commandParser(r *http.Request) string {

	paramsArr1 := containers.FindStringSubmatch(r.URL.Path)
	paramsArr2 := containersWithIdentifier.FindStringSubmatch(r.URL.Path)
	//assert the it is not possible for two of them to co-Exist

	log.Debug(paramsArr1)
	log.Debug(paramsArr2)

	switch r.Method {
	case "DELETE":
		if len(paramsArr1) > 0 && strings.HasPrefix(paramsArr1[0], "/containers") {
			return "containerdelete"
		}
	}
	//Order IS important
	if len(paramsArr2) == 3 {

		return "container" + paramsArr2[2]
	}
	if len(paramsArr1) == 2 {

		if paramsArr1[1] == "json" {
			return "listContainers"
		}
		return "container" + paramsArr1[1]
	}

	if strings.HasSuffix(r.URL.Path, "/networks") {
		return "listNetworks"
	}
	
	if strings.HasSuffix(r.URL.Path, "/info") {
		return "clusterInfo"
	}

	return "This is not supported yet and will end up in the default of the Switch"
}

//FilterNetworks - filter out all networks not created by tenant.
func FilterNetworks(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	var networks cluster.Networks
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&networks); err != nil {
		log.Error(err)
		return nil
	}
	var candidates cluster.Networks
	tenantName := r.Header.Get(headers.AuthZTenantIdHeaderName)
	for _, network := range networks {
		fullName := strings.SplitN(network.Name, "/", 2)
		name := fullName[len(fullName)-1]
		if strings.HasPrefix(name, tenantName) {
			network.Name = strings.TrimLeft(name, tenantName)
			candidates = append(candidates, network)
		}
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(candidates); err != nil {
		log.Error(err)
		return nil
	}
	return buf.Bytes()
}
