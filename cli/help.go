package cli

import (
	"os"
	"path"

	"github.com/codegangsta/cli"
)

func init() {
	// See https://github.com/codegangsta/cli/pull/171/files
	cli.CommandHelpTemplate = `{{$DISCOVERY := or (eq .Name "manage") (eq .Name "join") (eq .Name "list")}}NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   ` + path.Base(os.Args[0]) + ` {{.Name}}{{if .Flags}} [options]{{end}} {{if $DISCOVERY}}<discovery>{{end}}{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if $DISCOVERY}}
ARGUMENTS:
   discovery{{printf "\t"}}discovery service to use [$SWARM_DISCOVERY]
            {{printf "\t"}} * token://<token>
            {{printf "\t"}} * consul://<ip>/<path>
            {{printf "\t"}} * etcd://<ip1>,<ip2>/<path>
            {{printf "\t"}} * file://path/to/file
            {{printf "\t"}} * zk://<ip1>,<ip2>/<path>
            {{printf "\t"}} * <ip1>,<ip2>{{end}}{{if .Flags}}
OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{if (eq .Name "manage")}}{{printf "\t * swarm.overcommit=0.05\tovercommit to apply on resources"}}
                                    {{printf "\t * swarm.discovery.heartbeat=25s\tperiod between each heartbeat"}}{{end}}{{ end }}
`

}
