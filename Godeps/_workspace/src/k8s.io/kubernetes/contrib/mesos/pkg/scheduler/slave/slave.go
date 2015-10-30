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

package slave

import (
	"sync"
)

type Registry struct {
	lock      sync.Mutex
	hostNames map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		hostNames: map[string]string{},
	}
}

// Register creates a mapping between a slaveId and slave if not existing.
func (st *Registry) Register(slaveId, slaveHostname string) {
	st.lock.Lock()
	defer st.lock.Unlock()
	_, exists := st.hostNames[slaveId]
	if !exists {
		st.hostNames[slaveId] = slaveHostname
	}
}

// SlaveIDs returns the keys of the registry
func (st *Registry) SlaveIDs() []string {
	st.lock.Lock()
	defer st.lock.Unlock()
	slaveIds := make([]string, 0, len(st.hostNames))
	for slaveID := range st.hostNames {
		slaveIds = append(slaveIds, slaveID)
	}
	return slaveIds
}

// HostName looks up a hostname for a given slaveId
func (st *Registry) HostName(slaveId string) string {
	st.lock.Lock()
	defer st.lock.Unlock()
	return st.hostNames[slaveId]
}
