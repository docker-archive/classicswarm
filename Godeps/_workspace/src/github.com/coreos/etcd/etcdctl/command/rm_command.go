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

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

// NewRemoveCommand returns the CLI command for "rm".
func NewRemoveCommand() cli.Command {
	return cli.Command{
		Name:  "rm",
		Usage: "remove a key or a directory",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "dir", Usage: "removes the key if it is an empty directory or a key-value pair"},
			cli.BoolFlag{Name: "recursive", Usage: "removes the key and all child keys(if it is a directory)"},
			cli.StringFlag{Name: "with-value", Value: "", Usage: "previous value"},
			cli.IntFlag{Name: "with-index", Value: 0, Usage: "previous index"},
		},
		Action: func(c *cli.Context) {
			rmCommandFunc(c, mustNewKeyAPI(c))
		},
	}
}

// rmCommandFunc executes the "rm" command.
func rmCommandFunc(c *cli.Context, ki client.KeysAPI) {
	if len(c.Args()) == 0 {
		handleError(ExitBadArgs, errors.New("key required"))
	}
	key := c.Args()[0]
	recursive := c.Bool("recursive")
	dir := c.Bool("dir")
	prevValue := c.String("with-value")
	prevIndex := c.Int("with-index")

	resp, err := ki.Delete(context.TODO(), key, &client.DeleteOptions{PrevIndex: uint64(prevIndex), PrevValue: prevValue, Dir: dir, Recursive: recursive})
	if err != nil {
		handleError(ExitServerError, err)
	}

	if !resp.Node.Dir {
		printResponseKey(resp, c.GlobalString("output"))
	}
}
