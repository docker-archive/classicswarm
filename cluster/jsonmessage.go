package cluster

import "encoding/json"

// Types lifted from docker/docker/pkg/jsonmessage to avoid TTY build breakages

// JSONError represents a JSON Error
type JSONError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// JSONProgress represents a JSON-encoded progress instance
type JSONProgress struct {
	//terminalFd uintptr
	Current int64 `json:"current,omitempty"`
	Total   int64 `json:"total,omitempty"`
	Start   int64 `json:"start,omitempty"`
}

// JSONMessage represents a JSON-encoded message regarding the status of a stream
type JSONMessage struct {
	Stream          string        `json:"stream,omitempty"`
	Status          string        `json:"status,omitempty"`
	Progress        *JSONProgress `json:"progressDetail,omitempty"`
	ProgressMessage string        `json:"progress,omitempty"` //deprecated
	ID              string        `json:"id,omitempty"`
	From            string        `json:"from,omitempty"`
	Time            int64         `json:"time,omitempty"`
	TimeNano        int64         `json:"timeNano,omitempty"`
	Error           *JSONError    `json:"errorDetail,omitempty"`
	ErrorMessage    string        `json:"error,omitempty"` //deprecated
	// Aux contains out-of-band data, such as digests for push signing.
	Aux *json.RawMessage `json:"aux,omitempty"`
}

// JSONMessageWrapper is used in callback functions for API calls that send back
// JSONMessages; this allows us to pass info around within Swarm classic.
type JSONMessageWrapper struct {
	Msg        JSONMessage
	EngineName string
	Err        error
	Success    bool
}
