// +build integration,!no-etcd

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

package integration

import (
	"testing"

	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/test/integration/framework"
)

func TestKubectlValidation(t *testing.T) {
	testCases := []struct {
		data string
		err  bool
	}{
		{`{"apiVersion": "v1", "kind": "thisObjectShouldNotExistInAnyGroup"}`, true},
		{`{"apiVersion": "invalidVersion", "kind": "Pod"}`, true},
		{`{"apiVersion": "v1", "kind": "Pod"}`, false},

		// The following test the experimental api.
		// TOOD: Replace with something more robust. These may move.
		{`{"apiVersion": "experimental/v1alpha1", "kind": "DaemonSet"}`, false},
		{`{"apiVersion": "experimental/v1alpha1", "kind": "Job"}`, false},
		{`{"apiVersion": "vNotAVersion", "kind": "Job"}`, true},
	}
	components := framework.NewMasterComponents(&framework.Config{})
	defer components.Stop(true, true)
	ctx := clientcmdapi.NewContext()
	cfg := clientcmdapi.NewConfig()
	cluster := clientcmdapi.NewCluster()
	cluster.Server = components.ApiServer.URL
	cluster.InsecureSkipTLSVerify = true
	overrides := clientcmd.ConfigOverrides{
		ClusterInfo:    *cluster,
		Context:        *ctx,
		CurrentContext: "test",
	}
	cmdConfig := clientcmd.NewNonInteractiveClientConfig(*cfg, "test", &overrides)
	factory := util.NewFactory(cmdConfig)
	schema, err := factory.Validator(true, "")
	if err != nil {
		t.Errorf("failed to get validator: %v", err)
		return
	}
	for i, test := range testCases {
		err := schema.ValidateBytes([]byte(test.data))
		if err == nil {
			if test.err {
				t.Errorf("case %d: expected error", i)
			}
		} else {
			if !test.err {
				t.Errorf("case %d: unexpected error: %v", i, err)
			}
		}
	}
}
