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
	"flag"
	"log"
	"net/http"
	"strings"
)

func main() {
	endpointStr := flag.String("agent-endpoints", ":9027", "HTTP RPC endpoints of agents")
	datadir := flag.String("data-dir", "agent.etcd", "etcd data directory location on agent machine")
	stressKeySize := flag.Int("stress-key-size", 100, "the size of each key written into etcd")
	stressKeySuffixRange := flag.Int("stress-key-count", 250000, "the count of key range written into etcd")
	limit := flag.Int("limit", 3, "the limit of rounds to run failure set")
	flag.Parse()

	endpoints := strings.Split(*endpointStr, ",")
	c, err := newCluster(endpoints, *datadir, *stressKeySize, *stressKeySuffixRange)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Terminate()

	t := &tester{
		failures: []failure{
			newFailureKillAll(),
			newFailureKillMajority(),
			newFailureKillOne(),
			newFailureKillOneForLongTime(),
			newFailureIsolate(),
			newFailureIsolateAll(),
		},
		cluster: c,
		limit:   *limit,
	}

	sh := statusHandler{status: &t.status}
	http.Handle("/status", sh)
	go func() { log.Fatal(http.ListenAndServe(":9028", nil)) }()

	t.runLoop()
}
