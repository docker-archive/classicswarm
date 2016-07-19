package nopclient

import (
	"reflect"
	"testing"

	"github.com/docker/swarm/swarmclient"
	"golang.org/x/net/context"
)

func TestNop(t *testing.T) {
	nop := NewNopClient()
	_, err := nop.Info(context.Background())
	if err != errNoEngine {
		t.Fatalf("Nop client did not return error")
	}
}

func TestNopInterface(t *testing.T) {
	iface := reflect.TypeOf((*swarmclient.SwarmAPIClient)(nil)).Elem()
	nop := NewNopClient()

	if !reflect.TypeOf(nop).Implements(iface) {
		t.Fatalf("Nop does not implement the SwarmAPIClient interface")
	}
}
