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
	"net/url"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

func getPodTemplatesResoureName() string {
	return "podtemplates"
}

func TestPodTemplateCreate(t *testing.T) {
	ns := api.NamespaceDefault
	podTemplate := api.PodTemplate{
		ObjectMeta: api.ObjectMeta{
			Name:      "abc",
			Namespace: ns,
		},
		Template: api.PodTemplateSpec{},
	}
	c := &testClient{
		Request: testRequest{
			Method: "POST",
			Path:   testapi.Default.ResourcePath(getPodTemplatesResoureName(), ns, ""),
			Query:  buildQueryValues(nil),
			Body:   &podTemplate,
		},
		Response: Response{StatusCode: 200, Body: &podTemplate},
	}

	response, err := c.Setup(t).PodTemplates(ns).Create(&podTemplate)
	c.Validate(t, response, err)
}

func TestPodTemplateGet(t *testing.T) {
	ns := api.NamespaceDefault
	podTemplate := &api.PodTemplate{
		ObjectMeta: api.ObjectMeta{
			Name:      "abc",
			Namespace: ns,
		},
		Template: api.PodTemplateSpec{},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePath(getPodTemplatesResoureName(), ns, "abc"),
			Query:  buildQueryValues(nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: podTemplate},
	}

	response, err := c.Setup(t).PodTemplates(ns).Get("abc")
	c.Validate(t, response, err)
}

func TestPodTemplateList(t *testing.T) {
	ns := api.NamespaceDefault
	podTemplateList := &api.PodTemplateList{
		Items: []api.PodTemplate{
			{
				ObjectMeta: api.ObjectMeta{
					Name:      "foo",
					Namespace: ns,
				},
			},
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePath(getPodTemplatesResoureName(), ns, ""),
			Query:  buildQueryValues(nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: podTemplateList},
	}
	response, err := c.Setup(t).PodTemplates(ns).List(labels.Everything(), fields.Everything())
	c.Validate(t, response, err)
}

func TestPodTemplateUpdate(t *testing.T) {
	ns := api.NamespaceDefault
	podTemplate := &api.PodTemplate{
		ObjectMeta: api.ObjectMeta{
			Name:            "abc",
			Namespace:       ns,
			ResourceVersion: "1",
		},
		Template: api.PodTemplateSpec{},
	}
	c := &testClient{
		Request:  testRequest{Method: "PUT", Path: testapi.Default.ResourcePath(getPodTemplatesResoureName(), ns, "abc"), Query: buildQueryValues(nil)},
		Response: Response{StatusCode: 200, Body: podTemplate},
	}
	response, err := c.Setup(t).PodTemplates(ns).Update(podTemplate)
	c.Validate(t, response, err)
}

func TestPodTemplateDelete(t *testing.T) {
	ns := api.NamespaceDefault
	c := &testClient{
		Request:  testRequest{Method: "DELETE", Path: testapi.Default.ResourcePath(getPodTemplatesResoureName(), ns, "foo"), Query: buildQueryValues(nil)},
		Response: Response{StatusCode: 200},
	}
	err := c.Setup(t).PodTemplates(ns).Delete("foo", nil)
	c.Validate(t, nil, err)
}

func TestPodTemplateWatch(t *testing.T) {
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePathWithPrefix("watch", getPodTemplatesResoureName(), "", ""),
			Query:  url.Values{"resourceVersion": []string{}}},
		Response: Response{StatusCode: 200},
	}
	_, err := c.Setup(t).PodTemplates(api.NamespaceAll).Watch(labels.Everything(), fields.Everything(), "")
	c.Validate(t, nil, err)
}
