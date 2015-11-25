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

package cinder

import (
	"testing"

	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/mount"
)

func TestSafeFormatAndMount(t *testing.T) {
	tests := []struct {
		fstype       string
		expectedArgs []string
		err          error
	}{
		{
			fstype:       "ext4",
			expectedArgs: []string{"/dev/foo", "/mnt/bar"},
		},
		{
			fstype:       "ext3",
			expectedArgs: []string{"/dev/foo/blah", "/mnt/bar/blah"},
		},
	}
	for _, test := range tests {

		var cmdOut string
		var argsOut []string
		fake := exec.FakeExec{
			CommandScript: []exec.FakeCommandAction{
				func(cmd string, args ...string) exec.Cmd {
					cmdOut = cmd
					argsOut = args
					fake := exec.FakeCmd{
						CombinedOutputScript: []exec.FakeCombinedOutputAction{
							func() ([]byte, error) { return []byte{}, test.err },
						},
					}
					return exec.InitFakeCmd(&fake, cmd, args...)
				},
			},
		}

		mounter := cinderSafeFormatAndMount{
			&mount.FakeMounter{},
			&fake,
		}

		err := mounter.Mount("/dev/foo", "/mnt/bar", test.fstype, nil)
		if test.err == nil && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if test.err != nil {
			if err == nil {
				t.Errorf("unexpected non-error")
			}
			return
		}
		if cmdOut != "/bin/lsblk" {
			t.Errorf("unexpected command: %s", cmdOut)
		}
		if len(argsOut) != len(test.expectedArgs) {
			t.Errorf("unexpected args: %v, expected: %v", argsOut, test.expectedArgs)
		}
	}
}
