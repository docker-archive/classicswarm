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

package config

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/registered"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/securitycontext"

	"github.com/ghodss/yaml"
)

func noDefault(*api.Pod) error { return nil }

func TestDecodeSinglePod(t *testing.T) {
	grace := int64(30)
	pod := &api.Pod{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      "test",
			UID:       "12345",
			Namespace: "mynamespace",
		},
		Spec: api.PodSpec{
			RestartPolicy:                 api.RestartPolicyAlways,
			DNSPolicy:                     api.DNSClusterFirst,
			TerminationGracePeriodSeconds: &grace,
			Containers: []api.Container{{
				Name:                   "image",
				Image:                  "test/image",
				ImagePullPolicy:        "IfNotPresent",
				TerminationMessagePath: "/dev/termination-log",
				SecurityContext:        securitycontext.ValidSecurityContextWithContainerDefaults(),
			}},
		},
	}
	json, err := testapi.Default.Codec().Encode(pod)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	parsed, podOut, err := tryDecodeSinglePod(json, noDefault)
	if !parsed {
		t.Errorf("expected to have parsed file: (%s)", string(json))
	}
	if err != nil {
		t.Errorf("unexpected error: %v (%s)", err, string(json))
	}
	if !reflect.DeepEqual(pod, podOut) {
		t.Errorf("expected:\n%#v\ngot:\n%#v\n%s", pod, podOut, string(json))
	}

	for _, version := range registered.GroupVersionsForGroup("") {
		externalPod, err := testapi.Default.Converter().ConvertToVersion(pod, version)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		yaml, err := yaml.Marshal(externalPod)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		parsed, podOut, err = tryDecodeSinglePod(yaml, noDefault)
		if !parsed {
			t.Errorf("expected to have parsed file: (%s)", string(yaml))
		}
		if err != nil {
			t.Errorf("unexpected error: %v (%s)", err, string(yaml))
		}
		if !reflect.DeepEqual(pod, podOut) {
			t.Errorf("expected:\n%#v\ngot:\n%#v\n%s", pod, podOut, string(yaml))
		}
	}
}

func TestDecodePodList(t *testing.T) {
	grace := int64(30)
	pod := &api.Pod{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      "test",
			UID:       "12345",
			Namespace: "mynamespace",
		},
		Spec: api.PodSpec{
			RestartPolicy:                 api.RestartPolicyAlways,
			DNSPolicy:                     api.DNSClusterFirst,
			TerminationGracePeriodSeconds: &grace,
			Containers: []api.Container{{
				Name:                   "image",
				Image:                  "test/image",
				ImagePullPolicy:        "IfNotPresent",
				TerminationMessagePath: "/dev/termination-log",
				SecurityContext:        securitycontext.ValidSecurityContextWithContainerDefaults(),
			}},
		},
	}
	podList := &api.PodList{
		Items: []api.Pod{*pod},
	}
	json, err := testapi.Default.Codec().Encode(podList)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	parsed, podListOut, err := tryDecodePodList(json, noDefault)
	if !parsed {
		t.Errorf("expected to have parsed file: (%s)", string(json))
	}
	if err != nil {
		t.Errorf("unexpected error: %v (%s)", err, string(json))
	}
	if !reflect.DeepEqual(podList, &podListOut) {
		t.Errorf("expected:\n%#v\ngot:\n%#v\n%s", podList, &podListOut, string(json))
	}

	for _, version := range registered.GroupVersionsForGroup("") {
		externalPodList, err := testapi.Default.Converter().ConvertToVersion(podList, version)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		yaml, err := yaml.Marshal(externalPodList)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		parsed, podListOut, err = tryDecodePodList(yaml, noDefault)
		if !parsed {
			t.Errorf("expected to have parsed file: (%s)", string(yaml))
		}
		if err != nil {
			t.Errorf("unexpected error: %v (%s)", err, string(yaml))
		}
		if !reflect.DeepEqual(podList, &podListOut) {
			t.Errorf("expected:\n%#v\ngot:\n%#v\n%s", pod, &podListOut, string(yaml))
		}
	}
}
