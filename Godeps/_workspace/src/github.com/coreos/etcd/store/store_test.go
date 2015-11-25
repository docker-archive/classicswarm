// Copyright 2015 CoreOS, Inc.
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

package store

import (
	"testing"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/jonboulle/clockwork"
	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	etcdErr "github.com/coreos/etcd/error"
)

func TestNewStoreWithNamespaces(t *testing.T) {
	s := newStore("/0", "/1")

	_, err := s.Get("/0", false, false)
	assert.Nil(t, err, "")
	_, err = s.Get("/1", false, false)
	assert.Nil(t, err, "")
}

// Ensure that the store can retrieve an existing value.
func TestStoreGetValue(t *testing.T) {
	s := newStore()
	s.Create("/foo", false, "bar", false, Permanent)
	var eidx uint64 = 1
	e, err := s.Get("/foo", false, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "get", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.Equal(t, *e.Node.Value, "bar", "")
}

// Ensure that any TTL <= minExpireTime becomes Permanent
func TestMinExpireTime(t *testing.T) {
	s := newStore()
	fc := clockwork.NewFakeClock()
	s.clock = fc
	// FakeClock starts at 0, so minExpireTime should be far in the future.. but just in case
	assert.True(t, minExpireTime.After(fc.Now()), "minExpireTime should be ahead of FakeClock!")
	s.Create("/foo", false, "Y", false, fc.Now().Add(3*time.Second))
	fc.Advance(5 * time.Second)
	// Ensure it hasn't expired
	s.DeleteExpiredKeys(fc.Now())
	var eidx uint64 = 1
	e, err := s.Get("/foo", true, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "get", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.Equal(t, e.Node.TTL, 0)
}

// Ensure that the store can recrusively retrieve a directory listing.
// Note that hidden files should not be returned.
func TestStoreGetDirectory(t *testing.T) {
	s := newStore()
	fc := newFakeClock()
	s.clock = fc
	s.Create("/foo", true, "", false, Permanent)
	s.Create("/foo/bar", false, "X", false, Permanent)
	s.Create("/foo/_hidden", false, "*", false, Permanent)
	s.Create("/foo/baz", true, "", false, Permanent)
	s.Create("/foo/baz/bat", false, "Y", false, Permanent)
	s.Create("/foo/baz/_hidden", false, "*", false, Permanent)
	s.Create("/foo/baz/ttl", false, "Y", false, fc.Now().Add(time.Second*3))
	var eidx uint64 = 7
	e, err := s.Get("/foo", true, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "get", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.Equal(t, len(e.Node.Nodes), 2, "")
	var bazNodes NodeExterns
	for _, node := range e.Node.Nodes {
		switch node.Key {
		case "/foo/bar":
			assert.Equal(t, *node.Value, "X", "")
			assert.Equal(t, node.Dir, false, "")
		case "/foo/baz":
			assert.Equal(t, node.Dir, true, "")
			assert.Equal(t, len(node.Nodes), 2, "")
			bazNodes = node.Nodes
		default:
			t.Errorf("key = %s, not matched", node.Key)
		}
	}
	for _, node := range bazNodes {
		switch node.Key {
		case "/foo/baz/bat":
			assert.Equal(t, *node.Value, "Y", "")
			assert.Equal(t, node.Dir, false, "")
		case "/foo/baz/ttl":
			assert.Equal(t, *node.Value, "Y", "")
			assert.Equal(t, node.Dir, false, "")
			assert.Equal(t, node.TTL, 3, "")
		default:
			t.Errorf("key = %s, not matched", node.Key)
		}
	}
}

// Ensure that the store can retrieve a directory in sorted order.
func TestStoreGetSorted(t *testing.T) {
	s := newStore()
	s.Create("/foo", true, "", false, Permanent)
	s.Create("/foo/x", false, "0", false, Permanent)
	s.Create("/foo/z", false, "0", false, Permanent)
	s.Create("/foo/y", true, "", false, Permanent)
	s.Create("/foo/y/a", false, "0", false, Permanent)
	s.Create("/foo/y/b", false, "0", false, Permanent)
	var eidx uint64 = 6
	e, err := s.Get("/foo", true, true)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	var yNodes NodeExterns
	for _, node := range e.Node.Nodes {
		switch node.Key {
		case "/foo/x":
		case "/foo/y":
			yNodes = node.Nodes
		case "/foo/z":
		default:
			t.Errorf("key = %s, not matched", node.Key)
		}
	}
	for _, node := range yNodes {
		switch node.Key {
		case "/foo/y/a":
		case "/foo/y/b":
		default:
			t.Errorf("key = %s, not matched", node.Key)
		}
	}
}

func TestSet(t *testing.T) {
	s := newStore()

	// Set /foo=""
	var eidx uint64 = 1
	e, err := s.Set("/foo", false, "", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "set", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "", "")
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(1), "")

	// Set /foo="bar"
	eidx = 2
	e, err = s.Set("/foo", false, "bar", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "set", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "bar", "")
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(2), "")
	// check prevNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")
	// Set /foo="baz" (for testing prevNode)
	eidx = 3
	e, err = s.Set("/foo", false, "baz", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "set", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(3), "")
	// check prevNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(2), "")

	// Set /dir as a directory
	eidx = 4
	e, err = s.Set("/dir", true, "", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "set", "")
	assert.Equal(t, e.Node.Key, "/dir", "")
	assert.True(t, e.Node.Dir, "")
	assert.Nil(t, e.Node.Value)
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(4), "")
}

// Ensure that the store can create a new key if it doesn't already exist.
func TestStoreCreateValue(t *testing.T) {
	s := newStore()
	// Create /foo=bar
	var eidx uint64 = 1
	e, err := s.Create("/foo", false, "bar", false, Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "bar", "")
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(1), "")

	// Create /empty=""
	eidx = 2
	e, err = s.Create("/empty", false, "", false, Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/empty", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "", "")
	assert.Nil(t, e.Node.Nodes, "")
	assert.Nil(t, e.Node.Expiration, "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(2), "")

}

// Ensure that the store can create a new directory if it doesn't already exist.
func TestStoreCreateDirectory(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	e, err := s.Create("/foo", true, "", false, Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.True(t, e.Node.Dir, "")
}

// Ensure that the store fails to create a key if it already exists.
func TestStoreCreateFailsIfExists(t *testing.T) {
	s := newStore()
	// create /foo as dir
	s.Create("/foo", true, "", false, Permanent)

	// create /foo as dir again
	e, _err := s.Create("/foo", true, "", false, Permanent)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeNodeExist, "")
	assert.Equal(t, err.Message, "Key already exists", "")
	assert.Equal(t, err.Cause, "/foo", "")
	assert.Equal(t, err.Index, uint64(1), "")
	assert.Nil(t, e, 0, "")
}

// Ensure that the store can update a key if it already exists.
func TestStoreUpdateValue(t *testing.T) {
	s := newStore()
	// create /foo=bar
	s.Create("/foo", false, "bar", false, Permanent)
	// update /foo="bzr"
	var eidx uint64 = 2
	e, err := s.Update("/foo", "baz", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(2), "")
	// check prevNode
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.TTL, 0, "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")

	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Equal(t, e.EtcdIndex, eidx, "")

	// update /foo=""
	eidx = 3
	e, err = s.Update("/foo", "", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.False(t, e.Node.Dir, "")
	assert.Equal(t, *e.Node.Value, "", "")
	assert.Equal(t, e.Node.TTL, 0, "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(3), "")
	// check prevNode
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "baz", "")
	assert.Equal(t, e.PrevNode.TTL, 0, "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(2), "")

	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, *e.Node.Value, "", "")
}

// Ensure that the store cannot update a directory.
func TestStoreUpdateFailsIfDirectory(t *testing.T) {
	s := newStore()
	s.Create("/foo", true, "", false, Permanent)
	e, _err := s.Update("/foo", "baz", Permanent)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeNotFile, "")
	assert.Equal(t, err.Message, "Not a file", "")
	assert.Equal(t, err.Cause, "/foo", "")
	assert.Nil(t, e, "")
}

// Ensure that the store can update the TTL on a value.
func TestStoreUpdateValueTTL(t *testing.T) {
	s := newStore()
	fc := newFakeClock()
	s.clock = fc

	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	_, err := s.Update("/foo", "baz", fc.Now().Add(500*time.Millisecond))
	e, _ := s.Get("/foo", false, false)
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())
	e, err = s.Get("/foo", false, false)
	assert.Nil(t, e, "")
	assert.Equal(t, err.(*etcdErr.Error).ErrorCode, etcdErr.EcodeKeyNotFound, "")
}

// Ensure that the store can update the TTL on a directory.
func TestStoreUpdateDirTTL(t *testing.T) {
	s := newStore()
	fc := newFakeClock()
	s.clock = fc

	var eidx uint64 = 3
	s.Create("/foo", true, "", false, Permanent)
	s.Create("/foo/bar", false, "baz", false, Permanent)
	e, err := s.Update("/foo", "", fc.Now().Add(500*time.Millisecond))
	assert.Equal(t, e.Node.Dir, true, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	e, _ = s.Get("/foo/bar", false, false)
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Equal(t, e.EtcdIndex, eidx, "")

	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())
	e, err = s.Get("/foo/bar", false, false)
	assert.Nil(t, e, "")
	assert.Equal(t, err.(*etcdErr.Error).ErrorCode, etcdErr.EcodeKeyNotFound, "")
}

// Ensure that the store can delete a value.
func TestStoreDeleteValue(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	e, err := s.Delete("/foo", false, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "delete", "")
	// check prevNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
}

// Ensure that the store can delete a directory if recursive is specified.
func TestStoreDeleteDiretory(t *testing.T) {
	s := newStore()
	// create directory /foo
	var eidx uint64 = 2
	s.Create("/foo", true, "", false, Permanent)
	// delete /foo with dir = true and recursive = false
	// this should succeed, since the directory is empty
	e, err := s.Delete("/foo", true, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "delete", "")
	// check prevNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, e.PrevNode.Dir, true, "")

	// create directory /foo and directory /foo/bar
	s.Create("/foo/bar", true, "", false, Permanent)
	// delete /foo with dir = true and recursive = false
	// this should fail, since the directory is not empty
	_, err = s.Delete("/foo", true, false)
	assert.NotNil(t, err, "")

	// delete /foo with dir=false and recursive = true
	// this should succeed, since recursive implies dir=true
	// and recursively delete should be able to delete all
	// items under the given directory
	e, err = s.Delete("/foo", false, true)
	assert.Nil(t, err, "")
	assert.Equal(t, e.Action, "delete", "")

}

// Ensure that the store cannot delete a directory if both of recursive
// and dir are not specified.
func TestStoreDeleteDiretoryFailsIfNonRecursiveAndDir(t *testing.T) {
	s := newStore()
	s.Create("/foo", true, "", false, Permanent)
	e, _err := s.Delete("/foo", false, false)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeNotFile, "")
	assert.Equal(t, err.Message, "Not a file", "")
	assert.Nil(t, e, "")
}

func TestRootRdOnly(t *testing.T) {
	s := newStore("/0")

	for _, tt := range []string{"/", "/0"} {
		_, err := s.Set(tt, true, "", Permanent)
		assert.NotNil(t, err, "")

		_, err = s.Delete(tt, true, true)
		assert.NotNil(t, err, "")

		_, err = s.Create(tt, true, "", false, Permanent)
		assert.NotNil(t, err, "")

		_, err = s.Update(tt, "", Permanent)
		assert.NotNil(t, err, "")

		_, err = s.CompareAndSwap(tt, "", 0, "", Permanent)
		assert.NotNil(t, err, "")
	}
}

func TestStoreCompareAndDeletePrevValue(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	e, err := s.CompareAndDelete("/foo", "bar", 0)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndDelete", "")
	assert.Equal(t, e.Node.Key, "/foo", "")

	// check pervNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")
	assert.Equal(t, e.PrevNode.CreatedIndex, uint64(1), "")
}

func TestStoreCompareAndDeletePrevValueFailsIfNotMatch(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	e, _err := s.CompareAndDelete("/foo", "baz", 0)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeTestFailed, "")
	assert.Equal(t, err.Message, "Compare failed", "")
	assert.Nil(t, e, "")
	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, *e.Node.Value, "bar", "")
}

func TestStoreCompareAndDeletePrevIndex(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	e, err := s.CompareAndDelete("/foo", "", 1)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndDelete", "")
	// check pervNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")
	assert.Equal(t, e.PrevNode.CreatedIndex, uint64(1), "")
}

func TestStoreCompareAndDeletePrevIndexFailsIfNotMatch(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	e, _err := s.CompareAndDelete("/foo", "", 100)
	assert.NotNil(t, _err, "")
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeTestFailed, "")
	assert.Equal(t, err.Message, "Compare failed", "")
	assert.Nil(t, e, "")
	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, *e.Node.Value, "bar", "")
}

// Ensure that the store cannot delete a directory.
func TestStoreCompareAndDeleteDiretoryFail(t *testing.T) {
	s := newStore()
	s.Create("/foo", true, "", false, Permanent)
	_, _err := s.CompareAndDelete("/foo", "", 0)
	assert.NotNil(t, _err, "")
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeNotFile, "")
}

// Ensure that the store can conditionally update a key if it has a previous value.
func TestStoreCompareAndSwapPrevValue(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	e, err := s.CompareAndSwap("/foo", "bar", 0, "baz", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndSwap", "")
	assert.Equal(t, *e.Node.Value, "baz", "")
	// check pervNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")
	assert.Equal(t, e.PrevNode.CreatedIndex, uint64(1), "")

	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, *e.Node.Value, "baz", "")
}

// Ensure that the store cannot conditionally update a key if it has the wrong previous value.
func TestStoreCompareAndSwapPrevValueFailsIfNotMatch(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	e, _err := s.CompareAndSwap("/foo", "wrong_value", 0, "baz", Permanent)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeTestFailed, "")
	assert.Equal(t, err.Message, "Compare failed", "")
	assert.Nil(t, e, "")
	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, *e.Node.Value, "bar", "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
}

// Ensure that the store can conditionally update a key if it has a previous index.
func TestStoreCompareAndSwapPrevIndex(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, Permanent)
	e, err := s.CompareAndSwap("/foo", "", 1, "baz", Permanent)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndSwap", "")
	assert.Equal(t, *e.Node.Value, "baz", "")
	// check prevNode
	assert.NotNil(t, e.PrevNode, "")
	assert.Equal(t, e.PrevNode.Key, "/foo", "")
	assert.Equal(t, *e.PrevNode.Value, "bar", "")
	assert.Equal(t, e.PrevNode.ModifiedIndex, uint64(1), "")
	assert.Equal(t, e.PrevNode.CreatedIndex, uint64(1), "")

	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, *e.Node.Value, "baz", "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
}

// Ensure that the store cannot conditionally update a key if it has the wrong previous index.
func TestStoreCompareAndSwapPrevIndexFailsIfNotMatch(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	e, _err := s.CompareAndSwap("/foo", "", 100, "baz", Permanent)
	err := _err.(*etcdErr.Error)
	assert.Equal(t, err.ErrorCode, etcdErr.EcodeTestFailed, "")
	assert.Equal(t, err.Message, "Compare failed", "")
	assert.Nil(t, e, "")
	e, _ = s.Get("/foo", false, false)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, *e.Node.Value, "bar", "")
}

// Ensure that the store can watch for key creation.
func TestStoreWatchCreate(t *testing.T) {
	s := newStore()
	var eidx uint64 = 0
	w, _ := s.Watch("/foo", false, false, 0)
	c := w.EventChan()
	assert.Equal(t, w.StartIndex(), eidx, "")
	s.Create("/foo", false, "bar", false, Permanent)
	eidx = 1
	e := nbselect(c)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	e = nbselect(c)
	assert.Nil(t, e, "")
}

// Ensure that the store can watch for recursive key creation.
func TestStoreWatchRecursiveCreate(t *testing.T) {
	s := newStore()
	var eidx uint64 = 0
	w, _ := s.Watch("/foo", true, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 1
	s.Create("/foo/bar", false, "baz", false, Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/foo/bar", "")
}

// Ensure that the store can watch for key updates.
func TestStoreWatchUpdate(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	w, _ := s.Watch("/foo", false, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.Update("/foo", "baz", Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
}

// Ensure that the store can watch for recursive key updates.
func TestStoreWatchRecursiveUpdate(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo/bar", false, "baz", false, Permanent)
	w, _ := s.Watch("/foo", true, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.Update("/foo/bar", "baz", Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/foo/bar", "")
}

// Ensure that the store can watch for key deletions.
func TestStoreWatchDelete(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	w, _ := s.Watch("/foo", false, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.Delete("/foo", false, false)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "delete", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
}

// Ensure that the store can watch for recursive key deletions.
func TestStoreWatchRecursiveDelete(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo/bar", false, "baz", false, Permanent)
	w, _ := s.Watch("/foo", true, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.Delete("/foo/bar", false, false)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "delete", "")
	assert.Equal(t, e.Node.Key, "/foo/bar", "")
}

// Ensure that the store can watch for CAS updates.
func TestStoreWatchCompareAndSwap(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo", false, "bar", false, Permanent)
	w, _ := s.Watch("/foo", false, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.CompareAndSwap("/foo", "bar", 0, "baz", Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndSwap", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
}

// Ensure that the store can watch for recursive CAS updates.
func TestStoreWatchRecursiveCompareAndSwap(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	s.Create("/foo/bar", false, "baz", false, Permanent)
	w, _ := s.Watch("/foo", true, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	eidx = 2
	s.CompareAndSwap("/foo/bar", "baz", 0, "bat", Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "compareAndSwap", "")
	assert.Equal(t, e.Node.Key, "/foo/bar", "")
}

// Ensure that the store can watch for key expiration.
func TestStoreWatchExpire(t *testing.T) {
	s := newStore()
	fc := newFakeClock()
	s.clock = fc

	var eidx uint64 = 2
	s.Create("/foo", false, "bar", false, fc.Now().Add(500*time.Millisecond))
	s.Create("/foofoo", false, "barbarbar", false, fc.Now().Add(500*time.Millisecond))

	w, _ := s.Watch("/", true, false, 0)
	assert.Equal(t, w.StartIndex(), eidx, "")
	c := w.EventChan()
	e := nbselect(c)
	assert.Nil(t, e, "")
	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())
	eidx = 3
	e = nbselect(c)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "expire", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	w, _ = s.Watch("/", true, false, 4)
	eidx = 4
	assert.Equal(t, w.StartIndex(), eidx, "")
	e = nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "expire", "")
	assert.Equal(t, e.Node.Key, "/foofoo", "")
}

// Ensure that the store can watch in streaming mode.
func TestStoreWatchStream(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	w, _ := s.Watch("/foo", false, true, 0)
	// first modification
	s.Create("/foo", false, "bar", false, Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.Equal(t, *e.Node.Value, "bar", "")
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
	// second modification
	eidx = 2
	s.Update("/foo", "baz", Permanent)
	e = nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/foo", "")
	assert.Equal(t, *e.Node.Value, "baz", "")
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store can recover from a previously saved state.
func TestStoreRecover(t *testing.T) {
	s := newStore()
	var eidx uint64 = 4
	s.Create("/foo", true, "", false, Permanent)
	s.Create("/foo/x", false, "bar", false, Permanent)
	s.Update("/foo/x", "barbar", Permanent)
	s.Create("/foo/y", false, "baz", false, Permanent)
	b, err := s.Save()

	s2 := newStore()
	s2.Recovery(b)

	e, err := s.Get("/foo/x", false, false)
	assert.Equal(t, e.Node.CreatedIndex, uint64(2), "")
	assert.Equal(t, e.Node.ModifiedIndex, uint64(3), "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Nil(t, err, "")
	assert.Equal(t, *e.Node.Value, "barbar", "")

	e, err = s.Get("/foo/y", false, false)
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Nil(t, err, "")
	assert.Equal(t, *e.Node.Value, "baz", "")
}

// Ensure that the store can recover from a previously saved state that includes an expiring key.
func TestStoreRecoverWithExpiration(t *testing.T) {
	s := newStore()
	s.clock = newFakeClock()

	fc := newFakeClock()

	var eidx uint64 = 4
	s.Create("/foo", true, "", false, Permanent)
	s.Create("/foo/x", false, "bar", false, Permanent)
	s.Create("/foo/y", false, "baz", false, fc.Now().Add(5*time.Millisecond))
	b, err := s.Save()

	time.Sleep(10 * time.Millisecond)

	s2 := newStore()
	s2.clock = fc

	s2.Recovery(b)

	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())

	e, err := s.Get("/foo/x", false, false)
	assert.Nil(t, err, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, *e.Node.Value, "bar", "")

	e, err = s.Get("/foo/y", false, false)
	assert.NotNil(t, err, "")
	assert.Nil(t, e, "")
}

// Ensure that the store can watch for hidden keys as long as it's an exact path match.
func TestStoreWatchCreateWithHiddenKey(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	w, _ := s.Watch("/_foo", false, false, 0)
	s.Create("/_foo", false, "bar", false, Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/_foo", "")
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store doesn't see hidden key creates without an exact path match in recursive mode.
func TestStoreWatchRecursiveCreateWithHiddenKey(t *testing.T) {
	s := newStore()
	w, _ := s.Watch("/foo", true, false, 0)
	s.Create("/foo/_bar", false, "baz", false, Permanent)
	e := nbselect(w.EventChan())
	assert.Nil(t, e, "")
	w, _ = s.Watch("/foo", true, false, 0)
	s.Create("/foo/_baz", true, "", false, Permanent)
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
	s.Create("/foo/_baz/quux", false, "quux", false, Permanent)
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store doesn't see hidden key updates.
func TestStoreWatchUpdateWithHiddenKey(t *testing.T) {
	s := newStore()
	s.Create("/_foo", false, "bar", false, Permanent)
	w, _ := s.Watch("/_foo", false, false, 0)
	s.Update("/_foo", "baz", Permanent)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.Action, "update", "")
	assert.Equal(t, e.Node.Key, "/_foo", "")
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store doesn't see hidden key updates without an exact path match in recursive mode.
func TestStoreWatchRecursiveUpdateWithHiddenKey(t *testing.T) {
	s := newStore()
	s.Create("/foo/_bar", false, "baz", false, Permanent)
	w, _ := s.Watch("/foo", true, false, 0)
	s.Update("/foo/_bar", "baz", Permanent)
	e := nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store can watch for key deletions.
func TestStoreWatchDeleteWithHiddenKey(t *testing.T) {
	s := newStore()
	var eidx uint64 = 2
	s.Create("/_foo", false, "bar", false, Permanent)
	w, _ := s.Watch("/_foo", false, false, 0)
	s.Delete("/_foo", false, false)
	e := nbselect(w.EventChan())
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "delete", "")
	assert.Equal(t, e.Node.Key, "/_foo", "")
	e = nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store doesn't see hidden key deletes without an exact path match in recursive mode.
func TestStoreWatchRecursiveDeleteWithHiddenKey(t *testing.T) {
	s := newStore()
	s.Create("/foo/_bar", false, "baz", false, Permanent)
	w, _ := s.Watch("/foo", true, false, 0)
	s.Delete("/foo/_bar", false, false)
	e := nbselect(w.EventChan())
	assert.Nil(t, e, "")
}

// Ensure that the store doesn't see expirations of hidden keys.
func TestStoreWatchExpireWithHiddenKey(t *testing.T) {
	s := newStore()
	fc := newFakeClock()
	s.clock = fc

	s.Create("/_foo", false, "bar", false, fc.Now().Add(500*time.Millisecond))
	s.Create("/foofoo", false, "barbarbar", false, fc.Now().Add(1000*time.Millisecond))

	w, _ := s.Watch("/", true, false, 0)
	c := w.EventChan()
	e := nbselect(c)
	assert.Nil(t, e, "")
	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())
	e = nbselect(c)
	assert.Nil(t, e, "")
	fc.Advance(600 * time.Millisecond)
	s.DeleteExpiredKeys(fc.Now())
	e = nbselect(c)
	assert.Equal(t, e.Action, "expire", "")
	assert.Equal(t, e.Node.Key, "/foofoo", "")
}

// Ensure that the store does see hidden key creates if watching deeper than a hidden key in recursive mode.
func TestStoreWatchRecursiveCreateDeeperThanHiddenKey(t *testing.T) {
	s := newStore()
	var eidx uint64 = 1
	w, _ := s.Watch("/_foo/bar", true, false, 0)
	s.Create("/_foo/bar/baz", false, "baz", false, Permanent)

	e := nbselect(w.EventChan())
	assert.NotNil(t, e, "")
	assert.Equal(t, e.EtcdIndex, eidx, "")
	assert.Equal(t, e.Action, "create", "")
	assert.Equal(t, e.Node.Key, "/_foo/bar/baz", "")
}

// Ensure that slow consumers are handled properly.
//
// Since Watcher.EventChan() has a buffer of size 1 we can only queue 1
// event per watcher. If the consumer cannot consume the event on time and
// another event arrives, the channel is closed and event is discarded.
// This test ensures that after closing the channel, the store can continue
// to operate correctly.
func TestStoreWatchSlowConsumer(t *testing.T) {
	s := newStore()
	s.Watch("/foo", true, true, 0)       // stream must be true
	s.Set("/foo", false, "1", Permanent) // ok
	s.Set("/foo", false, "2", Permanent) // ok
	s.Set("/foo", false, "3", Permanent) // must not panic
}

// Performs a non-blocking select on an event channel.
func nbselect(c <-chan *Event) *Event {
	select {
	case e := <-c:
		return e
	default:
		return nil
	}
}

// newFakeClock creates a new FakeClock that has been advanced to at least minExpireTime
func newFakeClock() clockwork.FakeClock {
	fc := clockwork.NewFakeClock()
	for minExpireTime.After(fc.Now()) {
		fc.Advance((0x1 << 62) * time.Nanosecond)
	}
	return fc
}
