package discovery

import "testing"

func TestRegisterLocal(t *testing.T) {
	expected := "127.0.0.1:2675"
	if err := RegisterSlave(expected, "TEST_TOKEN"); err != nil {
		t.Fatal(err)
	}

	addrs, err := FetchSlaves("TEST_TOKEN")
	if err != nil {
		t.Fatal(err)
	}

	if len(addrs) != 1 {
		t.Fatalf("expected addr len == 1, got len = %d", len(addrs))
	}

	if addrs[0] != expected {
		t.Fatalf("expected addr %q but received %q", expected, addrs[0])
	}

	if err = RegisterSlave(expected, "TEST_TOKEN"); err != nil {
		t.Fatal(err)
	}
}
