package cluster

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "store-test")
	assert.NoError(t, err)
	store := NewStore(dir)
	assert.NoError(t, store.Initialize())

	c1 := &Container{}
	c1.Id = "foo"
	c2 := &Container{}
	c2.Id = "bar"

	var ret *Container

	// Add "foo" into the store.
	assert.NoError(t, store.Add("foo", c1))

	// Retrieve "foo" from the store.
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c1.Id, ret.Id)

	// Try to add "foo" again.
	assert.EqualError(t, store.Add("foo", c1), ErrAlreadyExists.Error())

	// Replace "foo" with c2.
	assert.NoError(t, store.Replace("foo", c2))
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c2.Id, ret.Id)

	// Initialize a brand new store and retrieve "foo" again.
	// This is to ensure data load on initialization works correctly.
	store = NewStore(dir)
	assert.NoError(t, store.Initialize())
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c2.Id, ret.Id)
}
