package godo

import (
	"fmt"
	"net/url"
)

// ActionRequest reprents DigitalOcean Action Request
type ActionRequest map[string]interface{}

// DropletActionsService is an interface for interfacing with the droplet actions
// endpoints of the Digital Ocean API
// See: https://developers.digitalocean.com/documentation/v2#droplet-actions
type DropletActionsService interface {
	Shutdown(int) (*Action, *Response, error)
	PowerOff(int) (*Action, *Response, error)
	PowerOn(int) (*Action, *Response, error)
	PowerCycle(int) (*Action, *Response, error)
	Reboot(int) (*Action, *Response, error)
	Restore(int, int) (*Action, *Response, error)
	Resize(int, string, bool) (*Action, *Response, error)
	Rename(int, string) (*Action, *Response, error)
	Snapshot(int, string) (*Action, *Response, error)
	doAction(int, *ActionRequest) (*Action, *Response, error)
	Get(int, int) (*Action, *Response, error)
	GetByURI(string) (*Action, *Response, error)
}

// DropletActionsServiceOp handles communication with the droplet action related
// methods of the DigitalOcean API.
type DropletActionsServiceOp struct {
	client *Client
}

var _ DropletActionsService = &DropletActionsServiceOp{}

// Shutdown a Droplet
func (s *DropletActionsServiceOp) Shutdown(id int) (*Action, *Response, error) {
	request := &ActionRequest{"type": "shutdown"}
	return s.doAction(id, request)
}

// PowerOff a Droplet
func (s *DropletActionsServiceOp) PowerOff(id int) (*Action, *Response, error) {
	request := &ActionRequest{"type": "power_off"}
	return s.doAction(id, request)
}

// PowerOn a Droplet
func (s *DropletActionsServiceOp) PowerOn(id int) (*Action, *Response, error) {
	request := &ActionRequest{"type": "power_on"}
	return s.doAction(id, request)
}

// PowerCycle a Droplet
func (s *DropletActionsServiceOp) PowerCycle(id int) (*Action, *Response, error) {
	request := &ActionRequest{"type": "power_cycle"}
	return s.doAction(id, request)
}

// Reboot a Droplet
func (s *DropletActionsServiceOp) Reboot(id int) (*Action, *Response, error) {
	request := &ActionRequest{"type": "reboot"}
	return s.doAction(id, request)
}

// Restore an image to a Droplet
func (s *DropletActionsServiceOp) Restore(id, imageID int) (*Action, *Response, error) {
	requestType := "restore"
	request := &ActionRequest{
		"type":  requestType,
		"image": float64(imageID),
	}
	return s.doAction(id, request)
}

// Resize a Droplet
func (s *DropletActionsServiceOp) Resize(id int, sizeSlug string, resizeDisk bool) (*Action, *Response, error) {
	requestType := "resize"
	request := &ActionRequest{
		"type": requestType,
		"size": sizeSlug,
		"disk": resizeDisk,
	}
	return s.doAction(id, request)
}

// Rename a Droplet
func (s *DropletActionsServiceOp) Rename(id int, name string) (*Action, *Response, error) {
	requestType := "rename"
	request := &ActionRequest{
		"type": requestType,
		"name": name,
	}
	return s.doAction(id, request)
}

// Snapshot a Droplet
func (s *DropletActionsServiceOp) Snapshot(id int, name string) (*Action, *Response, error) {
	requestType := "snapshot"
	request := &ActionRequest{
		"type": requestType,
		"name": name,
	}
	return s.doAction(id, request)
}

func (s *DropletActionsServiceOp) doAction(id int, request *ActionRequest) (*Action, *Response, error) {
	path := dropletActionPath(id)

	req, err := s.client.NewRequest("POST", path, request)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}

// Get an action for a particular droplet by id.
func (s *DropletActionsServiceOp) Get(dropletID, actionID int) (*Action, *Response, error) {
	path := fmt.Sprintf("%s/%d", dropletActionPath(dropletID), actionID)
	return s.get(path)
}

// GetByURI gets an action for a particular droplet by id.
func (s *DropletActionsServiceOp) GetByURI(rawurl string) (*Action, *Response, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, nil, err
	}

	return s.get(u.Path)

}

func (s *DropletActionsServiceOp) get(path string) (*Action, *Response, error) {
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err

}

func dropletActionPath(dropletID int) string {
	return fmt.Sprintf("v2/droplets/%d/actions", dropletID)
}
