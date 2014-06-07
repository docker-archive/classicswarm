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
