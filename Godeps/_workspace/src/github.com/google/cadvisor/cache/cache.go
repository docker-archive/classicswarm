// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import info "github.com/google/cadvisor/info/v1"

type Cache interface {
	// Add a ContainerStats for the specified container.
	AddStats(ref info.ContainerReference, stats *info.ContainerStats) error

	// Remove all cached information for the specified container.
	RemoveContainer(containerName string) error

	// Read most recent stats. numStats indicates max number of stats
	// returned. The returned stats must be consecutive observed stats. If
	// numStats < 0, then return all stats stored in the storage. The
	// returned stats should be sorted in time increasing order, i.e. Most
	// recent stats should be the last.
	RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error)

	// Close will clear the state of the storage driver. The elements
	// stored in the underlying storage may or may not be deleted depending
	// on the implementation of the storage driver.
	Close() error
}
