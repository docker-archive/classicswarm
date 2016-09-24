package mockclient

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/swarm/swarmclient"
	"github.com/stretchr/testify/mock"
)

func TestMock(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.On("ServerVersion", mock.Anything).Return(types.Version{Version: "foo"}, nil).Once()

	v, err := mockClient.ServerVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "foo" {
		t.Fatal(v)
	}

	mockClient.Mock.AssertExpectations(t)
}

func TestMockInterface(t *testing.T) {
	iface := reflect.TypeOf((*swarmclient.SwarmAPIClient)(nil)).Elem()
	mockClient := NewMockClient()

	if !reflect.TypeOf(mockClient).Implements(iface) {
		t.Fatalf("Mock does not implement the SwarmAPIClient interface")
	}
}
