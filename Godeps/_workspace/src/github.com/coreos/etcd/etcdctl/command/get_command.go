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
	"os"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

// NewGetCommand returns the CLI command for "get".
func NewGetCommand() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "retrieve the value of a key",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "sort", Usage: "returns result in sorted order"},
		},
		Action: func(c *cli.Context) {
			getCommandFunc(c, mustNewKeyAPI(c))
		},
	}
}

// getCommandFunc executes the "get" command.
func getCommandFunc(c *cli.Context, ki client.KeysAPI) {
	if len(c.Args()) == 0 {
		handleError(ExitBadArgs, errors.New("key required"))
	}

	key := c.Args()[0]
	sorted := c.Bool("sort")

	resp, err := ki.Get(context.TODO(), key, &client.GetOptions{Sort: sorted})
	if err != nil {
		handleError(ExitServerError, err)
	}

	if resp.Node.Dir {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s: is a directory", resp.Node.Key))
		os.Exit(1)
	}

	printResponseKey(resp, c.GlobalString("output"))
}
