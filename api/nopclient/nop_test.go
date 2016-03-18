package nopclient

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
)

func TestNop(t *testing.T) {
	nop := NewNopClient()
	_, err := nop.Info(context.TODO())
	if err != errNoEngine {
		t.Fatalf("Nop client did not return error")
	}
}

func TestNopInterface(t *testing.T) {
	iface := reflect.TypeOf((*client.APIClient)(nil)).Elem()
	nop := NewNopClient()

	if !reflect.TypeOf(nop).Implements(iface) {
		t.Fatalf("Nop does not implement the APIClient interface")
	}
}
