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

package resource

import (
	"fmt"
	"strconv"
)

type MegaBytes float64
type CPUShares float64

func (f *CPUShares) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = CPUShares(v)
	return err
}

func (f *CPUShares) Type() string {
	return "float64"
}

func (f *CPUShares) String() string { return fmt.Sprintf("%v", *f) }

func (f *MegaBytes) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = MegaBytes(v)
	return err
}

func (f *MegaBytes) Type() string {
	return "float64"
}

func (f *MegaBytes) String() string { return fmt.Sprintf("%v", *f) }
