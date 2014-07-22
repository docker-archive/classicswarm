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

var verbs = []string{"Ack", "Attach", "Connect", "Error", "File", "Get", "Log", "Ls", "Set", "Spawn", "Start", "Stop", "Watch"}

func VerbFromString(s string) (Verb, error) {
	for i, verb := range verbs {
		if verb == s {
			return Verb(i), nil
		}
	}
	return 0, fmt.Errorf("Unrecognised verb: %s", s)
}

func (v Verb) String() string {
	if int(v) < len(verbs)-1 {
		return verbs[v]
	}
	return ""
}
