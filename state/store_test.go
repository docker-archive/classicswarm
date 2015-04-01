package state

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

	c1 := &RequestedState{}
	c1.Name = "foo"
	c2 := &RequestedState{}
	c2.Name = "bar"

	var ret *RequestedState

	// Add an invalid key
	assert.EqualError(t, store.Add("", c1), ErrInvalidKey.Error())

	// Add "foo" into the store.
	assert.NoError(t, store.Add("foo", c1))

	// Retrieve "foo" from the store.
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c1.Name, ret.Name)

	// Try to add "foo" again.
	assert.EqualError(t, store.Add("foo", c1), ErrAlreadyExists.Error())

	// Replace "foo" with c2.
	assert.NoError(t, store.Replace("foo", c2))
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c2.Name, ret.Name)

	// Only one item in the store
	all := store.All()
	assert.Equal(t, 1, len(all))
	// The same name
	assert.Equal(t, c2.Name, all[0].Name)
	// It's actually the same pointer
	assert.Equal(t, c2, all[0])

	// Initialize a brand new store and retrieve "foo" again.
	// This is to ensure data load on initialization works correctly.
	store = NewStore(dir)
	assert.NoError(t, store.Initialize())
	ret, err = store.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, c2.Name, ret.Name)
}
