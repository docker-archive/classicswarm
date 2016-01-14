package cli

import (
	"os"
	"path"

	"github.com/codegangsta/cli"
)

func init() {
	cli.AppHelpTemplate = `Usage: {{.Name}} {{if .Flags}}[OPTIONS] {{end}}COMMAND [arg...]

{{.Usage}}

Version: {{.Version}}{{if or .Author .Email}}

Author:{{if .Author}}
  {{.Author}}{{if .Email}} - <{{.Email}}>{{end}}{{else}}
  {{.Email}}{{end}}{{end}}
{{if .Flags}}
Options:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
Commands:
  {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
  {{end}}
Run '{{.Name}} COMMAND --help' for more information on a command.
`

	// See https://github.com/codegangsta/cli/pull/171/files
	cli.CommandHelpTemplate = `{{$DISCOVERY := or (eq .Name "manage") (eq .Name "join") (eq .Name "list")}}Usage: ` + path.Base(os.Args[0]) + ` {{.Name}}{{if .Flags}} [OPTIONS]{{end}} {{if $DISCOVERY}}<discovery>{{end}}

{{.Usage}}{{if $DISCOVERY}}

Arguments: 
   <discovery>    discovery service to use [$SWARM_DISCOVERY]
                   * token://<token>
                   * consul://<ip>/<path>
                   * etcd://<ip1>,<ip2>/<path>
                   * file://path/to/file
                   * zk://<ip1>,<ip2>/<path>
                   * [nodes://]<ip1>,<ip2>{{end}}{{if .Flags}}

Options:
   {{range .Flags}}{{.}}
   {{end}}{{if (eq .Name "manage")}}{{printf "\t * swarm.overcommit=0.05\tovercommit to apply on resources"}}
                                    {{printf "\t * swarm.createretry=0\tcontainer create retry count after initial failure"}}
                                    {{printf "\t * mesos.address=\taddress to bind on [$SWARM_MESOS_ADDRESS]"}}
                                    {{printf "\t * mesos.checkpointfailover=false\tcheckpointing allows a restarted slave to reconnect with old executors and recover status updates, at the cost of disk I/O [$SWARM_MESOS_CHECKPOINT_FAILOVER]"}}
                                    {{printf "\t * mesos.port=\tport to bind on [$SWARM_MESOS_PORT]"}}
                                    {{printf "\t * mesos.offertimeout=30s\ttimeout for offers [$SWARM_MESOS_OFFER_TIMEOUT]"}}
                                    {{printf "\t * mesos.offerrefusetimeout=5s\tseconds to consider unused resources refused [$SWARM_MESOS_OFFER_REFUSE_TIMEOUT]"}}
                                    {{printf "\t * mesos.tasktimeout=5s\ttimeout for task creation [$SWARM_MESOS_TASK_TIMEOUT]"}}
                                    {{printf "\t * mesos.user=\tframework user [$SWARM_MESOS_USER]"}}{{end}}{{ end }}
`

}
