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

package scheduler

import (
	"errors"

	"k8s.io/kubernetes/contrib/mesos/pkg/offers"
	"k8s.io/kubernetes/contrib/mesos/pkg/scheduler/podtask"
)

type AllocationStrategy interface {
	// FitPredicate returns the selector used to determine pod fitness w/ respect to a given offer
	FitPredicate() podtask.FitPredicate

	// Procurement returns a func that obtains resources for a task from resource offer
	Procurement() podtask.Procurement
}

type PodScheduler interface {
	AllocationStrategy

	// SchedulePod implements how to schedule pods among slaves.
	// We can have different implementation for different scheduling policy.
	//
	// The function accepts a group of slaves (each contains offers from
	// that slave) and a single pod, which aligns well with the k8s scheduling
	// algorithm. It returns an offerId that is acceptable for the pod, otherwise
	// nil. The caller is responsible for filling in task state w/ relevant offer
	// details.
	//
	// See the FCFSPodScheduler for example.
	SchedulePod(r offers.Registry, slaves SlaveIndex, task *podtask.T) (offers.Perishable, error)
}

// A minimal placeholder
type empty struct{}

var (
	noSuitableOffersErr = errors.New("No suitable offers for pod/task")
	noSuchPodErr        = errors.New("No such pod exists")
	noSuchTaskErr       = errors.New("No such task exists")
)

type SlaveIndex interface {
	slaveHostNameFor(id string) string
}
