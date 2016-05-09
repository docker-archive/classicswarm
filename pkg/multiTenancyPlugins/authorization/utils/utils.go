package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"strings"

	"net/http/httptest"
	"math/rand"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"

	"strconv"

	"github.com/docker/swarm/pkg/multiTenancyPlugins/authorization/headers"
	"github.com/gorilla/mux"
	"github.com/jeffail/gabs"
//	"encoding/json"
	"github.com/samalba/dockerclient"
)

type ValidationOutPutDTO struct {
	ContainerID string
	Links       []string
	VolumesFrom []string
	Binds       []string
	Env         []string
	ErrorMessage string
	//Quota can live here too? Currently quota needs only raise error
	//What else
}

//UTILS

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
/*
func CheckLinksOwnerShip(cluster cluster.Cluster, tenantName string, r *http.Request, reqBody []byte, containerConfig dockerclient.ContainerConfig) (bool, *ValidationOutPutDTO) {
log.Debugf("CheckLinksOwnerShip for tenant %s\n",tenantName)
	jsonParsed, _ := gabs.ParseJSON(reqBody)

	//TODO - Consider refactor all to use json parse and not regexp and maybe save memory on de duplication
	log.Debug("Checking links...")
	children, _ := jsonParsed.Path("HostConfig.Links").Children()
	containers := cluster.Containers()
	linkSet := make(map[string]string)
	var c int
	var l int
	log.Debug("**************************************************")
	for _, child := range children {
		log.Debug("_________________")
		c++

		pair := child.Data().(string)
		linkPair := strings.Split(pair, ":")
		link := strings.TrimSpace(linkPair[0])
		log.Debug("Pair:" + pair)
		log.Debug(linkPair)
		log.Debug("Link:" + link)
		for _, container := range containers {
			log.Debug("containerName: " + container.Info.Name)
			log.Debug("Comparing with: " + "/"+link+tenantName)
			log.Debug("or Comparing with: " + "/"+link)
			log.Debug("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
			if "/"+link+tenantName == container.Info.Name || "/"+link == container.Info.Name {
				log.Debug("XXXXXXXXXXXXXXLINK FOUNDXXXXXXXXXXXXXXXXXXXXXXXX")
				linkSet[container.Info.ID] = link
				l++
			}
		}
	}
	log.Debug("**************************************************")
	if l != c {
		//TODO - Change to pointer and return nil
		return false, &ValidationOutPutDTO{ContainerID: "", Links: linkSet}
	}
	v := ValidationOutPutDTO{ContainerID: "", Links: linkSet}
	return true, &v

}
*/
func CheckVolumeBinds(tenantName string, containerConfig dockerclient.ContainerConfig) ([]string,error) {
	log.Debug("CheckVolumeBinds")
	binds := containerConfig.HostConfig.Binds
	for i,b := range containerConfig.HostConfig.Binds {
		if index := strings.Index(b,":"); index > -1 {
		  v := b[0:index]
		  log.Debug("v: ",v) 		
		  if strings.Contains(v,"/") {
			return nil,errors.New("Mount to host file system is prohibited!")
		  }
		  binds[i] = strings.Replace(binds[i],v,v + tenantName,1)		  
		}
	}
	return binds,nil
}

//Check that tenant references only containers that it owns and modify reference name to use tenantName suffix or the container id
func CheckContainerReferences(cluster cluster.Cluster, tenantName string, containerConfig dockerclient.ContainerConfig) (bool, *ValidationOutPutDTO) {
	log.Debug("in CheckContainerReferences")
	log.Debugf("containerConfig: %+v",containerConfig)
	linksSize := len(containerConfig.HostConfig.Links)
	volumesFrom := containerConfig.HostConfig.VolumesFrom
	volumesFromSize := len(containerConfig.HostConfig.VolumesFrom)
	env := containerConfig.Env
	containers := cluster.Containers()
	linkSet := make(map[string]bool)
	links := make([]string,0)
	var v int  // count of volumesFrom links validated 
	var l int  // count of links validated
	var affinityContainerRef string  // docker affinity container env var ( affinity:container==<container ref> ) 
	var affinityContainerIndex int   // index of affinity container env element in env[]
	var affinityLabelRef string  // docker affinity label env var ( affinity:<label>==<label value> ) 
	var affinityLabelIndex int   // index of affinity label env element in env[]
	// check for affinity in environment vars.  
	// If yes save it for checking whether it belongs to tenant.
	affinityContainerCheckRequired := false
	affinityLabelCheckRequired := false
	for i, e := range env {
		if strings.HasPrefix(e,"affinity:") {
		  if strings.HasPrefix(e,"affinity:image==") {
			break  // we ignore affinity for images 
		  } else if strings.HasPrefix(e,"affinity:container==") {
			affinityContainerCheckRequired = true
			containerRefIndex := strings.Index(e,"affinity:container==")+len("affinity:container==")
			affinityContainerRef = e[containerRefIndex:len(e)]
			affinityContainerIndex = i
			break
		  } else {
			affinityLabelCheckRequired = true
			labelRefIndex := strings.Index(e,"affinity:")+len("affinity:")
			affinityLabelRef = e[labelRefIndex:len(e)]
			log.Debug("affinityLabelRef: ",affinityLabelRef)
			affinityLabelIndex = i
			break
		  }
		}

	}

		

	if linksSize < 1 && volumesFromSize < 1 && !affinityContainerCheckRequired && !affinityLabelCheckRequired {
		return true, &ValidationOutPutDTO{ContainerID: ""}
	}
	for _, container := range containers {
		//log.Debugf("Examine container %s %s",container.Info.Name,container.Info.ID)
		if(container.Config.Labels[headers.TenancyLabel] == tenantName) {
			log.Debugf("Look for container references in container %s %s for tenant %s",container.Info.Name,container.Info.ID,tenantName)
			// check for volumeFrom reference
			for i := 0; i < volumesFromSize; i++ {
				if v == volumesFromSize {
					break
				}
				log.Debugf("Examine VolumeFrom[%d] == %s", i, containerConfig.HostConfig.VolumesFrom[i])
				// volumesFrom element format <container_name>:<RW|RO>
				volumeFromArray := strings.SplitN(strings.TrimSpace(containerConfig.HostConfig.VolumesFrom[i]),":",2)
				volumeFrom := strings.TrimSpace(volumeFromArray[0])				
				if strings.HasPrefix(container.Info.ID,volumeFrom) {
					log.Debug("volumesFrom element with container id matches tenant container")
					// no need to modify volumesFrom
					v++					
				} else if container.Info.Name == "/"+volumeFrom+tenantName {
					log.Debug("volumesFrom element with container name matches tenant container")
					volumesFrom[i] = container.Info.ID
					if len(volumeFromArray) > 1 {
						volumesFrom[i] += ":"
						volumesFrom[i] += strings.TrimSpace(volumeFromArray[1])
					}
					v++					
				}
			}
			// check for links reference
			for i := 0; i < linksSize; i++ {
				if l == linksSize {
						break
				}
				log.Debugf("Examine links[%d] == %s", i, containerConfig.HostConfig.Links[i])

				linkArray := strings.SplitN(containerConfig.HostConfig.Links[i],":",2)
				link := strings.TrimSpace(linkArray[0])
				if strings.HasPrefix(container.Info.ID,link) || "/"+link+tenantName == container.Info.Name {
					log.Debug("Add link and alias to linkset")
					_, ok := linkSet[link]
					if !ok {
						linkSet[link] = true
						links = append(links,container.Info.ID + ":" + link)						
					}
					// check for alias  
					if len(linkArray) > 1 {						
						links = append(links,container.Info.ID + ":" + strings.TrimSpace(linkArray[1]))
					}
					l++
				}
			}
			// check for affinity:container==<container> reference
			// modify affinity container environment variable to reference container+tenantName
			if affinityContainerCheckRequired {
				if strings.HasPrefix(container.Info.ID,affinityContainerRef) {
				  affinityContainerCheckRequired = false				     
				} else if container.Info.Name == "/"+affinityContainerRef+tenantName {
				  env[affinityContainerIndex] = env[affinityContainerIndex] + tenantName
				  log.Debugf("Updated environment variable %s ",env[affinityContainerIndex])
				  affinityContainerCheckRequired = false	  
				}
			}
			// check for affinity:<label key>==<label value> reference
			// modify affinity label container environment variable with affinity container env var to reference container id
			if affinityLabelCheckRequired {
				kv := strings.Split(affinityLabelRef,"==")
				for k,v := range container.Config.Labels {
					if k == kv[0] && v == kv[1] {
						affinityLabelCheckRequired=false
						env[affinityLabelIndex] = "affinity:container==" + container.Info.ID
						log.Debugf("Updated environment variable %s ",env[affinityLabelIndex])
						break						
					} 
				}
			}

		}
		// are we done?
		if v == volumesFromSize && l == linksSize && !affinityContainerCheckRequired && !affinityLabelCheckRequired {
			break
		}
	}

	if v != volumesFromSize || l != linksSize || affinityContainerCheckRequired || affinityLabelCheckRequired {
		return false, &ValidationOutPutDTO{ContainerID: "", ErrorMessage: "Tenant references containers that it does not own."}
	}
	return true, &ValidationOutPutDTO{ContainerID: "", Links: links, VolumesFrom: volumesFrom, Env: env}

}



type Config struct {
    HostConfig struct {
		Links []interface{}
    	    VolumesFrom []interface{}
	}
}
/*
func CheckConfigOwnerShip(cluster cluster.Cluster, tenantName string, r *http.Request, reqBody []byte) (bool, *ValidationOutPutDTO) {
	
	config := &Config{}
	
	err := json.Unmarshal(reqBody, &config)
    if err != nil {
        panic(err)
    }

	log.Debug("*******************XXXXXXXXXX*******************************")
	log.Debug(fmt.Println(config.HostConfig.Links))
	log.Debug(fmt.Println(config.HostConfig.VolumesFrom))
	
	containers := cluster.Containers()
	
	log.Debug("*******************OOOOOOOOOOOOOOOO***********************")
	log.Debugf("vLength: %d", len(config.HostConfig.VolumesFrom))
	log.Debugf("lLength: %d", len(config.HostConfig.Links))
	log.Debug("*******************OOOOOOOOOOOOOOOO***********************")
	log.Debug("LL: " + string(len(config.HostConfig.Links)))
	
	volSet := make(map[string]string)
	linkSet := make(map[string]string)
	var v int
	var l int
	
	if len(config.HostConfig.VolumesFrom) != 0 || len(config.HostConfig.Links) != 0 {
		for _, container := range containers {
			log.Debug("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
			
			for _, volume := range config.HostConfig.VolumesFrom {
				vol := volume.(string)
				log.Debug("VolumesFrom: #" + vol + "#")
				log.Debug("containerName: " + container.Info.Name)
				log.Debug("Comparing with: " + "/" + vol + tenantName)
				
				if "/" + vol + tenantName == container.Info.Name || "/"+vol == container.Info.Name {
					volSet[container.Info.ID] = vol
					v++
				}
			}
			for _, link := range config.HostConfig.Links {
				link := link.(string)
				log.Debug("Link: #" + link + "#")
				log.Debug("containerName: " + container.Info.Name)
				log.Debug("Comparing with: " + "/" + link + tenantName)
				
				if "/" + link + tenantName == container.Info.Name || "/"+link == container.Info.Name {
					linkSet[container.Info.ID] = link
					l++
				}
			}
		}
		
		if v != len(config.HostConfig.VolumesFrom) || l != len(config.HostConfig.Links) {
			return false, nil
		}
	}
	log.Debug("*********************XXXXXXXXX*****************************")
	
	res := ValidationOutPutDTO{ContainerID: "", Links: linkSet, VolumesFrom: volSet}
	return true, &res
}
*/

//TODO - Pass by ref ?
func CheckOwnerShip(cluster cluster.Cluster, tenantName string, r *http.Request) (bool, *ValidationOutPutDTO) {
	containers := cluster.Containers()
	log.Debug("got name: ", mux.Vars(r)["name"])
	if mux.Vars(r)["name"] == ""{
		return true, nil
	}
	tenantSet := make(map[string]bool)
	for _, container := range containers {
		names := container.Names
		for _, name := range names {
			if "/"+mux.Vars(r)["name"]+tenantName == name {
				log.Debug("Match by name.")
				return true, &ValidationOutPutDTO{ContainerID: container.Info.ID, Links: nil}
			}
		}
		if "/"+mux.Vars(r)["name"] == container.Info.Name {
			if container.Labels[headers.TenancyLabel] == tenantName {
				return true, &ValidationOutPutDTO{ContainerID: container.Info.ID, Links: nil}
			}
		} else if mux.Vars(r)["name"] == container.Info.ID {
			log.Debug("Match By full ID! Checking Ownership...")
			log.Debug("Tenant name: ", tenantName)
			log.Debug("Tenant Lable: ", container.Labels[headers.TenancyLabel])
			if container.Labels[headers.TenancyLabel] == tenantName {
				return true, &ValidationOutPutDTO{ContainerID: container.Info.ID, Links: nil}
			}
			return false, nil

		}
		if container.Labels[headers.TenancyLabel] == tenantName {
			tenantSet[container.Info.ID] = true
		}
	}

	//Handle short ID
	ambiguityCounter := 0
	var returnID string
	for k := range tenantSet {
		if strings.HasPrefix(cluster.Container(k).Info.ID, mux.Vars(r)["name"]) {
			ambiguityCounter++
			returnID = cluster.Container(k).Info.ID
		}
		if ambiguityCounter == 1 {
			log.Debug("Matched by short ID")
			return true, &ValidationOutPutDTO{ContainerID: returnID, Links: nil}
		}
		if ambiguityCounter > 1 {
			log.Debug("Ambiguiy by short ID")
			//TODO - ambiguity
		}
		if ambiguityCounter == 0 {
			log.Debug("No match by short ID")
			//TODO - no such container
		}
	}
	return false, nil
}

func CleanUpLabeling(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	newBody := bytes.Replace(rec.Body.Bytes(), []byte(headers.TenancyLabel), []byte(" "), -1)
	//TODO - Here we just use the token for the tenant name for now so we remove it from the data before returning to user.
	newBody = bytes.Replace(newBody, []byte(r.Header.Get(headers.AuthZTenantIdHeaderName)), []byte(""), -1)
	newBody = bytes.Replace(newBody, []byte(",\" \":\" \""), []byte(""), -1)
	log.Debug("Got this new body...", string(newBody))
	return newBody
}

func ParseField(field string, fieldType interface{}, body []byte) (interface{}, error) {
	log.Debugf("In parseField, field: %s Request body: %s", field, string(body))
	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		log.Error("failed to parse!")
		return nil, err
	}

	switch v := fieldType.(type) {
	case float64:
		log.Debug("Parsing type: ", v)
		parsedField, ok := jsonParsed.Path(field).Data().(float64)
		if ok {
			res := strconv.FormatFloat(parsedField, 'f', -1, 64)
			log.Debugf("Parsed field: " + res)
			return parsedField, nil
		}
	case []string:
		log.Debug("Parsing type: ", v)
		parsedField, ok := jsonParsed.Path(field).Data().([]string)
		if ok {
			log.Debug(parsedField)
			return parsedField, nil
		}
	default:
		log.Error("Unknown field type to parse")
	}

	return nil, errors.New(fmt.Sprintf("failed to parse field %s from request body %s", field, string(body)))
}

/*
ParseID - parse the body and return Id.
*/
func ParseID(body []byte) (string, error) {
	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		log.Error("failed to parse!")
		return "", err
	}
	return jsonParsed.Path("Id").Data().(string), nil
}

// RandStringBytesRmndr used to generate a name for docker volume create when no name is supplied
// The tenant id is then appended to the name by the caller
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
func RandStringBytesRmndr(n int) string {
    b := make([]byte, n)
    for i := range b {
        b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
    }
    return string(b)
}