/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// CAUTION: If you update code in this file, you may need to also update code
//          in contrib/mesos/cmd/km/hyperkube.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"

	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/version/verflag"

	"github.com/spf13/pflag"
)

// HyperKube represents a single binary that can morph/manage into multiple
// servers.
type HyperKube struct {
	Name string // The executable name, used for help and soft-link invocation
	Long string // A long description of the binary.  It will be world wrapped before output.

	servers     []Server
	baseFlags   *pflag.FlagSet
	out         io.Writer
	helpFlagVal bool
}

// AddServer adds a server to the HyperKube object.
func (hk *HyperKube) AddServer(s *Server) {
	hk.servers = append(hk.servers, *s)
	hk.servers[len(hk.servers)-1].hk = hk
}

// FindServer will find a specific server named name.
func (hk *HyperKube) FindServer(name string) (*Server, error) {
	for _, s := range hk.servers {
		if s.Name() == name {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("Server not found: %s", name)
}

// Servers returns a list of all of the registred servers
func (hk *HyperKube) Servers() []Server {
	return hk.servers
}

// Flags returns a flagset for "global" flags.
func (hk *HyperKube) Flags() *pflag.FlagSet {
	if hk.baseFlags == nil {
		hk.baseFlags = pflag.NewFlagSet(hk.Name, pflag.ContinueOnError)
		hk.baseFlags.SetOutput(ioutil.Discard)
		hk.baseFlags.SetNormalizeFunc(util.WordSepNormalizeFunc)
		hk.baseFlags.BoolVarP(&hk.helpFlagVal, "help", "h", false, "help for "+hk.Name)

		// These will add all of the "global" flags (defined with both the
		// flag and pflag packages) to the new flag set we have.
		hk.baseFlags.AddGoFlagSet(flag.CommandLine)
		hk.baseFlags.AddFlagSet(pflag.CommandLine)

	}
	return hk.baseFlags
}

// Out returns the io.Writer that is used for all usage/error information
func (hk *HyperKube) Out() io.Writer {
	if hk.out == nil {
		hk.out = os.Stderr
	}
	return hk.out
}

// SetOut sets the output writer for all usage/error information
func (hk *HyperKube) SetOut(w io.Writer) {
	hk.out = w
}

// Print is a convenience method to Print to the defined output
func (hk *HyperKube) Print(i ...interface{}) {
	fmt.Fprint(hk.Out(), i...)
}

// Println is a convenience method to Println to the defined output
func (hk *HyperKube) Println(i ...interface{}) {
	fmt.Fprintln(hk.Out(), i...)
}

// Printf is a convenience method to Printf to the defined output
func (hk *HyperKube) Printf(format string, i ...interface{}) {
	fmt.Fprintf(hk.Out(), format, i...)
}

// Run the server.  This will pick the appropriate server and run it.
func (hk *HyperKube) Run(args []string) error {
	// If we are called directly, parse all flags up to the first real
	// argument.  That should be the server to run.
	baseCommand := path.Base(args[0])
	serverName := baseCommand
	if serverName == hk.Name {
		args = args[1:]

		baseFlags := hk.Flags()
		baseFlags.SetInterspersed(false) // Only parse flags up to the next real command
		err := baseFlags.Parse(args)
		if err != nil || hk.helpFlagVal {
			if err != nil {
				hk.Println("Error:", err)
			}
			hk.Usage()
			return err
		}

		verflag.PrintAndExitIfRequested()

		args = baseFlags.Args()
		if len(args) > 0 && len(args[0]) > 0 {
			serverName = args[0]
			baseCommand = baseCommand + " " + serverName
			args = args[1:]
		} else {
			err = errors.New("No server specified")
			hk.Printf("Error: %v\n\n", err)
			hk.Usage()
			return err
		}
	}

	s, err := hk.FindServer(serverName)
	if err != nil {
		hk.Printf("Error: %v\n\n", err)
		hk.Usage()
		return err
	}

	s.Flags().AddFlagSet(hk.Flags())
	err = s.Flags().Parse(args)
	if err != nil || hk.helpFlagVal {
		if err != nil {
			hk.Printf("Error: %v\n\n", err)
		}
		s.Usage()
		return err
	}

	verflag.PrintAndExitIfRequested()

	util.InitLogs()
	defer util.FlushLogs()

	err = s.Run(s, s.Flags().Args())
	if err != nil {
		hk.Println("Error:", err)
	}

	return err
}

// RunToExit will run the hyperkube and then call os.Exit with an appropriate exit code.
func (hk *HyperKube) RunToExit(args []string) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	err := hk.Run(args)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// Usage will write out a summary for all servers that this binary supports.
func (hk *HyperKube) Usage() {
	tt := `{{if .Long}}{{.Long | trim | wrap ""}}
{{end}}Usage

  {{.Name}} <server> [flags]

Servers
{{range .Servers}}
  {{.Name}}
{{.Long | trim | wrap "    "}}{{end}}
Call '{{.Name}} <server> --help' for help on a specific server.
`
	util.ExecuteTemplate(hk.Out(), tt, hk)
}
