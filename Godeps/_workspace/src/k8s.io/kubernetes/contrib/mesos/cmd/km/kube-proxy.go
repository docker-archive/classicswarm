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

// clone of the upstream cmd/hypercube/kube-proxy.go
package main

import (
	kubeproxy "k8s.io/kubernetes/cmd/kube-proxy/app"
	"k8s.io/kubernetes/contrib/mesos/pkg/hyperkube"
)

// NewKubeProxy creates a new hyperkube Server object that includes the
// description and flags.

func NewKubeProxy() *Server {
	config := kubeproxy.NewProxyConfig()

	hks := Server{
		SimpleUsage: hyperkube.CommandProxy,
		Long: `The Kubernetes proxy server is responsible for taking traffic directed at
		services and forwarding it to the appropriate pods. It generally runs on
		nodes next to the Kubelet and proxies traffic from local pods to remote pods.
		It is also used when handling incoming external traffic.`,
	}

	config.AddFlags(hks.Flags())

	hks.Run = func(_ *Server, args []string) error {
		s, err := kubeproxy.NewProxyServerDefault(config)
		if err != nil {
			return err
		}

		return s.Run(args)
	}

	return &hks
}
