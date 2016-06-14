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

type commandEnum string

const (
	//For reference look at primary.go
	PING    commandEnum = "ping"
	EVENTS  commandEnum = "events"
	INFO    commandEnum = "info"
	VERSION commandEnum = "version"
	//SKIP ...
	CONTAINERS_PS     commandEnum = "ps"
	CONTAINERS_JSON   commandEnum = "json"
	CONTAINER_ARCHIVE commandEnum = "containerArchive"
	CONTAINER_EXPORT  commandEnum = "containerExport"
	CONTAINER_CHANGES commandEnum = "containerChanges"
	CONTAINER_JSON    commandEnum = "containerJson"
	CONTAINER_TOP     commandEnum = "containerTop"
	CONTAINER_LOGS    commandEnum = "containerLogs"
	CONTAINER_STATS   commandEnum = "containerStats"
	//SKIP ...
	NETWORKS_LIST   commandEnum = "NetworksList"
	NETWORK_INSPECT commandEnum = "NetworkInspect"
	//SKIP ...
	//POST
	CONTAINER_CREATE  commandEnum = "containerCreate"
	CONTAINER_KILL    commandEnum = "containerKill"
	CONTAINER_PAUSE   commandEnum = "containerPause"
	CONTAINER_UNPAUSE commandEnum = "containerUnpause"
	CONTAINER_RENAME  commandEnum = "containerRename"
	CONTAINER_RESTART commandEnum = "containerRestart"
	CONTAINER_START   commandEnum = "containerStart"
	CONTAINER_STOP    commandEnum = "containerStop"
	CONTAINER_UPDATE  commandEnum = "containerUpdate"
	CONTAINER_WAIT    commandEnum = "containerWait"
	CONTAINER_RESIZE  commandEnum = "containerResize"
	CONTAINER_ATTACH  commandEnum = "containerAttach"
	CONTAINER_COPY    commandEnum = "containerCopy"
	CONTAINER_EXEC    commandEnum = "containerExec"
	//SKIP ...

	CONTAINER_DELETE commandEnum = "containerDelete"
)

var containersRegexp = regexp.MustCompile("/containers/(.*)/(.*)|/containers/(\\w+)")
var networksRegexp = regexp.MustCompile("/networks/(.*)/(.*)|/networks/(\\w+)")
var clusterRegExp = regexp.MustCompile("/(.*)/(.*)")

func commandParser(r *http.Request) string {
	containersParams := containersRegexp.FindStringSubmatch(r.URL.Path)
	networksParams := networksRegexp.FindStringSubmatch(r.URL.Path)
	clusterParams := clusterRegExp.FindStringSubmatch(r.URL.Path)

	log.Debug(containersParams)
	log.Debug(networksParams)
	log.Debug(clusterParams)

	switch r.Method {
	case "DELETE":
		if len(containersParams) > 0 {
			return "containerdelete"
		}
		if len(networksParams) > 0 {
			return "networkdelete"
		}

	case "GET", "POST":
		if len(containersParams) == 4 && containersParams[2] != "" {
			return "container" + containersParams[2]
		} else if len(containersParams) == 4 && containersParams[3] != "" {
			return "containers" + containersParams[3] //S
		}
		if len(clusterParams) == 3 {
			return clusterParams[2]
		}
		if len(networksParams) == 4 && networksParams[3] != "" {
			return "networkInspect"
		} else if len(networksParams) == 4 && networksParams[1] == "" && networksParams[2] == "" && networksParams[3] == "" {
			return "networksList" //S
		}
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
			network.Name = strings.TrimPrefix(name, tenantName)
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
