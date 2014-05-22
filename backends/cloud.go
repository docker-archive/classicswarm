//
// Copyright (C) 2013 The Docker Cloud authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package backends

import (
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/engine"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

// The Cloud interface provides the contract that cloud providers should implement to enable
// running Docker containers in their cloud.
// TODO(bburns): Restructure this into Cloud, Instance and Tunnel interfaces
type Cloud interface {
	// GetPublicIPAddress returns the stringified address (e.g "1.2.3.4") of the runtime
	GetPublicIPAddress(name string, zone string) (string, error)

	// CreateInstance creates a virtual machine instance given a name and a zone.  Returns the
	// IP address of the instance.  Waits until Docker is up and functioning on the machine
	// before returning.
	CreateInstance(name string, zone string) (string, error)

	// DeleteInstance deletes a virtual machine instance, given the instance name and zone.
	DeleteInstance(name string, zone string) error

	// Open a secure tunnel (generally SSH) between the local host and a remote host.
	OpenSecureTunnel(name string, zone string, localPort int, remotePort int) (*os.Process, error)
}

func CloudBackend() engine.Installer {
	return &cloud{}
}

type cloud struct {
}

type Container struct {
	Id	string
	Image string
	Tty bool
}

// returns true, if the connection was successful, false otherwise
type Tunnel struct {
        url.URL
}

func (t Tunnel) isActive() bool {
        _, err := http.Get(t.String())
        return err == nil
}

func (s *cloud) Install(eng *engine.Engine) error {
	eng.Register("cloud", func(job *engine.Job) engine.Status {
		if len(job.Args) < 3 {
			return job.Errorf("usage: %s <provider> <instance> <zone> <aditional> <provider> <args> <proto>://<addr>", job.Name)
		}
		instance := job.Args[1]
		zone := job.Args[2]
		var cloud Cloud
		var err error
		switch job.Args[0] {
		case "gce":
			if len(job.Args) < 4 {
				return job.Errorf("usage: %s gce <instance> <zone> <project>")
			}
			cloud, err = NewCloudGCE(job.Args[3])
			if err != nil {
				job.Errorf("Unexpected error: %#v", err)
			}
		default:
			return job.Errorf("Unknown cloud provider: %s", job.Args[0])
		}
		ip, err := cloud.GetPublicIPAddress(instance, zone)
		instanceRunning := len(ip) > 0
		if !instanceRunning {
			log.Print("Instance doesn't exist, creating....");
			_, err = cloud.CreateInstance(instance, zone)
		}
		if err != nil {
			job.Errorf("Unexpected error: %#v", err)
		}
		remotePort := 8000
		localPort := 8001
		apiVersion := "v1.10"
		tunnelUrl, err := url.Parse(fmt.Sprintf("http://localhost:%d/%s/containers/json", localPort, apiVersion))
        if err != nil {
                return job.Errorf("Unexpected error: %#v", err)
        }
        tunnel := Tunnel{*tunnelUrl}

        if !tunnel.isActive() {
                fmt.Printf("Creating tunnel")
				_, err = cloud.OpenSecureTunnel(instance, zone, localPort, remotePort)
				if err != nil {
					job.Errorf("Failed to open tunnel: %#v", err)
				}
		}
		host := fmt.Sprintf("tcp://localhost:%d", localPort)
		client, err := newClient(host, apiVersion)
		if err != nil {
			job.Errorf("Unexpected error: %#v", err)
		}
		//job.Eng.Register("inspect", func(job *engine.Job) engine.Status {
		//	resp, err := client.call("GET", "/containers/
		job.Eng.Register("create", func(job *engine.Job) engine.Status {
			container := Container{
				Image: job.Getenv("Image"),
				Tty: job.Getenv("Tty") == "true",
			}
			data, err := json.Marshal(container)
			resp, err := client.call("POST", "/containers/create", string(data))
			if err != nil {
				return job.Errorf("%s: post: %v", client.URL.String(), err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return job.Errorf("%s: read body: %#v", client.URL.String(), err)
			}
			var containerOut Container
			err = json.Unmarshal([]byte(body), &containerOut)
			_, err = job.Printf("%s\n", containerOut.Id)
			if err != nil {
				return job.Errorf("%s: write body: %#v", client.URL.String(), err)
			}
			log.Printf("%s", string(body))
			return engine.StatusOK
		})
		
		job.Eng.Register("start", func(job *engine.Job) engine.Status {
			path := fmt.Sprintf("/containers/%s/start", job.Args[0])
			resp, err := client.call("POST", path, "{\"Binds\":[],\"ContainerIDFile\":\"\",\"LxcConf\":[],\"Privileged\":false,\"PortBindings\":{},\"Links\":null,\"PublishAllPorts\":false,\"Dns\":null,\"DnsSearch\":[],\"VolumesFrom\":[]}")
			if err != nil {
				return job.Errorf("%s: post: %v", client.URL.String(), err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return job.Errorf("%s: read body: %#v", client.URL.String(), err)
			}
			log.Printf("%s", string(body))
			return engine.StatusOK
		})

		job.Eng.Register("containers", func(job *engine.Job) engine.Status {
			path := fmt.Sprintf(
				"/containers/json?all=%s&size=%s&since=%s&before=%s&limit=%s",
				url.QueryEscape(job.Getenv("all")),
				url.QueryEscape(job.Getenv("size")),
				url.QueryEscape(job.Getenv("since")),
				url.QueryEscape(job.Getenv("before")),
				url.QueryEscape(job.Getenv("limit")),
			)
			resp, err := client.call("GET", path, "")
			if err != nil {
				return job.Errorf("%s: get: %v", client.URL.String(), err)
			}
			// FIXME: check for response error
			c := engine.NewTable("Created", 0)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return job.Errorf("%s: read body: %v", client.URL.String(), err)
			}
			fmt.Printf("---> '%s'\n", body)
			if _, err := c.ReadListFrom(body); err != nil {
				return job.Errorf("%s: readlist: %v", client.URL.String(), err)
			}
			c.WriteListTo(job.Stdout)
			return engine.StatusOK
		})
		
		job.Eng.Register("container_delete", func(job *engine.Job) engine.Status {
			log.Printf("%#v", job.Args)
			path := "/containers/" + job.Args[0] 
			
			resp, err := client.call("DELETE", path, "")
			if err != nil {
				return job.Errorf("%s: delete: %v", client.URL.String(), err)
			}
			log.Printf("%#v", resp)
			return engine.StatusOK
		})
		
		job.Eng.Register("stop", func(job *engine.Job) engine.Status {
			log.Printf("%#v", job.Args)
			path := "/containers/" + job.Args[0] + "/stop"
			
			resp, err := client.call("POST", path, "")
			if err != nil {
				return job.Errorf("%s: delete: %v", client.URL.String(), err)
			}
			log.Printf("%#v", resp)
			return engine.StatusOK
		})
		
		job.Eng.RegisterCatchall(func(job *engine.Job) engine.Status {
			log.Printf("%#v %#v %#v", *job, job.Env(), job.Args)
			return engine.StatusOK
		})
		return engine.StatusOK
	})
	return nil
}
