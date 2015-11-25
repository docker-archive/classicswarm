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

// This package main implements the executable Kubernetes Mesos controller manager.
//
// It is mainly a clone of the upstream cmd/hyperkube module right now because
// the upstream hyperkube module is not reusable.
//
// TODO(jdef,sttts): refactor upstream cmd/kube-controller-manager to be reusable with the necessary mesos changes
package main
