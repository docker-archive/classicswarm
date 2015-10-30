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

package testclient

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/yaml"
	"k8s.io/kubernetes/pkg/watch"
)

// ObjectRetriever abstracts the implementation for retrieving or setting generic
// objects. It is intended to be used to fake calls to a server by returning
// objects based on their kind and name.
type ObjectRetriever interface {
	// Kind should return a resource or a list of resources (depending on the provided kind and
	// name). It should return an error if the caller should communicate an error to the server.
	Kind(kind, name string) (runtime.Object, error)
	// Add adds a runtime object for test purposes into this object.
	Add(runtime.Object) error
}

// ObjectReaction returns a ReactionFunc that takes a generic action string of the form
// <verb>-<resource> or <verb>-<subresource>-<resource> and attempts to return a runtime
// Object or error that matches the requested action. For instance, list-replicationControllers
// should attempt to return a list of replication controllers. This method delegates to the
// ObjectRetriever interface to satisfy retrieval of lists or retrieval of single items.
// TODO: add support for sub resources
func ObjectReaction(o ObjectRetriever, mapper meta.RESTMapper) ReactionFunc {

	return func(action Action) (bool, runtime.Object, error) {
		_, kind, err := mapper.VersionAndKindForResource(action.GetResource())
		if err != nil {
			return false, nil, fmt.Errorf("unrecognized action %s: %v", action.GetResource(), err)
		}

		// TODO: have mapper return a Kind for a subresource?
		switch castAction := action.(type) {
		case ListAction:
			resource, err := o.Kind(kind+"List", "")
			return true, resource, err

		case GetAction:
			resource, err := o.Kind(kind, castAction.GetName())
			return true, resource, err

		case DeleteAction:
			resource, err := o.Kind(kind, castAction.GetName())
			return true, resource, err

		case CreateAction:
			meta, err := api.ObjectMetaFor(castAction.GetObject())
			if err != nil {
				return true, nil, err
			}
			resource, err := o.Kind(kind, meta.Name)
			return true, resource, err

		case UpdateAction:
			meta, err := api.ObjectMetaFor(castAction.GetObject())
			if err != nil {
				return true, nil, err
			}
			resource, err := o.Kind(kind, meta.Name)
			return true, resource, err

		default:
			return false, nil, fmt.Errorf("no reaction implemented for %s", action)
		}

		return true, nil, nil
	}
}

// AddObjectsFromPath loads the JSON or YAML file containing Kubernetes API resources
// and adds them to the provided ObjectRetriever.
func AddObjectsFromPath(path string, o ObjectRetriever, decoder runtime.Decoder) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	data, err = yaml.ToJSON(data)
	if err != nil {
		return err
	}
	obj, err := decoder.Decode(data)
	if err != nil {
		return err
	}
	if err := o.Add(obj); err != nil {
		return err
	}
	return nil
}

type objects struct {
	types   map[string][]runtime.Object
	last    map[string]int
	scheme  runtime.ObjectScheme
	decoder runtime.ObjectDecoder
}

var _ ObjectRetriever = &objects{}

// NewObjects implements the ObjectRetriever interface by introspecting the
// objects provided to Add() and returning them when the Kind method is invoked.
// If an api.List object is provided to Add(), each child item is added. If an
// object is added that is itself a list (PodList, ServiceList) then that is added
// to the "PodList" kind. If no PodList is added, the retriever will take any loaded
// Pods and return them in a list. If an api.Status is added, and the Details.Kind field
// is set, that status will be returned instead (as an error if Status != Success, or
// as a runtime.Object if Status == Success).  If multiple PodLists are provided, they
// will be returned in order by the Kind call, and the last PodList will be reused for
// subsequent calls.
func NewObjects(scheme runtime.ObjectScheme, decoder runtime.ObjectDecoder) ObjectRetriever {
	return objects{
		types:   make(map[string][]runtime.Object),
		last:    make(map[string]int),
		scheme:  scheme,
		decoder: decoder,
	}
}

func (o objects) Kind(kind, name string) (runtime.Object, error) {
	empty, _ := o.scheme.New("", kind)
	nilValue := reflect.Zero(reflect.TypeOf(empty)).Interface().(runtime.Object)

	arr, ok := o.types[kind]
	if !ok {
		if strings.HasSuffix(kind, "List") {
			itemKind := kind[:len(kind)-4]
			arr, ok := o.types[itemKind]
			if !ok {
				return empty, nil
			}
			out, err := o.scheme.New("", kind)
			if err != nil {
				return nilValue, err
			}
			if err := runtime.SetList(out, arr); err != nil {
				return nilValue, err
			}
			if out, err = o.scheme.Copy(out); err != nil {
				return nilValue, err
			}
			return out, nil
		}
		return nilValue, errors.NewNotFound(kind, name)
	}

	index := o.last[kind]
	if index >= len(arr) {
		index = len(arr) - 1
	}
	if index < 0 {
		return nilValue, errors.NewNotFound(kind, name)
	}
	out, err := o.scheme.Copy(arr[index])
	if err != nil {
		return nilValue, err
	}
	o.last[kind] = index + 1

	if status, ok := out.(*unversioned.Status); ok {
		if status.Details != nil {
			status.Details.Kind = kind
		}
		if status.Status != unversioned.StatusSuccess {
			return nilValue, &errors.StatusError{ErrStatus: *status}
		}
	}

	return out, nil
}

func (o objects) Add(obj runtime.Object) error {
	_, kind, err := o.scheme.ObjectVersionAndKind(obj)
	if err != nil {
		return err
	}

	switch {
	case runtime.IsListType(obj):
		if kind != "List" {
			o.types[kind] = append(o.types[kind], obj)
		}

		list, err := runtime.ExtractList(obj)
		if err != nil {
			return err
		}
		if errs := runtime.DecodeList(list, o.decoder); len(errs) > 0 {
			return errs[0]
		}
		for _, obj := range list {
			if err := o.Add(obj); err != nil {
				return err
			}
		}
	default:
		if status, ok := obj.(*unversioned.Status); ok && status.Details != nil {
			kind = status.Details.Kind
		}
		o.types[kind] = append(o.types[kind], obj)
	}

	return nil
}

func DefaultWatchReactor(watchInterface watch.Interface, err error) WatchReactionFunc {
	return func(action Action) (bool, watch.Interface, error) {
		return true, watchInterface, err
	}
}

// SimpleReactor is a Reactor.  Each reaction function is attached to a given verb,resource tuple.  "*" in either field matches everything for that value.
// For instance, *,pods matches all verbs on pods.  This allows for easier composition of reaction functions
type SimpleReactor struct {
	Verb     string
	Resource string

	Reaction ReactionFunc
}

func (r *SimpleReactor) Handles(action Action) bool {
	verbCovers := r.Verb == "*" || r.Verb == action.GetVerb()
	if !verbCovers {
		return false
	}
	resourceCovers := r.Resource == "*" || r.Resource == action.GetResource()
	if !resourceCovers {
		return false
	}

	return true
}

func (r *SimpleReactor) React(action Action) (bool, runtime.Object, error) {
	return r.Reaction(action)
}

// SimpleWatchReactor is a WatchReactor.  Each reaction function is attached to a given resource.  "*" matches everything for that value.
// For instance, *,pods matches all verbs on pods.  This allows for easier composition of reaction functions
type SimpleWatchReactor struct {
	Resource string

	Reaction WatchReactionFunc
}

func (r *SimpleWatchReactor) Handles(action Action) bool {
	resourceCovers := r.Resource == "*" || r.Resource == action.GetResource()
	if !resourceCovers {
		return false
	}

	return true
}

func (r *SimpleWatchReactor) React(action Action) (bool, watch.Interface, error) {
	return r.Reaction(action)
}

// SimpleProxyReactor is a ProxyReactor.  Each reaction function is attached to a given resource.  "*" matches everything for that value.
// For instance, *,pods matches all verbs on pods.  This allows for easier composition of reaction functions.
type SimpleProxyReactor struct {
	Resource string

	Reaction ProxyReactionFunc
}

func (r *SimpleProxyReactor) Handles(action Action) bool {
	resourceCovers := r.Resource == "*" || r.Resource == action.GetResource()
	if !resourceCovers {
		return false
	}

	return true
}

func (r *SimpleProxyReactor) React(action Action) (bool, client.ResponseWrapper, error) {
	return r.Reaction(action)
}
