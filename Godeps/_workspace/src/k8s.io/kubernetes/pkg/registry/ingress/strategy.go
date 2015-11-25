/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package ingress

import (
	"fmt"
	"reflect"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/apis/experimental/validation"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/fielderrors"
)

// ingressStrategy implements verification logic for Replication Ingresss.
type ingressStrategy struct {
	runtime.ObjectTyper
	api.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Replication Ingress objects.
var Strategy = ingressStrategy{api.Scheme, api.SimpleNameGenerator}

// NamespaceScoped returns true because all Ingress' need to be within a namespace.
func (ingressStrategy) NamespaceScoped() bool {
	return true
}

// PrepareForCreate clears the status of an Ingress before creation.
func (ingressStrategy) PrepareForCreate(obj runtime.Object) {
	ingress := obj.(*experimental.Ingress)
	ingress.Status = experimental.IngressStatus{}

	ingress.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (ingressStrategy) PrepareForUpdate(obj, old runtime.Object) {
	newIngress := obj.(*experimental.Ingress)
	oldIngress := old.(*experimental.Ingress)
	//TODO: Clear Ingress status once we have a sub-resource.

	// Any changes to the spec increment the generation number, any changes to the
	// status should reflect the generation number of the corresponding object.
	// See api.ObjectMeta description for more information on Generation.
	if !reflect.DeepEqual(oldIngress.Spec, newIngress.Spec) {
		newIngress.Generation = oldIngress.Generation + 1
	}

}

// Validate validates a new Ingress.
func (ingressStrategy) Validate(ctx api.Context, obj runtime.Object) fielderrors.ValidationErrorList {
	ingress := obj.(*experimental.Ingress)
	err := validation.ValidateIngress(ingress)
	return err
}

// AllowCreateOnUpdate is false for Ingress; this means POST is needed to create one.
func (ingressStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (ingressStrategy) ValidateUpdate(ctx api.Context, obj, old runtime.Object) fielderrors.ValidationErrorList {
	validationErrorList := validation.ValidateIngress(obj.(*experimental.Ingress))
	updateErrorList := validation.ValidateIngressUpdate(old.(*experimental.Ingress), obj.(*experimental.Ingress))
	return append(validationErrorList, updateErrorList...)
}

// AllowUnconditionalUpdate is the default update policy for Ingress objects.
func (ingressStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// IngressToSelectableFields returns a label set that represents the object.
func IngressToSelectableFields(ingress *experimental.Ingress) fields.Set {
	return fields.Set{
		"metadata.name": ingress.Name,
	}
}

// MatchIngress is the filter used by the generic etcd backend to ingress
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchIngress(label labels.Selector, field fields.Selector) generic.Matcher {
	return &generic.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			ingress, ok := obj.(*experimental.Ingress)
			if !ok {
				return nil, nil, fmt.Errorf("Given object is not an Ingress.")
			}
			return labels.Set(ingress.ObjectMeta.Labels), IngressToSelectableFields(ingress), nil
		},
	}
}
