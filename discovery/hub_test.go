package discovery

import (
	"testing"
)

func TestUrl(t *testing.T) {
	if setupUrl(DOCKER_HUB_SCHEME) != DISCOVERY_URL_DEFAULT {
		t.Fatal("Bad url")
	}

	if setupUrl(DOCKER_HUB_SCHEME+DISCOVERY_URL_DEFAULT) != DISCOVERY_URL_DEFAULT {
		t.Fatal("Bad url")
	}

	if setupUrl(DOCKER_HUB_SCHEME+"test/path") != "https://test/path" {
		t.Fatal("Bad url")
	}

	if setupUrl(DOCKER_HUB_SCHEME+"http://test/path") != "http://test/path" {
		t.Fatal("Bad url")
	}
}

func TestSupports(t *testing.T) {
	hub := &DockerHubDiscovery{}

	if ok, _ := hub.Supports("blah"); ok {
		t.Fatal("Bad url")
	}

	if ok, _ := hub.Supports(DOCKER_HUB_SCHEME); !ok {
		t.Fatal("Bad url")
	}

	if ok, _ := hub.Supports(DOCKER_HUB_SCHEME + "garbage"); !ok {
		t.Fatal("Bad url")
	}

	if ok, _ := hub.Supports(DOCKER_HUB_SCHEME + "http://blah"); !ok {
		t.Fatal("Bad url")
	}

	if ok, _ := hub.Supports(DOCKER_HUB_SCHEME + "https://blah"); !ok {
		t.Fatal("Bad url")
	}
}

func TestRegister(t *testing.T) {
	url := DOCKER_HUB_SCHEME
	hub := &DockerHubDiscovery{}

	expected := "127.0.0.1:2675"
	if err := hub.RegisterSlave(url, expected, "TEST_TOKEN"); err != nil {
		t.Fatal(err)
	}

	addrs, err := hub.FetchSlaves(url, "TEST_TOKEN")
	if err != nil {
		t.Fatal(err)
	}

	if len(addrs) != 1 {
		t.Fatalf("expected addr len == 1, got len = %d", len(addrs))
	}

	if addrs[0] != expected {
		t.Fatalf("expected addr %q but received %q", expected, addrs[0])
	}

	if err = hub.RegisterSlave(url, expected, "TEST_TOKEN"); err != nil {
		t.Fatal(err)
	}
}
