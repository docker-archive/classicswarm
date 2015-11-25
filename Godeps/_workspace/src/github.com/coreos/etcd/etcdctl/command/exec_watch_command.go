// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

// NewExecWatchCommand returns the CLI command for "exec-watch".
func NewExecWatchCommand() cli.Command {
	return cli.Command{
		Name:  "exec-watch",
		Usage: "watch a key for changes and exec an executable",
		Flags: []cli.Flag{
			cli.IntFlag{Name: "after-index", Value: 0, Usage: "watch after the given index"},
			cli.BoolFlag{Name: "recursive", Usage: "watch all values for key and child keys"},
		},
		Action: func(c *cli.Context) {
			execWatchCommandFunc(c, mustNewKeyAPI(c))
		},
	}
}

// execWatchCommandFunc executes the "exec-watch" command.
func execWatchCommandFunc(c *cli.Context, ki client.KeysAPI) {
	args := c.Args()
	argslen := len(args)

	if argslen < 2 {
		handleError(ExitBadArgs, errors.New("key and command to exec required"))
	}

	var (
		key     string
		cmdArgs []string
	)

	foundSep := false
	for i := range args {
		if args[i] == "--" && i != 0 {
			foundSep = true
			break
		}
	}

	if foundSep {
		key = args[0]
		cmdArgs = args[2:]
	} else {
		// If no flag is parsed, the order of key and cmdArgs will be switched and
		// args will not contain `--`.
		key = args[argslen-1]
		cmdArgs = args[:argslen-1]
	}

	index := 0
	if c.Int("after-index") != 0 {
		index = c.Int("after-index") + 1
	}

	recursive := c.Bool("recursive")

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)

	go func() {
		<-sigch
		os.Exit(0)
	}()

	w := ki.Watcher(key, &client.WatcherOptions{AfterIndex: uint64(index), Recursive: recursive})

	for {
		resp, err := w.Next(context.TODO())
		if err != nil {
			handleError(ExitServerError, err)
		}
		if resp.Node.Dir {
			fmt.Fprintf(os.Stderr, "Ignored dir %s change", resp.Node.Key)
			continue
		}

		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Env = environResponse(resp, os.Environ())

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}

		go func() {
			err := cmd.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
			go io.Copy(os.Stdout, stdout)
			go io.Copy(os.Stderr, stderr)
			cmd.Wait()
		}()
	}
}

func environResponse(resp *client.Response, env []string) []string {
	env = append(env, "ETCD_WATCH_ACTION="+resp.Action)
	env = append(env, "ETCD_WATCH_MODIFIED_INDEX="+fmt.Sprintf("%d", resp.Node.ModifiedIndex))
	env = append(env, "ETCD_WATCH_KEY="+resp.Node.Key)
	env = append(env, "ETCD_WATCH_VALUE="+resp.Node.Value)
	return env
}
