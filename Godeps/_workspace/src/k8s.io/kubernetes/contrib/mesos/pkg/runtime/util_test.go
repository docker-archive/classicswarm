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

package runtime

import (
	"testing"
	"time"
)

func TestUntil(t *testing.T) {
	ch := make(chan struct{})
	close(ch)
	Until(func() {
		t.Fatal("should not have been invoked")
	}, 0, ch)

	//--
	ch = make(chan struct{})
	called := make(chan struct{})
	After(func() {
		Until(func() {
			called <- struct{}{}
		}, 0, ch)
	}).Then(func() { close(called) })

	<-called
	close(ch)
	<-called

	//--
	ch = make(chan struct{})
	called = make(chan struct{})
	running := make(chan struct{})
	After(func() {
		Until(func() {
			close(running)
			called <- struct{}{}
		}, 2*time.Second, ch)
	}).Then(func() { close(called) })

	<-running
	close(ch)
	<-called // unblock the goroutine
	now := time.Now()

	<-called
	if time.Since(now) > 1800*time.Millisecond {
		t.Fatalf("Until should not have waited the full timeout period since we closed the stop chan")
	}
}
