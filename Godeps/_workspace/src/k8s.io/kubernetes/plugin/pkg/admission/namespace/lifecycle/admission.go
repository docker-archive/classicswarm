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

package lifecycle

import (
	"fmt"
	"io"
	"time"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/kubernetes/pkg/watch"
)

func init() {
	admission.RegisterPlugin("NamespaceLifecycle", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		return NewLifecycle(client), nil
	})
}

// lifecycle is an implementation of admission.Interface.
// It enforces life-cycle constraints around a Namespace depending on its Phase
type lifecycle struct {
	*admission.Handler
	client             client.Interface
	store              cache.Store
	immortalNamespaces sets.String
}

func (l *lifecycle) Admit(a admission.Attributes) (err error) {

	// prevent deletion of immortal namespaces
	if a.GetOperation() == admission.Delete && a.GetKind() == "Namespace" && l.immortalNamespaces.Has(a.GetName()) {
		return errors.NewForbidden(a.GetKind(), a.GetName(), fmt.Errorf("namespace can never be deleted"))
	}

	defaultVersion, kind, err := api.RESTMapper.VersionAndKindForResource(a.GetResource())
	if err != nil {
		return errors.NewInternalError(err)
	}
	mapping, err := api.RESTMapper.RESTMapping(kind, defaultVersion)
	if err != nil {
		return errors.NewInternalError(err)
	}
	if mapping.Scope.Name() != meta.RESTScopeNameNamespace {
		return nil
	}
	namespaceObj, exists, err := l.store.Get(&api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name:      a.GetNamespace(),
			Namespace: "",
		},
	})
	if err != nil {
		return errors.NewInternalError(err)
	}

	// refuse to operate on non-existent namespaces
	if !exists {
		// in case of latency in our caches, make a call direct to storage to verify that it truly exists or not
		namespaceObj, err = l.client.Namespaces().Get(a.GetNamespace())
		if err != nil {
			if errors.IsNotFound(err) {
				return err
			}
			return errors.NewInternalError(err)
		}
	}

	// ensure that we're not trying to create objects in terminating namespaces
	if a.GetOperation() == admission.Create {
		namespace := namespaceObj.(*api.Namespace)
		if namespace.Status.Phase != api.NamespaceTerminating {
			return nil
		}

		// TODO: This should probably not be a 403
		return admission.NewForbidden(a, fmt.Errorf("Unable to create new content in namespace %s because it is being terminated.", a.GetNamespace()))
	}

	return nil
}

// NewLifecycle creates a new namespace lifecycle admission control handler
func NewLifecycle(c client.Interface) admission.Interface {
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	reflector := cache.NewReflector(
		&cache.ListWatch{
			ListFunc: func() (runtime.Object, error) {
				return c.Namespaces().List(labels.Everything(), fields.Everything())
			},
			WatchFunc: func(resourceVersion string) (watch.Interface, error) {
				return c.Namespaces().Watch(labels.Everything(), fields.Everything(), resourceVersion)
			},
		},
		&api.Namespace{},
		store,
		5*time.Minute,
	)
	reflector.Run()
	return &lifecycle{
		Handler:            admission.NewHandler(admission.Create, admission.Update, admission.Delete),
		client:             c,
		store:              store,
		immortalNamespaces: sets.NewString(api.NamespaceDefault),
	}
}
