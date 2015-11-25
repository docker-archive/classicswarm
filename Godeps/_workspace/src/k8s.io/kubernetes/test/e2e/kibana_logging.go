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
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kibana Logging Instances Is Alive", func() {
	f := NewFramework("kibana-logging")

	BeforeEach(func() {
		// TODO: For now assume we are only testing cluster logging with Elasticsearch
		// and Kibana on GCE. Once we are sure that Elasticsearch and Kibana cluster level logging
		// works for other providers we should widen this scope of this test.
		SkipUnlessProviderIs("gce")
	})

	It("should check that the Kibana logging instance is alive", func() {
		ClusterLevelLoggingWithKibana(f)
	})
})

const (
	kibanaKey   = "k8s-app"
	kibanaValue = "kibana-logging"
)

// ClusterLevelLoggingWithKibana is an end to end test that checks to see if Kibana is alive.
func ClusterLevelLoggingWithKibana(f *Framework) {
	// graceTime is how long to keep retrying requests for status information.
	const graceTime = 2 * time.Minute

	// Check for the existence of the Kibana service.
	By("Checking the Kibana service exists.")
	s := f.Client.Services(api.NamespaceSystem)
	// Make a few attempts to connect. This makes the test robust against
	// being run as the first e2e test just after the e2e cluster has been created.
	var err error
	for start := time.Now(); time.Since(start) < graceTime; time.Sleep(5 * time.Second) {
		if _, err = s.Get("kibana-logging"); err == nil {
			break
		}
		Logf("Attempt to check for the existence of the Kibana service failed after %v", time.Since(start))
	}
	Expect(err).NotTo(HaveOccurred())

	// Wait for the Kibana pod(s) to enter the running state.
	By("Checking to make sure the Kibana pods are running")
	label := labels.SelectorFromSet(labels.Set(map[string]string{kibanaKey: kibanaValue}))
	pods, err := f.Client.Pods(api.NamespaceSystem).List(label, fields.Everything())
	Expect(err).NotTo(HaveOccurred())
	for _, pod := range pods.Items {
		err = waitForPodRunningInNamespace(f.Client, pod.Name, api.NamespaceSystem)
		Expect(err).NotTo(HaveOccurred())
	}

	By("Checking to make sure we get a response from the Kibana UI.")
	err = nil
	for start := time.Now(); time.Since(start) < graceTime; time.Sleep(5 * time.Second) {
		// Query against the root URL for Kibana.
		_, err = f.Client.Get().
			Namespace(api.NamespaceSystem).
			Prefix("proxy").
			Resource("services").
			Name("kibana-logging").
			DoRaw()
		if err != nil {
			Logf("After %v proxy call to kibana-logging failed: %v", time.Since(start), err)
			continue
		}
		break
	}
	Expect(err).NotTo(HaveOccurred())
}
