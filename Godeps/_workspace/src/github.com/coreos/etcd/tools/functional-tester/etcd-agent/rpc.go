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

package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/coreos/etcd/tools/functional-tester/etcd-agent/client"
)

func (a *Agent) serveRPC() {
	rpc.Register(a)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":9027")
	if e != nil {
		log.Fatal("agent:", e)
	}
	go http.Serve(l, nil)
}

func (a *Agent) RPCStart(args []string, pid *int) error {
	log.Printf("rpc: start etcd with args %v", args)
	err := a.start(args...)
	if err != nil {
		log.Println("rpc: error starting etcd", err)
		return err
	}
	*pid = a.cmd.Process.Pid
	return nil
}

func (a *Agent) RPCStop(args struct{}, reply *struct{}) error {
	log.Printf("rpc: stop etcd")
	err := a.stop()
	if err != nil {
		log.Println("rpc: error stopping etcd", err)
		return err
	}
	return nil
}

func (a *Agent) RPCRestart(args struct{}, pid *int) error {
	log.Printf("rpc: restart etcd")
	err := a.restart()
	if err != nil {
		log.Println("rpc: error restarting etcd", err)
		return err
	}
	*pid = a.cmd.Process.Pid
	return nil
}

func (a *Agent) RPCCleanup(args struct{}, reply *struct{}) error {
	log.Printf("rpc: cleanup etcd")
	err := a.cleanup()
	if err != nil {
		log.Println("rpc: error cleaning up etcd", err)
		return err
	}
	return nil
}

func (a *Agent) RPCTerminate(args struct{}, reply *struct{}) error {
	log.Printf("rpc: terminate etcd")
	err := a.terminate()
	if err != nil {
		log.Println("rpc: error terminating etcd", err)
	}
	return nil
}

func (a *Agent) RPCDropPort(port int, reply *struct{}) error {
	log.Printf("rpc: drop port %d", port)
	err := a.dropPort(port)
	if err != nil {
		log.Println("rpc: error dropping port", err)
	}
	return nil
}

func (a *Agent) RPCRecoverPort(port int, reply *struct{}) error {
	log.Printf("rpc: recover port %d", port)
	err := a.recoverPort(port)
	if err != nil {
		log.Println("rpc: error recovering port", err)
	}
	return nil
}

func (a *Agent) RPCStatus(args struct{}, status *client.Status) error {
	*status = a.status()
	return nil
}
