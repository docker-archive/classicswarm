/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
This soak tests places a specified number of pods on each node and then
repeatedly sends queries to a service running on these pods via
a serivce
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util"
)

var (
	queriesAverage = flag.Int("queries", 100, "Number of hostname queries to make in each iteration per pod on average")
	podsPerNode    = flag.Int("pods_per_node", 1, "Number of serve_hostname pods per node")
	upTo           = flag.Int("up_to", 1, "Number of iterations or -1 for no limit")
	maxPar         = flag.Int("max_par", 500, "Maximum number of queries in flight")
	gke            = flag.String("gke_context", "", "Target GKE cluster with context gke_{project}_{zone}_{cluster-name}")
)

const (
	deleteTimeout          = 2 * time.Minute
	endpointTimeout        = 5 * time.Minute
	nodeListTimeout        = 2 * time.Minute
	podCreateTimeout       = 2 * time.Minute
	podStartTimeout        = 30 * time.Minute
	serviceCreateTimeout   = 2 * time.Minute
	namespaceDeleteTimeout = 5 * time.Minute
)

func main() {
	flag.Parse()

	glog.Infof("Starting serve_hostnames soak test with queries=%d and podsPerNode=%d upTo=%d",
		*queriesAverage, *podsPerNode, *upTo)

	var spec string
	if *gke != "" {
		spec = filepath.Join(os.Getenv("HOME"), ".config", "gcloud", "kubernetes", "kubeconfig")
	} else {
		spec = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	settings, err := clientcmd.LoadFromFile(spec)
	if err != nil {
		glog.Fatalf("Error loading configuration: %v", err.Error())
	}
	if *gke != "" {
		settings.CurrentContext = *gke
	}
	config, err := clientcmd.NewDefaultClientConfig(*settings, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		glog.Fatalf("Failed to construct config: %v", err)
	}

	c, err := client.New(config)
	if err != nil {
		glog.Fatalf("Failed to make client: %v", err)
	}

	var nodes *api.NodeList
	for start := time.Now(); time.Since(start) < nodeListTimeout; time.Sleep(2 * time.Second) {
		nodes, err = c.Nodes().List(labels.Everything(), fields.Everything())
		if err == nil {
			break
		}
		glog.Warningf("Failed to list nodes: %v", err)
	}
	if err != nil {
		glog.Fatalf("Giving up trying to list nodes: %v", err)
	}

	if len(nodes.Items) == 0 {
		glog.Fatalf("Failed to find any nodes.")
	}

	glog.Infof("Found %d nodes on this cluster:", len(nodes.Items))
	for i, node := range nodes.Items {
		glog.Infof("%d: %s", i, node.Name)
	}

	queries := *queriesAverage * len(nodes.Items) * *podsPerNode

	// Create the namespace
	got, err := c.Namespaces().Create(&api.Namespace{ObjectMeta: api.ObjectMeta{GenerateName: "serve-hostnames-"}})
	if err != nil {
		glog.Fatalf("Failed to create namespace: %v", err)
	}
	ns := got.Name
	defer func(ns string) {
		if err := c.Namespaces().Delete(ns); err != nil {
			glog.Warningf("Failed to delete namespace ns: %e", ns, err)
		} else {
			// wait until the namespace disappears
			for i := 0; i < int(namespaceDeleteTimeout/time.Second); i++ {
				if _, err := c.Namespaces().Get(ns); err != nil {
					if errors.IsNotFound(err) {
						return
					}
				}
				time.Sleep(time.Second)
			}
		}
	}(ns)
	glog.Infof("Created namespace %s", ns)

	// Create a service for these pods.
	glog.Infof("Creating service %s/serve-hostnames", ns)
	// Make several attempts to create a service.
	var svc *api.Service
	for start := time.Now(); time.Since(start) < serviceCreateTimeout; time.Sleep(2 * time.Second) {
		t := time.Now()
		svc, err = c.Services(ns).Create(&api.Service{
			ObjectMeta: api.ObjectMeta{
				Name: "serve-hostnames",
				Labels: map[string]string{
					"name": "serve-hostname",
				},
			},
			Spec: api.ServiceSpec{
				Ports: []api.ServicePort{{
					Protocol:   "TCP",
					Port:       9376,
					TargetPort: util.NewIntOrStringFromInt(9376),
				}},
				Selector: map[string]string{
					"name": "serve-hostname",
				},
			},
		})
		glog.V(4).Infof("Service create %s/server-hostnames took %v", ns, time.Since(t))
		if err == nil {
			break
		}
		glog.Warningf("After %v failed to create service %s/serve-hostnames: %v", time.Since(start), ns, err)
	}
	if err != nil {
		glog.Warningf("Unable to create service %s/%s: %v", ns, svc.Name, err)
		return
	}
	// Clean up service
	defer func() {
		glog.Infof("Cleaning up service %s/serve-hostnames", ns)
		// Make several attempts to delete the service.
		for start := time.Now(); time.Since(start) < deleteTimeout; time.Sleep(1 * time.Second) {
			if err := c.Services(ns).Delete(svc.Name); err == nil {
				return
			}
			glog.Warningf("After %v unable to delete service %s/%s: %v", time.Since(start), ns, svc.Name, err)
		}
	}()

	// Put serve-hostname pods on each node.
	podNames := []string{}
	for i, node := range nodes.Items {
		for j := 0; j < *podsPerNode; j++ {
			podName := fmt.Sprintf("serve-hostname-%d-%d", i, j)
			podNames = append(podNames, podName)
			// Make several attempts
			for start := time.Now(); time.Since(start) < podCreateTimeout; time.Sleep(2 * time.Second) {
				glog.Infof("Creating pod %s/%s on node %s", ns, podName, node.Name)
				t := time.Now()
				_, err = c.Pods(ns).Create(&api.Pod{
					ObjectMeta: api.ObjectMeta{
						Name: podName,
						Labels: map[string]string{
							"name": "serve-hostname",
						},
					},
					Spec: api.PodSpec{
						Containers: []api.Container{
							{
								Name:  "serve-hostname",
								Image: "gcr.io/google_containers/serve_hostname:1.1",
								Ports: []api.ContainerPort{{ContainerPort: 9376}},
							},
						},
						NodeName: node.Name,
					},
				})
				glog.V(4).Infof("Pod create %s/%s request took %v", ns, podName, time.Since(t))
				if err == nil {
					break
				}
				glog.Warningf("After %s failed to create pod %s/%s: %v", time.Since(start), ns, podName, err)
			}
			if err != nil {
				glog.Warningf("Failed to create pod %s/%s: %v", ns, podName, err)
				return
			}
		}
	}
	// Clean up the pods
	defer func() {
		glog.Info("Cleaning up pods")
		// Make several attempts to delete the pods.
		for _, podName := range podNames {
			for start := time.Now(); time.Since(start) < deleteTimeout; time.Sleep(1 * time.Second) {
				if err = c.Pods(ns).Delete(podName, nil); err == nil {
					break
				}
				glog.Warningf("After %v failed to delete pod %s/%s: %v", time.Since(start), ns, podName, err)
			}
		}
	}()

	glog.Info("Waiting for the serve-hostname pods to be ready")
	for _, podName := range podNames {
		var pod *api.Pod
		for start := time.Now(); time.Since(start) < podStartTimeout; time.Sleep(5 * time.Second) {
			pod, err = c.Pods(ns).Get(podName)
			if err != nil {
				glog.Warningf("Get pod %s/%s failed, ignoring for %v: %v", ns, podName, err, podStartTimeout)
				continue
			}
			if pod.Status.Phase == api.PodRunning {
				break
			}
		}
		if pod.Status.Phase != api.PodRunning {
			glog.Warningf("Gave up waiting on pod %s/%s to be running (saw %v)", ns, podName, pod.Status.Phase)
		} else {
			glog.Infof("%s/%s is running", ns, podName)
		}
	}

	// Wait for the endpoints to propagate.
	for start := time.Now(); time.Since(start) < endpointTimeout; time.Sleep(10 * time.Second) {
		hostname, err := c.Get().
			Namespace(ns).
			Prefix("proxy").
			Resource("services").
			Name("serve-hostnames").
			DoRaw()
		if err != nil {
			glog.Infof("After %v while making a proxy call got error %v", time.Since(start), err)
			continue
		}
		var r unversioned.Status
		if err := api.Scheme.DecodeInto(hostname, &r); err != nil {
			break
		}
		if r.Status == unversioned.StatusFailure {
			glog.Infof("After %v got status %v", time.Since(start), string(hostname))
			continue
		}
		break
	}

	// Repeatedly make requests.
	for iteration := 0; iteration != *upTo; iteration++ {
		responseChan := make(chan string, queries)
		// Use a channel of size *maxPar to throttle the number
		// of in-flight requests to avoid overloading the service.
		inFlight := make(chan struct{}, *maxPar)
		start := time.Now()
		for q := 0; q < queries; q++ {
			go func(i int, query int) {
				inFlight <- struct{}{}
				t := time.Now()
				hostname, err := c.Get().
					Namespace(ns).
					Prefix("proxy").
					Resource("services").
					Name("serve-hostnames").
					DoRaw()
				glog.V(4).Infof("Proxy call in namespace %s took %v", ns, time.Since(t))
				if err != nil {
					glog.Warningf("Call failed during iteration %d query %d : %v", i, query, err)
					// If the query failed return a string which starts with a character
					// that can't be part of a hostname.
					responseChan <- fmt.Sprintf("!failed in iteration %d to issue query %d: %v", i, query, err)
				} else {
					responseChan <- string(hostname)
				}
				<-inFlight
			}(iteration, q)
		}
		responses := make(map[string]int, *podsPerNode*len(nodes.Items))
		missing := 0
		for q := 0; q < queries; q++ {
			r := <-responseChan
			glog.V(4).Infof("Got response from %s", r)
			responses[r]++
			// If the returned hostname starts with '!' then it indicates
			// an error response.
			if len(r) > 0 && r[0] == '!' {
				glog.V(3).Infof("Got response %s", r)
				missing++
			}
		}
		if missing > 0 {
			glog.Warningf("Missing %d responses out of %d", missing, queries)
		}
		// Report any nodes that did not respond.
		for n, node := range nodes.Items {
			for i := 0; i < *podsPerNode; i++ {
				name := fmt.Sprintf("serve-hostname-%d-%d", n, i)
				if _, ok := responses[name]; !ok {
					glog.Warningf("No response from pod %s on node %s at iteration %d", name, node.Name, iteration)
				}
			}
		}
		glog.Infof("Iteration %d took %v for %d queries (%.2f QPS) with %d missing",
			iteration, time.Since(start), queries-missing, float64(queries-missing)/time.Since(start).Seconds(), missing)
	}
}
