package main

import "github.com/codegangsta/cli"

func init() {

	cli.CommandHelpTemplate = `{{$DISCOVERY := or (eq .Name "manage") (eq .Name "join") (eq .Name "list")}}NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   swarm {{.Name}}{{if .Flags}} [command options]{{end}} {{if $DISCOVERY}}[discovery]{{end}}{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if $DISCOVERY}}
ARGUMENTS:
   discovery{{printf "\t"}}discovery service to use [$SWARM_DISCOVERY]
            {{printf "\t"}} * token://<token>
            {{printf "\t"}} * etcd://<ip1>,<ip2>/<path>
            {{printf "\t"}} * file://path/to/file
            {{printf "\t"}} * zk://<ip1>,<ip2>/<path>
            {{printf "\t"}} * <ip1>,<ip2>{{end}}{{if .Flags}}
OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`

}
