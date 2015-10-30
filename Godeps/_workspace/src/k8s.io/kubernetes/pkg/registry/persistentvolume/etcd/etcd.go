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
	"path"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	etcdgeneric "k8s.io/kubernetes/pkg/registry/generic/etcd"
	"k8s.io/kubernetes/pkg/registry/persistentvolume"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
)

type REST struct {
	*etcdgeneric.Etcd
}

// NewREST returns a RESTStorage object that will work against persistent volumes.
func NewREST(s storage.Interface) (*REST, *StatusREST) {
	prefix := "/persistentvolumes"
	store := &etcdgeneric.Etcd{
		NewFunc:     func() runtime.Object { return &api.PersistentVolume{} },
		NewListFunc: func() runtime.Object { return &api.PersistentVolumeList{} },
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return path.Join(prefix, name), nil
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*api.PersistentVolume).Name, nil
		},
		PredicateFunc: func(label labels.Selector, field fields.Selector) generic.Matcher {
			return persistentvolume.MatchPersistentVolumes(label, field)
		},
		EndpointName: "persistentvolume",

		CreateStrategy:      persistentvolume.Strategy,
		UpdateStrategy:      persistentvolume.Strategy,
		ReturnDeletedObject: true,

		Storage: s,
	}

	statusStore := *store
	statusStore.UpdateStrategy = persistentvolume.StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}
}

// StatusREST implements the REST endpoint for changing the status of a persistentvolume.
type StatusREST struct {
	store *etcdgeneric.Etcd
}

func (r *StatusREST) New() runtime.Object {
	return &api.PersistentVolume{}
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx api.Context, obj runtime.Object) (runtime.Object, bool, error) {
	return r.store.Update(ctx, obj)
}
