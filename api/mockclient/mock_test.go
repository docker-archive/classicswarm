package mockclient

import (
	"reflect"
	"testing"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
)

func TestMock(t *testing.T) {
	mock := NewMockClient()
	mock.On("ServerVersion").Return(types.Version{Version: "foo"}, nil).Once()

	v, err := mock.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "foo" {
		t.Fatal(v)
	}

	mock.Mock.AssertExpectations(t)
}

func TestMockInterface(t *testing.T) {
	iface := reflect.TypeOf((*client.APIClient)(nil)).Elem()
	mock := NewMockClient()

	if !reflect.TypeOf(mock).Implements(iface) {
		t.Fatalf("Mock does not implement the APIClient interface")
	}
}
