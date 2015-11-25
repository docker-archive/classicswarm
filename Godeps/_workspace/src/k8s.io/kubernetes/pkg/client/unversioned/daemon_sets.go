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

package unversioned

import (
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

// DaemonsSetsNamespacer has methods to work with DaemonSet resources in a namespace
type DaemonSetsNamespacer interface {
	DaemonSets(namespace string) DaemonSetInterface
}

type DaemonSetInterface interface {
	List(selector labels.Selector) (*experimental.DaemonSetList, error)
	Get(name string) (*experimental.DaemonSet, error)
	Create(ctrl *experimental.DaemonSet) (*experimental.DaemonSet, error)
	Update(ctrl *experimental.DaemonSet) (*experimental.DaemonSet, error)
	UpdateStatus(ctrl *experimental.DaemonSet) (*experimental.DaemonSet, error)
	Delete(name string) error
	Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error)
}

// daemonSets implements DaemonsSetsNamespacer interface
type daemonSets struct {
	r  *ExperimentalClient
	ns string
}

func newDaemonSets(c *ExperimentalClient, namespace string) *daemonSets {
	return &daemonSets{c, namespace}
}

// Ensure statically that daemonSets implements DaemonSetsInterface.
var _ DaemonSetInterface = &daemonSets{}

func (c *daemonSets) List(selector labels.Selector) (result *experimental.DaemonSetList, err error) {
	result = &experimental.DaemonSetList{}
	err = c.r.Get().Namespace(c.ns).Resource("daemonsets").LabelsSelectorParam(selector).Do().Into(result)
	return
}

// Get returns information about a particular daemon set.
func (c *daemonSets) Get(name string) (result *experimental.DaemonSet, err error) {
	result = &experimental.DaemonSet{}
	err = c.r.Get().Namespace(c.ns).Resource("daemonsets").Name(name).Do().Into(result)
	return
}

// Create creates a new daemon set.
func (c *daemonSets) Create(daemon *experimental.DaemonSet) (result *experimental.DaemonSet, err error) {
	result = &experimental.DaemonSet{}
	err = c.r.Post().Namespace(c.ns).Resource("daemonsets").Body(daemon).Do().Into(result)
	return
}

// Update updates an existing daemon set.
func (c *daemonSets) Update(daemon *experimental.DaemonSet) (result *experimental.DaemonSet, err error) {
	result = &experimental.DaemonSet{}
	err = c.r.Put().Namespace(c.ns).Resource("daemonsets").Name(daemon.Name).Body(daemon).Do().Into(result)
	return
}

// UpdateStatus updates an existing daemon set status
func (c *daemonSets) UpdateStatus(daemon *experimental.DaemonSet) (result *experimental.DaemonSet, err error) {
	result = &experimental.DaemonSet{}
	err = c.r.Put().Namespace(c.ns).Resource("daemonsets").Name(daemon.Name).SubResource("status").Body(daemon).Do().Into(result)
	return
}

// Delete deletes an existing daemon set.
func (c *daemonSets) Delete(name string) error {
	return c.r.Delete().Namespace(c.ns).Resource("daemonsets").Name(name).Do().Error()
}

// Watch returns a watch.Interface that watches the requested daemon sets.
func (c *daemonSets) Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("daemonsets").
		Param("resourceVersion", resourceVersion).
		LabelsSelectorParam(label).
		FieldsSelectorParam(field).
		Watch()
}
