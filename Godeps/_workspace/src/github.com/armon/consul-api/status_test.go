package consulapi

import (
	"testing"
)

func TestStatusLeader(t *testing.T) {
	c := makeClient(t)
	status := c.Status()

	leader, err := status.Leader()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if leader == "" {
		t.Fatalf("Expected leader")
	}
}

func TestStatusPeers(t *testing.T) {
	c := makeClient(t)
	status := c.Status()

	peers, err := status.Peers()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(peers) == 0 {
		t.Fatalf("Expected peers ")
	}
}
