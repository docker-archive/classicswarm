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

func VerbFromString(s string) (Verb, error) {
	switch s {
	case "Ack":
		return Ack, nil
	case "Attach":
		return Attach, nil
	case "Connect":
		return Connect, nil
	case "Error":
		return Error, nil
	case "File":
		return File, nil
	case "Get":
		return Get, nil
	case "Log":
		return Log, nil
	case "Ls":
		return Ls, nil
	case "Set":
		return Set, nil
	case "Spawn":
		return Spawn, nil
	case "Start":
		return Start, nil
	case "Stop":
		return Stop, nil
	case "Watch":
		return Watch, nil
	}
	return 0, fmt.Errorf("Unrecognised verb: %s", s)
}

func (v Verb) String() string {
	switch v {
	case Ack:
		return "Ack"
	case Attach:
		return "Attach"
	case Connect:
		return "Connect"
	case Error:
		return "Error"
	case File:
		return "File"
	case Get:
		return "Get"
	case Log:
		return "Log"
	case Ls:
		return "Ls"
	case Set:
		return "Set"
	case Spawn:
		return "Spawn"
	case Start:
		return "Start"
	case Stop:
		return "Stop"
	case Watch:
		return "Watch"
	}
	return ""
}
