package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestDropletActions_Shutdown(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "shutdown",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.Shutdown(1)
	if err != nil {
		t.Errorf("DropletActions.Shutdown returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Shutdown returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_PowerOff(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "power_off",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.PowerOff(1)
	if err != nil {
		t.Errorf("DropletActions.PowerOff returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Poweroff returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_PowerOn(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "power_on",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.PowerOn(1)
	if err != nil {
		t.Errorf("DropletActions.PowerOn returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.PowerOn returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_Reboot(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "reboot",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)

	})

	action, _, err := client.DropletActions.Reboot(1)
	if err != nil {
		t.Errorf("DropletActions.Reboot returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Reboot returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_Restore(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type":  "restore",
		"image": float64(1),
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")

		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)

	})

	action, _, err := client.DropletActions.Restore(1, 1)
	if err != nil {
		t.Errorf("DropletActions.Restore returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Restore returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_Resize(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "resize",
		"size": "1024mb",
		"disk": true,
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")

		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)

	})

	action, _, err := client.DropletActions.Resize(1, "1024mb", true)
	if err != nil {
		t.Errorf("DropletActions.Resize returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Resize returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_Rename(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "rename",
		"name": "Droplet-Name",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")

		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.Rename(1, "Droplet-Name")
	if err != nil {
		t.Errorf("DropletActions.Rename returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Rename returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_PowerCycle(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "power_cycle",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)

	})

	action, _, err := client.DropletActions.PowerCycle(1)
	if err != nil {
		t.Errorf("DropletActions.PowerCycle returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.PowerCycle returned %+v, expected %+v", action, expected)
	}
}

func TestDropletAction_Snapshot(t *testing.T) {
	setup()
	defer teardown()

	request := &ActionRequest{
		"type": "snapshot",
		"name": "Image-Name",
	}

	mux.HandleFunc("/v2/droplets/1/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")

		if !reflect.DeepEqual(v, request) {
			t.Errorf("Request body = %+v, expected %+v", v, request)
		}

		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.Snapshot(1, "Image-Name")
	if err != nil {
		t.Errorf("DropletActions.Snapshot returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Snapshot returned %+v, expected %+v", action, expected)
	}
}

func TestDropletActions_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/droplets/123/actions/456", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	action, _, err := client.DropletActions.Get(123, 456)
	if err != nil {
		t.Errorf("DropletActions.Get returned error: %v", err)
	}

	expected := &Action{Status: "in-progress"}
	if !reflect.DeepEqual(action, expected) {
		t.Errorf("DropletActions.Get returned %+v, expected %+v", action, expected)
	}
}
