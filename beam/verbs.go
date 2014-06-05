package beam

type Verb string

var (
	Ack    Verb = "ack"
	Log    Verb = "log"
	Start  Verb = "start"
	Stop   Verb = "stop"
	Attach Verb = "attach"
	Spawn  Verb = "spawn"
	Set    Verb = "set"
	Get    Verb = "get"
	File   Verb = "file"
	Error  Verb = "error"
	Ls     Verb = "ls"
)
