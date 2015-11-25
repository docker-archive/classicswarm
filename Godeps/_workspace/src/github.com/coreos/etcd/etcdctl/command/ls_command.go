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
	"fmt"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

func NewLsCommand() cli.Command {
	return cli.Command{
		Name:  "ls",
		Usage: "retrieve a directory",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "sort", Usage: "returns result in sorted order"},
			cli.BoolFlag{Name: "recursive", Usage: "returns all key names recursively for the given path"},
			cli.BoolFlag{Name: "p", Usage: "append slash (/) to directories"},
		},
		Action: func(c *cli.Context) {
			lsCommandFunc(c, mustNewKeyAPI(c))
		},
	}
}

// lsCommandFunc executes the "ls" command.
func lsCommandFunc(c *cli.Context, ki client.KeysAPI) {
	key := "/"
	if len(c.Args()) != 0 {
		key = c.Args()[0]
	}

	sort := c.Bool("sort")
	recursive := c.Bool("recursive")

	resp, err := ki.Get(context.TODO(), key, &client.GetOptions{Sort: sort, Recursive: recursive})
	if err != nil {
		handleError(ExitServerError, err)
	}

	printLs(c, resp)
}

// printLs writes a response out in a manner similar to the `ls` command in unix.
// Non-empty directories list their contents and files list their name.
func printLs(c *cli.Context, resp *client.Response) {
	if !resp.Node.Dir {
		fmt.Println(resp.Node.Key)
	}
	for _, node := range resp.Node.Nodes {
		rPrint(c, node)
	}
}

// rPrint recursively prints out the nodes in the node structure.
func rPrint(c *cli.Context, n *client.Node) {
	if n.Dir && c.Bool("p") {
		fmt.Println(fmt.Sprintf("%v/", n.Key))
	} else {
		fmt.Println(n.Key)
	}

	for _, node := range n.Nodes {
		rPrint(c, node)
	}
}
