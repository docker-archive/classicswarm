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
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

// NewMakeDirCommand returns the CLI command for "mkdir".
func NewMakeDirCommand() cli.Command {
	return cli.Command{
		Name:  "mkdir",
		Usage: "make a new directory",
		Flags: []cli.Flag{
			cli.IntFlag{Name: "ttl", Value: 0, Usage: "key time-to-live"},
		},
		Action: func(c *cli.Context) {
			mkdirCommandFunc(c, mustNewKeyAPI(c), client.PrevNoExist)
		},
	}
}

// mkdirCommandFunc executes the "mkdir" command.
func mkdirCommandFunc(c *cli.Context, ki client.KeysAPI, prevExist client.PrevExistType) {
	if len(c.Args()) == 0 {
		handleError(ExitBadArgs, errors.New("key required"))
	}

	key := c.Args()[0]
	ttl := c.Int("ttl")

	_, err := ki.Set(context.TODO(), key, "", &client.SetOptions{TTL: time.Duration(ttl) * time.Second, Dir: true, PrevExist: prevExist})
	if err != nil {
		handleError(ExitServerError, err)
	}
}
