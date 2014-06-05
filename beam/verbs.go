package beam

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
