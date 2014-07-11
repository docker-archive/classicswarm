package libswarm

import (
	"fmt"
)

type Verb uint32

const (
	Ack Verb = iota
	Attach
	Connect
	Error
	File
	Get
	Log
	Ls
	Set
	Spawn
	Start
	Stop
	Watch
)

var (
	VerbToString = map[Verb]string{
		Ack:     "Ack",
		Attach:  "Attach",
		Connect: "Connect",
		Error:   "Error",
		File:    "File",
		Get:     "Get",
		Log:     "Log",
		Ls:      "Ls",
		Set:     "Set",
		Spawn:   "Spawn",
		Start:   "Start",
		Stop:    "Stop",
		Watch:   "Watch",
	}
	VerbString = map[string]Verb{
		"Ack":     Ack,
		"Attach":  Attach,
		"Connect": Connect,
		"Error":   Error,
		"File":    File,
		"Get":     Get,
		"Log":     Log,
		"Ls":      Ls,
		"Set":     Set,
		"Spawn":   Spawn,
		"Start":   Start,
		"Stop":    Stop,
		"Watch":   Watch,
	}
)

func VerbFromString(s string) (Verb, error) {
	verb, ok := VerbString[s]

	if !ok {
		return 0, fmt.Errorf("Unrecognised verb: %s", s)
	}

	return verb, nil
}

func (v Verb) String() string {
	str, ok := VerbToString[v]

	if !ok {
		return ""
	}

	return str
}
