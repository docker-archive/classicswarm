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

package e2e

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/kubernetes/pkg/watch"

	. "github.com/onsi/ginkgo"
)

type durations []time.Duration

func (d durations) Len() int           { return len(d) }
func (d durations) Less(i, j int) bool { return d[i] < d[j] }
func (d durations) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }

var _ = Describe("Service endpoints latency", func() {
	f := NewFramework("svc-latency")

	It("should not be very high", func() {
		const (
			// These are very generous criteria. Ideally we will
			// get this much lower in the future. See issue
			// #10436.
			limitMedian = time.Second * 20
			limitTail   = time.Second * 50

			// Numbers chosen to make the test complete in a short amount
			// of time. This sample size is not actually large enough to
			// reliably measure tails (it may give false positives, but not
			// false negatives), but it should catch low hanging fruit.
			//
			// Note that these are fixed and do not depend on the
			// size of the cluster. Setting parallelTrials larger
			// distorts the measurements. Perhaps this wouldn't be
			// true on HA clusters.
			totalTrials    = 200
			parallelTrials = 15
			minSampleSize  = 100
		)

		// Turn off rate limiting--it interferes with our measurements.
		oldThrottle := f.Client.RESTClient.Throttle
		f.Client.RESTClient.Throttle = util.NewFakeRateLimiter()
		defer func() { f.Client.RESTClient.Throttle = oldThrottle }()

		failing := sets.NewString()
		d, err := runServiceLatencies(f, parallelTrials, totalTrials)
		if err != nil {
			failing.Insert(fmt.Sprintf("Not all RC/pod/service trials succeeded: %v", err))
		}
		dSorted := durations(d)
		sort.Sort(dSorted)
		n := len(dSorted)
		if n < minSampleSize {
			failing.Insert(fmt.Sprintf("Did not get a good sample size: %v", dSorted))
		}
		if n < 2 {
			failing.Insert("Less than two runs succeeded; aborting.")
			Fail(strings.Join(failing.List(), "\n"))
		}
		percentile := func(p int) time.Duration {
			est := n * p / 100
			if est >= n {
				return dSorted[n-1]
			}
			return dSorted[est]
		}
		Logf("Latencies: %v", dSorted)
		p50 := percentile(50)
		p90 := percentile(90)
		p99 := percentile(99)
		Logf("50 %%ile: %v", p50)
		Logf("90 %%ile: %v", p90)
		Logf("99 %%ile: %v", p99)
		Logf("Total sample count: %v", len(dSorted))

		if p50 > limitMedian {
			failing.Insert("Median latency should be less than " + limitMedian.String())
		}
		if p99 > limitTail {
			failing.Insert("Tail (99 percentile) latency should be less than " + limitTail.String())
		}
		if failing.Len() > 0 {
			errList := strings.Join(failing.List(), "\n")
			helpfulInfo := fmt.Sprintf("\n50, 90, 99 percentiles: %v %v %v", p50, p90, p99)
			Fail(errList + helpfulInfo)
		}
	})
})

func runServiceLatencies(f *Framework, inParallel, total int) (output []time.Duration, err error) {
	cfg := RCConfig{
		Client:       f.Client,
		Image:        "gcr.io/google_containers/pause:1.0",
		Name:         "svc-latency-rc",
		Namespace:    f.Namespace.Name,
		Replicas:     1,
		PollInterval: time.Second,
	}
	if err := RunRC(cfg); err != nil {
		return nil, err
	}
	defer DeleteRC(f.Client, f.Namespace.Name, cfg.Name)

	// Run a single watcher, to reduce the number of API calls we have to
	// make; this is to minimize the timing error. It's how kube-proxy
	// consumes the endpoints data, so it seems like the right thing to
	// test.
	endpointQueries := newQuerier()
	startEndpointWatcher(f, endpointQueries)
	defer close(endpointQueries.stop)

	// run one test and throw it away-- this is to make sure that the pod's
	// ready status has propagated.
	singleServiceLatency(f, cfg.Name, endpointQueries)

	// These channels are never closed, and each attempt sends on exactly
	// one of these channels, so the sum of the things sent over them will
	// be exactly total.
	errs := make(chan error, total)
	durations := make(chan time.Duration, total)

	blocker := make(chan struct{}, inParallel)
	for i := 0; i < total; i++ {
		go func() {
			defer GinkgoRecover()
			blocker <- struct{}{}
			defer func() { <-blocker }()
			if d, err := singleServiceLatency(f, cfg.Name, endpointQueries); err != nil {
				errs <- err
			} else {
				durations <- d
			}
		}()
	}

	errCount := 0
	for i := 0; i < total; i++ {
		select {
		case e := <-errs:
			Logf("Got error: %v", e)
			errCount += 1
		case d := <-durations:
			output = append(output, d)
		}
	}
	if errCount != 0 {
		return output, fmt.Errorf("got %v errors", errCount)
	}
	return output, nil
}

type endpointQuery struct {
	endpointsName string
	endpoints     *api.Endpoints
	result        chan<- struct{}
}

type endpointQueries struct {
	requests map[string]*endpointQuery

	stop        chan struct{}
	requestChan chan *endpointQuery
	seenChan    chan *api.Endpoints
}

func newQuerier() *endpointQueries {
	eq := &endpointQueries{
		requests: map[string]*endpointQuery{},

		stop:        make(chan struct{}, 100),
		requestChan: make(chan *endpointQuery),
		seenChan:    make(chan *api.Endpoints, 100),
	}
	go eq.join()
	return eq
}

// join merges the incoming streams of requests and added endpoints. It has
// nice properties like:
//  * remembering an endpoint if it happens to arrive before it is requested.
//  * closing all outstanding requests (returning nil) if it is stopped.
func (eq *endpointQueries) join() {
	defer func() {
		// Terminate all pending requests, so that no goroutine will
		// block indefinitely.
		for _, req := range eq.requests {
			if req.result != nil {
				close(req.result)
			}
		}
	}()

	for {
		select {
		case <-eq.stop:
			return
		case req := <-eq.requestChan:
			if cur, ok := eq.requests[req.endpointsName]; ok && cur.endpoints != nil {
				// We've already gotten the result, so we can
				// immediately satisfy this request.
				delete(eq.requests, req.endpointsName)
				req.endpoints = cur.endpoints
				close(req.result)
			} else {
				// Save this request.
				eq.requests[req.endpointsName] = req
			}
		case got := <-eq.seenChan:
			if req, ok := eq.requests[got.Name]; ok {
				if req.result != nil {
					// Satisfy a request.
					delete(eq.requests, got.Name)
					req.endpoints = got
					close(req.result)
				} else {
					// We've already recorded a result, but
					// haven't gotten the request yet. Only
					// keep the first result.
				}
			} else {
				// We haven't gotten the corresponding request
				// yet, save this result.
				eq.requests[got.Name] = &endpointQuery{
					endpoints: got,
				}
			}
		}
	}
}

// request blocks until the requested endpoint is seen.
func (eq *endpointQueries) request(endpointsName string) *api.Endpoints {
	result := make(chan struct{})
	req := &endpointQuery{
		endpointsName: endpointsName,
		result:        result,
	}
	eq.requestChan <- req
	<-result
	return req.endpoints
}

// marks e as added; does not block.
func (eq *endpointQueries) added(e *api.Endpoints) {
	eq.seenChan <- e
}

// blocks until it has finished syncing.
func startEndpointWatcher(f *Framework, q *endpointQueries) {
	_, controller := framework.NewInformer(
		&cache.ListWatch{
			ListFunc: func() (runtime.Object, error) {
				return f.Client.Endpoints(f.Namespace.Name).List(labels.Everything())
			},
			WatchFunc: func(rv string) (watch.Interface, error) {
				return f.Client.Endpoints(f.Namespace.Name).Watch(labels.Everything(), fields.Everything(), rv)
			},
		},
		&api.Endpoints{},
		0,
		framework.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if e, ok := obj.(*api.Endpoints); ok {
					if len(e.Subsets) > 0 && len(e.Subsets[0].Addresses) > 0 {
						q.added(e)
					}
				}
			},
			UpdateFunc: func(old, cur interface{}) {
				if e, ok := cur.(*api.Endpoints); ok {
					if len(e.Subsets) > 0 && len(e.Subsets[0].Addresses) > 0 {
						q.added(e)
					}
				}
			},
		},
	)

	go controller.Run(q.stop)

	// Wait for the controller to sync, so that we don't count any warm-up time.
	for !controller.HasSynced() {
		time.Sleep(100 * time.Millisecond)
	}
}

func singleServiceLatency(f *Framework, name string, q *endpointQueries) (time.Duration, error) {
	// Make a service that points to that pod.
	svc := &api.Service{
		ObjectMeta: api.ObjectMeta{
			GenerateName: "latency-svc-",
		},
		Spec: api.ServiceSpec{
			Ports:           []api.ServicePort{{Protocol: api.ProtocolTCP, Port: 80}},
			Selector:        map[string]string{"name": name},
			Type:            api.ServiceTypeClusterIP,
			SessionAffinity: api.ServiceAffinityNone,
		},
	}
	startTime := time.Now()
	gotSvc, err := f.Client.Services(f.Namespace.Name).Create(svc)
	if err != nil {
		return 0, err
	}
	Logf("Created: %v", gotSvc.Name)
	defer f.Client.Services(gotSvc.Namespace).Delete(gotSvc.Name)

	if e := q.request(gotSvc.Name); e == nil {
		return 0, fmt.Errorf("Never got a result for endpoint %v", gotSvc.Name)
	}
	stopTime := time.Now()
	d := stopTime.Sub(startTime)
	Logf("Got endpoints: %v [%v]", gotSvc.Name, d)
	return d, nil
}
