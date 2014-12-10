package token

import "testing"

func TestRegister(t *testing.T) {
	discovery := TokenDiscoveryService{token: "TEST_TOKEN"}
	expected := "127.0.0.1:2675"
	if err := discovery.RegisterNode(expected); err != nil {
		t.Fatal(err)
	}

	addrs, err := discovery.FetchNodes()
	if err != nil {
		t.Fatal(err)
	}

	if len(addrs) != 1 {
		t.Fatalf("expected addr len == 1, got len = %d", len(addrs))
	}

	if addrs[0] != expected {
		t.Fatalf("expected addr %q but received %q", expected, addrs[0])
	}

	if err = discovery.RegisterNode(expected); err != nil {
		t.Fatal(err)
	}
}
