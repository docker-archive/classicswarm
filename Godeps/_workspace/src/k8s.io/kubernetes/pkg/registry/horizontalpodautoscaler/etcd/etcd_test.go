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

package etcd

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/apis/experimental"
	// Ensure that experimental/v1alpha1 package is initialized.
	_ "k8s.io/kubernetes/pkg/apis/experimental/v1alpha1"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/registrytest"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/tools"
)

func newStorage(t *testing.T) (*REST, *tools.FakeEtcdClient) {
	etcdStorage, fakeClient := registrytest.NewEtcdStorage(t, "experimental")
	return NewREST(etcdStorage), fakeClient
}

func validNewHorizontalPodAutoscaler(name string) *experimental.HorizontalPodAutoscaler {
	return &experimental.HorizontalPodAutoscaler{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: api.NamespaceDefault,
		},
		Spec: experimental.HorizontalPodAutoscalerSpec{
			ScaleRef: &experimental.SubresourceReference{
				Subresource: "scale",
			},
			MinReplicas: 1,
			MaxReplicas: 5,
			Target:      experimental.ResourceConsumption{Resource: api.ResourceCPU, Quantity: resource.MustParse("0.8")},
		},
	}
}

func TestCreate(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	autoscaler := validNewHorizontalPodAutoscaler("foo")
	autoscaler.ObjectMeta = api.ObjectMeta{}
	test.TestCreate(
		// valid
		autoscaler,
		// invalid
		&experimental.HorizontalPodAutoscaler{},
	)
}

func TestUpdate(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestUpdate(
		// valid
		validNewHorizontalPodAutoscaler("foo"),
		// updateFunc
		func(obj runtime.Object) runtime.Object {
			object := obj.(*experimental.HorizontalPodAutoscaler)
			object.Spec.MaxReplicas = object.Spec.MaxReplicas + 1
			return object
		},
	)
}

func TestDelete(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestDelete(validNewHorizontalPodAutoscaler("foo"))
}

func TestGet(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestGet(validNewHorizontalPodAutoscaler("foo"))
}

func TestList(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestList(validNewHorizontalPodAutoscaler("foo"))
}

func TestWatch(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestWatch(
		validNewHorizontalPodAutoscaler("foo"),
		// matching labels
		[]labels.Set{},
		// not matching labels
		[]labels.Set{
			{"foo": "bar"},
		},
		// matching fields
		[]fields.Set{},
		// not matching fields
		[]fields.Set{
			{"metadata.name": "bar"},
			{"name": "foo"},
		},
	)
}
