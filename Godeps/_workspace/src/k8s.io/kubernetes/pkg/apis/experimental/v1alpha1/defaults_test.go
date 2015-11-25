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

package v1alpha1

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util"
)

func TestSetDefaultDaemonSet(t *testing.T) {
	tests := []struct {
		ds                 *DaemonSet
		expectLabelsChange bool
	}{
		{
			ds: &DaemonSet{
				Spec: DaemonSetSpec{
					Template: &v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabelsChange: true,
		},
		{
			ds: &DaemonSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: DaemonSetSpec{
					Template: &v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabelsChange: false,
		},
	}

	for _, test := range tests {
		ds := test.ds
		obj2 := roundTrip(t, runtime.Object(ds))
		ds2, ok := obj2.(*DaemonSet)
		if !ok {
			t.Errorf("unexpected object: %v", ds2)
			t.FailNow()
		}
		if test.expectLabelsChange != reflect.DeepEqual(ds2.Labels, ds2.Spec.Template.Labels) {
			if test.expectLabelsChange {
				t.Errorf("expected: %v, got: %v", ds2.Spec.Template.Labels, ds2.Labels)
			} else {
				t.Errorf("unexpected equality: %v", ds.Labels)
			}
		}
	}
}

func TestSetDefaultDeployment(t *testing.T) {
	defaultIntOrString := util.NewIntOrStringFromInt(1)
	differentIntOrString := util.NewIntOrStringFromInt(5)
	deploymentLabelKey := "deployment.kubernetes.io/podTemplateHash"
	tests := []struct {
		original *Deployment
		expected *Deployment
	}{
		{
			original: &Deployment{},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(1),
					Strategy: DeploymentStrategy{
						Type: RollingUpdateDeploymentStrategyType,
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge:       &defaultIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					UniqueLabelKey: newString(deploymentLabelKey),
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge: &differentIntOrString,
						},
					},
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						Type: RollingUpdateDeploymentStrategyType,
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge:       &differentIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					UniqueLabelKey: newString(deploymentLabelKey),
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					UniqueLabelKey: newString(deploymentLabelKey),
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					UniqueLabelKey: newString("customDeploymentKey"),
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					UniqueLabelKey: newString("customDeploymentKey"),
				},
			},
		},
	}

	for _, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*Deployment)
		if !ok {
			t.Errorf("unexpected object: %v", got)
			t.FailNow()
		}
		if !reflect.DeepEqual(got.Spec, expected.Spec) {
			t.Errorf("got different than expected: %v, %v", got, expected)
		}
	}
}

func TestSetDefaultJob(t *testing.T) {
	expected := &Job{
		Spec: JobSpec{
			Selector:    map[string]string{"job": "selector"},
			Completions: newInt(1),
			Parallelism: newInt(1),
		},
	}
	tests := []*Job{
		// selector set explicitly, completions and parallelism - default
		{
			Spec: JobSpec{
				Selector: map[string]string{"job": "selector"},
			},
		},
		// selector from template labels, completions and parallelism - default
		{
			Spec: JobSpec{
				Template: &v1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{"job": "selector"},
					},
				},
			},
		},
		// selector from template labels, completions set explicitly, parallelism - default
		{
			Spec: JobSpec{
				Completions: newInt(1),
				Template: &v1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{"job": "selector"},
					},
				},
			},
		},
		// selector from template labels, completions - default, parallelism set explicitly
		{
			Spec: JobSpec{
				Parallelism: newInt(1),
				Template: &v1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{"job": "selector"},
					},
				},
			},
		},
	}

	for _, original := range tests {
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*Job)
		if !ok {
			t.Errorf("unexpected object: %v", got)
			t.FailNow()
		}
		if *got.Spec.Completions != *expected.Spec.Completions {
			t.Errorf("got different completions than expected: %d %d", *got.Spec.Completions, *expected.Spec.Completions)
		}
		if *got.Spec.Parallelism != *expected.Spec.Parallelism {
			t.Errorf("got different parallelism than expected: %d %d", *got.Spec.Parallelism, *expected.Spec.Parallelism)
		}
		if !reflect.DeepEqual(got.Spec.Selector, expected.Spec.Selector) {
			t.Errorf("got different selectors %#v %#v", got.Spec.Selector, expected.Spec.Selector)
		}
	}
}

func roundTrip(t *testing.T, obj runtime.Object) runtime.Object {
	data, err := Codec.Encode(obj)
	if err != nil {
		t.Errorf("%v\n %#v", err, obj)
		return nil
	}
	obj2, err := api.Codec.Decode(data)
	if err != nil {
		t.Errorf("%v\nData: %s\nSource: %#v", err, string(data), obj)
		return nil
	}
	obj3 := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(runtime.Object)
	err = api.Scheme.Convert(obj2, obj3)
	if err != nil {
		t.Errorf("%v\nSource: %#v", err, obj2)
		return nil
	}
	return obj3
}

func newInt(val int) *int {
	p := new(int)
	*p = val
	return p
}

func newString(val string) *string {
	p := new(string)
	*p = val
	return p
}
