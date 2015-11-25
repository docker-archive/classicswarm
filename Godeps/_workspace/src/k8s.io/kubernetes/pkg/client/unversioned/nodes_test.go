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
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

func getNodesResourceName() string {
	return "nodes"
}

func TestListNodes(t *testing.T) {
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", ""),
		},
		Response: Response{StatusCode: 200, Body: &api.NodeList{ListMeta: unversioned.ListMeta{ResourceVersion: "1"}}},
	}
	response, err := c.Setup(t).Nodes().List(labels.Everything(), fields.Everything())
	c.Validate(t, response, err)
}

func TestListNodesLabels(t *testing.T) {
	labelSelectorQueryParamName := api.LabelSelectorQueryParam(testapi.Default.Version())
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", ""),
			Query:  buildQueryValues(url.Values{labelSelectorQueryParamName: []string{"foo=bar,name=baz"}})},
		Response: Response{
			StatusCode: 200,
			Body: &api.NodeList{
				Items: []api.Node{
					{
						ObjectMeta: api.ObjectMeta{
							Labels: map[string]string{
								"foo":  "bar",
								"name": "baz",
							},
						},
					},
				},
			},
		},
	}
	c.Setup(t)
	c.QueryValidator[labelSelectorQueryParamName] = validateLabels
	selector := labels.Set{"foo": "bar", "name": "baz"}.AsSelector()
	receivedNodeList, err := c.Nodes().List(selector, fields.Everything())
	c.Validate(t, receivedNodeList, err)
}

func TestGetNode(t *testing.T) {
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", "1"),
		},
		Response: Response{StatusCode: 200, Body: &api.Node{ObjectMeta: api.ObjectMeta{Name: "node-1"}}},
	}
	response, err := c.Setup(t).Nodes().Get("1")
	c.Validate(t, response, err)
}

func TestGetNodeWithNoName(t *testing.T) {
	c := &testClient{Error: true}
	receivedNode, err := c.Setup(t).Nodes().Get("")
	if (err != nil) && (err.Error() != nameRequiredError) {
		t.Errorf("Expected error: %v, but got %v", nameRequiredError, err)
	}

	c.Validate(t, receivedNode, err)
}

func TestCreateNode(t *testing.T) {
	requestNode := &api.Node{
		ObjectMeta: api.ObjectMeta{
			Name: "node-1",
		},
		Status: api.NodeStatus{
			Capacity: api.ResourceList{
				api.ResourceCPU:    resource.MustParse("1000m"),
				api.ResourceMemory: resource.MustParse("1Mi"),
			},
		},
		Spec: api.NodeSpec{
			Unschedulable: false,
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "POST",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", ""),
			Body:   requestNode},
		Response: Response{
			StatusCode: 200,
			Body:       requestNode,
		},
	}
	receivedNode, err := c.Setup(t).Nodes().Create(requestNode)
	c.Validate(t, receivedNode, err)
}

func TestDeleteNode(t *testing.T) {
	c := &testClient{
		Request: testRequest{
			Method: "DELETE",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", "foo"),
		},
		Response: Response{StatusCode: 200},
	}
	err := c.Setup(t).Nodes().Delete("foo")
	c.Validate(t, nil, err)
}

func TestUpdateNode(t *testing.T) {
	requestNode := &api.Node{
		ObjectMeta: api.ObjectMeta{
			Name:            "foo",
			ResourceVersion: "1",
		},
		Status: api.NodeStatus{
			Capacity: api.ResourceList{
				api.ResourceCPU:    resource.MustParse("1000m"),
				api.ResourceMemory: resource.MustParse("1Mi"),
			},
		},
		Spec: api.NodeSpec{
			Unschedulable: true,
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "PUT",
			Path:   testapi.Default.ResourcePath(getNodesResourceName(), "", "foo"),
		},
		Response: Response{StatusCode: 200, Body: requestNode},
	}
	response, err := c.Setup(t).Nodes().Update(requestNode)
	c.Validate(t, response, err)
}
