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

package podtask

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	mutil "github.com/mesos/mesos-go/mesosutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/contrib/mesos/pkg/node"
	mresource "k8s.io/kubernetes/contrib/mesos/pkg/scheduler/resource"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

const (
	t_min_cpu = 128
	t_min_mem = 128
)

func fakePodTask(id string) (*T, error) {
	return New(api.NewDefaultContext(), "", api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:      id,
			Namespace: api.NamespaceDefault,
		},
	}, &mesos.ExecutorInfo{})
}

func TestUnlimitedResources(t *testing.T) {
	assert := assert.New(t)

	task, _ := fakePodTask("unlimited")
	pod := &task.Pod
	pod.Spec = api.PodSpec{
		Containers: []api.Container{{
			Name: "a",
			Ports: []api.ContainerPort{{
				HostPort: 123,
			}},
			Resources: api.ResourceRequirements{
				Limits: api.ResourceList{
					api.ResourceCPU:    *resource.NewQuantity(3, resource.DecimalSI),
					api.ResourceMemory: *resource.NewQuantity(768*1024*1024, resource.BinarySI),
				},
			},
		}, {
			Name: "b",
		}, {
			Name: "c",
		}},
	}

	beforeLimitingCPU := mresource.CPUForPod(pod, mresource.DefaultDefaultContainerCPULimit)
	beforeLimitingMem := mresource.MemForPod(pod, mresource.DefaultDefaultContainerMemLimit)

	unboundedCPU := mresource.LimitPodCPU(pod, mresource.DefaultDefaultContainerCPULimit)
	unboundedMem := mresource.LimitPodMem(pod, mresource.DefaultDefaultContainerMemLimit)

	cpu := mresource.PodCPULimit(pod)
	mem := mresource.PodMemLimit(pod)

	assert.True(unboundedCPU, "CPU resources are defined as unlimited")
	assert.True(unboundedMem, "mem resources are defined as unlimited")

	assert.Equal(2*float64(mresource.DefaultDefaultContainerCPULimit)+3.0, float64(cpu))
	assert.Equal(2*float64(mresource.DefaultDefaultContainerMemLimit)+768.0, float64(mem))

	assert.Equal(cpu, beforeLimitingCPU)
	assert.Equal(mem, beforeLimitingMem)
}

func TestLimitedResources(t *testing.T) {
	assert := assert.New(t)

	task, _ := fakePodTask("limited")
	pod := &task.Pod
	pod.Spec = api.PodSpec{
		Containers: []api.Container{{
			Name: "a",
			Resources: api.ResourceRequirements{
				Limits: api.ResourceList{
					api.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
					api.ResourceMemory: *resource.NewQuantity(256*1024*1024, resource.BinarySI),
				},
			},
		}, {
			Name: "b",
			Resources: api.ResourceRequirements{
				Limits: api.ResourceList{
					api.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
					api.ResourceMemory: *resource.NewQuantity(512*1024*1024, resource.BinarySI),
				},
			},
		}},
	}

	beforeLimitingCPU := mresource.CPUForPod(pod, mresource.DefaultDefaultContainerCPULimit)
	beforeLimitingMem := mresource.MemForPod(pod, mresource.DefaultDefaultContainerMemLimit)

	unboundedCPU := mresource.LimitPodCPU(pod, mresource.DefaultDefaultContainerCPULimit)
	unboundedMem := mresource.LimitPodMem(pod, mresource.DefaultDefaultContainerMemLimit)

	cpu := mresource.PodCPULimit(pod)
	mem := mresource.PodMemLimit(pod)

	assert.False(unboundedCPU, "CPU resources are defined as limited")
	assert.False(unboundedMem, "mem resources are defined as limited")

	assert.Equal(3.0, float64(cpu))
	assert.Equal(768.0, float64(mem))

	assert.Equal(cpu, beforeLimitingCPU)
	assert.Equal(mem, beforeLimitingMem)
}

func TestEmptyOffer(t *testing.T) {
	t.Parallel()
	task, err := fakePodTask("foo")
	if err != nil {
		t.Fatal(err)
	}

	task.Pod.Spec = api.PodSpec{
		Containers: []api.Container{{
			Name: "a",
		}},
	}

	mresource.LimitPodCPU(&task.Pod, mresource.DefaultDefaultContainerCPULimit)
	mresource.LimitPodMem(&task.Pod, mresource.DefaultDefaultContainerMemLimit)

	if ok := DefaultPredicate(task, nil, nil); ok {
		t.Fatalf("accepted nil offer")
	}
	if ok := DefaultPredicate(task, &mesos.Offer{}, nil); ok {
		t.Fatalf("accepted empty offer")
	}
}

func TestNoPortsInPodOrOffer(t *testing.T) {
	t.Parallel()
	task, err := fakePodTask("foo")
	if err != nil || task == nil {
		t.Fatal(err)
	}

	task.Pod.Spec = api.PodSpec{
		Containers: []api.Container{{
			Name: "a",
		}},
	}

	mresource.LimitPodCPU(&task.Pod, mresource.DefaultDefaultContainerCPULimit)
	mresource.LimitPodMem(&task.Pod, mresource.DefaultDefaultContainerMemLimit)

	offer := &mesos.Offer{
		Resources: []*mesos.Resource{
			mutil.NewScalarResource("cpus", 0.001),
			mutil.NewScalarResource("mem", 0.001),
		},
	}
	if ok := DefaultPredicate(task, offer, nil); ok {
		t.Fatalf("accepted offer %v:", offer)
	}

	offer = &mesos.Offer{
		Resources: []*mesos.Resource{
			mutil.NewScalarResource("cpus", t_min_cpu),
			mutil.NewScalarResource("mem", t_min_mem),
		},
	}
	if ok := DefaultPredicate(task, offer, nil); !ok {
		t.Fatalf("did not accepted offer %v:", offer)
	}
}

func TestAcceptOfferPorts(t *testing.T) {
	t.Parallel()
	task, _ := fakePodTask("foo")
	pod := &task.Pod

	offer := &mesos.Offer{
		Resources: []*mesos.Resource{
			mutil.NewScalarResource("cpus", t_min_cpu),
			mutil.NewScalarResource("mem", t_min_mem),
			rangeResource("ports", []uint64{1, 1}),
		},
	}
	if ok := DefaultPredicate(task, offer, nil); !ok {
		t.Fatalf("did not accepted offer %v:", offer)
	}

	pod.Spec = api.PodSpec{
		Containers: []api.Container{{
			Ports: []api.ContainerPort{{
				HostPort: 123,
			}},
		}},
	}

	mresource.LimitPodCPU(&task.Pod, mresource.DefaultDefaultContainerCPULimit)
	mresource.LimitPodMem(&task.Pod, mresource.DefaultDefaultContainerMemLimit)

	if ok := DefaultPredicate(task, offer, nil); ok {
		t.Fatalf("accepted offer %v:", offer)
	}

	pod.Spec.Containers[0].Ports[0].HostPort = 1
	if ok := DefaultPredicate(task, offer, nil); !ok {
		t.Fatalf("did not accepted offer %v:", offer)
	}

	pod.Spec.Containers[0].Ports[0].HostPort = 0
	if ok := DefaultPredicate(task, offer, nil); !ok {
		t.Fatalf("did not accepted offer %v:", offer)
	}

	offer.Resources = []*mesos.Resource{
		mutil.NewScalarResource("cpus", t_min_cpu),
		mutil.NewScalarResource("mem", t_min_mem),
	}
	if ok := DefaultPredicate(task, offer, nil); ok {
		t.Fatalf("accepted offer %v:", offer)
	}

	pod.Spec.Containers[0].Ports[0].HostPort = 1
	if ok := DefaultPredicate(task, offer, nil); ok {
		t.Fatalf("accepted offer %v:", offer)
	}
}

func TestGeneratePodName(t *testing.T) {
	p := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	name := generateTaskName(p)
	expected := "foo.bar.pods"
	if name != expected {
		t.Fatalf("expected %q instead of %q", expected, name)
	}

	p.Namespace = ""
	name = generateTaskName(p)
	expected = "foo.default.pods"
	if name != expected {
		t.Fatalf("expected %q instead of %q", expected, name)
	}
}

func TestNodeSelector(t *testing.T) {
	t.Parallel()

	newNode := func(hostName string, l map[string]string) *api.Node {
		nodeLabels := map[string]string{"kubernetes.io/hostname": hostName}
		if l != nil {
			for k, v := range l {
				nodeLabels[k] = v
			}
		}
		return &api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   hostName,
				Labels: nodeLabels,
			},
			Spec: api.NodeSpec{
				ExternalID: hostName,
			},
		}
	}
	node1 := newNode("node1", node.SlaveAttributesToLabels([]*mesos.Attribute{
		newTextAttribute("rack", "a"),
		newTextAttribute("gen", "2014"),
		newScalarAttribute("num", 42.0),
	}))
	node2 := newNode("node2", node.SlaveAttributesToLabels([]*mesos.Attribute{
		newTextAttribute("rack", "b"),
		newTextAttribute("gen", "2015"),
		newScalarAttribute("num", 0.0),
	}))
	labels3 := node.SlaveAttributesToLabels([]*mesos.Attribute{
		newTextAttribute("rack", "c"),
		newTextAttribute("gen", "2015"),
		newScalarAttribute("old", 42),
	})
	labels3["some.other/label"] = "43"
	node3 := newNode("node3", labels3)

	tests := []struct {
		selector map[string]string
		node     *api.Node
		ok       bool
		desc     string
	}{
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "a"}, node1, true, "label value matches"},
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "b"}, node1, false, "label value does not match"},
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "a", "k8s.mesosphere.io/attribute-gen": "2014"}, node1, true, "multiple required labels match"},
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "a", "k8s.mesosphere.io/attribute-gen": "2015"}, node1, false, "one label does not match"},
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "a", "k8s.mesosphere.io/attribute-num": "42"}, node1, true, "scalar label matches"},
		{map[string]string{"k8s.mesosphere.io/attribute-rack": "a", "k8s.mesosphere.io/attribute-num": "43"}, node1, false, "scalar label does not match"},

		{map[string]string{"kubernetes.io/hostname": "node1"}, node1, true, "hostname label matches"},
		{map[string]string{"kubernetes.io/hostname": "node2"}, node1, false, "hostname label does not match"},
		{map[string]string{"kubernetes.io/hostname": "node2"}, node2, true, "hostname label matches"},

		{map[string]string{"some.other/label": "43"}, node1, false, "non-slave attribute does not match"},
		{map[string]string{"some.other/label": "43"}, node3, true, "non-slave attribute matches"},
	}

	for _, ts := range tests {
		task, _ := fakePodTask("foo")
		task.Pod.Spec.NodeSelector = ts.selector
		offer := &mesos.Offer{
			Resources: []*mesos.Resource{
				mutil.NewScalarResource("cpus", t_min_cpu),
				mutil.NewScalarResource("mem", t_min_mem),
			},
			Hostname: &ts.node.Name,
		}
		if got, want := DefaultPredicate(task, offer, ts.node), ts.ok; got != want {
			t.Fatalf("expected acceptance of offer for selector %v to be %v, got %v: %q", ts.selector, want, got, ts.desc)
		}
	}
}

func newTextAttribute(name string, val string) *mesos.Attribute {
	return &mesos.Attribute{
		Name: proto.String(name),
		Type: mesos.Value_TEXT.Enum(),
		Text: &mesos.Value_Text{Value: &val},
	}
}

func newScalarAttribute(name string, val float64) *mesos.Attribute {
	return &mesos.Attribute{
		Name:   proto.String(name),
		Type:   mesos.Value_SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{Value: proto.Float64(val)},
	}
}
