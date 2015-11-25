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

package etcdserver

import (
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/coreos/etcd/pkg/netutil"
	"github.com/coreos/etcd/pkg/types"
)

// ServerConfig holds the configuration of etcd as taken from the command line or discovery.
type ServerConfig struct {
	Name           string
	DiscoveryURL   string
	DiscoveryProxy string
	ClientURLs     types.URLs
	PeerURLs       types.URLs
	DataDir        string
	// DedicatedWALDir config will make the etcd to write the WAL to the WALDir
	// rather than the dataDir/member/wal.
	DedicatedWALDir     string
	SnapCount           uint64
	MaxSnapFiles        uint
	MaxWALFiles         uint
	InitialPeerURLsMap  types.URLsMap
	InitialClusterToken string
	NewCluster          bool
	ForceNewCluster     bool
	Transport           *http.Transport

	TickMs        uint
	ElectionTicks int

	V3demo bool

	StrictReconfigCheck bool
}

// VerifyBootstrapConfig sanity-checks the initial config for bootstrap case
// and returns an error for things that should never happen.
func (c *ServerConfig) VerifyBootstrap() error {
	if err := c.verifyLocalMember(true); err != nil {
		return err
	}
	if checkDuplicateURL(c.InitialPeerURLsMap) {
		return fmt.Errorf("initial cluster %s has duplicate url", c.InitialPeerURLsMap)
	}
	if c.InitialPeerURLsMap.String() == "" && c.DiscoveryURL == "" {
		return fmt.Errorf("initial cluster unset and no discovery URL found")
	}
	return nil
}

// VerifyJoinExisting sanity-checks the initial config for join existing cluster
// case and returns an error for things that should never happen.
func (c *ServerConfig) VerifyJoinExisting() error {
	// no need for strict checking since the member have announced its
	// peer urls to the cluster before starting and do not have to set
	// it in the configuration again.
	if err := c.verifyLocalMember(false); err != nil {
		return err
	}
	if checkDuplicateURL(c.InitialPeerURLsMap) {
		return fmt.Errorf("initial cluster %s has duplicate url", c.InitialPeerURLsMap)
	}
	if c.DiscoveryURL != "" {
		return fmt.Errorf("discovery URL should not be set when joining existing initial cluster")
	}
	return nil
}

// verifyLocalMember verifies the configured member is in configured
// cluster. If strict is set, it also verifies the configured member
// has the same peer urls as configured advertised peer urls.
func (c *ServerConfig) verifyLocalMember(strict bool) error {
	urls := c.InitialPeerURLsMap[c.Name]
	// Make sure the cluster at least contains the local server.
	if urls == nil {
		return fmt.Errorf("couldn't find local name %q in the initial cluster configuration", c.Name)
	}

	// Advertised peer URLs must match those in the cluster peer list
	apurls := c.PeerURLs.StringSlice()
	sort.Strings(apurls)
	urls.Sort()
	if strict {
		if !netutil.URLStringsEqual(apurls, urls.StringSlice()) {
			umap := map[string]types.URLs{c.Name: c.PeerURLs}
			return fmt.Errorf("--initial-cluster must include %s given --initial-advertise-peer-urls=%s", types.URLsMap(umap).String(), strings.Join(apurls, ","))
		}
	}
	return nil
}

func (c *ServerConfig) MemberDir() string { return path.Join(c.DataDir, "member") }

func (c *ServerConfig) WALDir() string {
	if c.DedicatedWALDir != "" {
		return c.DedicatedWALDir
	}
	return path.Join(c.MemberDir(), "wal")
}

func (c *ServerConfig) SnapDir() string { return path.Join(c.MemberDir(), "snap") }

func (c *ServerConfig) StorageDir() string { return path.Join(c.MemberDir(), "storage") }

func (c *ServerConfig) ShouldDiscover() bool { return c.DiscoveryURL != "" }

// ReqTimeout returns timeout for request to finish.
func (c *ServerConfig) ReqTimeout() time.Duration {
	// 5s for queue waiting, computation and disk IO delay
	// + 2 * election timeout for possible leader election
	return 5*time.Second + 2*time.Duration(c.ElectionTicks)*time.Duration(c.TickMs)*time.Millisecond
}

func (c *ServerConfig) PrintWithInitial() { c.print(true) }

func (c *ServerConfig) Print() { c.print(false) }

func (c *ServerConfig) print(initial bool) {
	plog.Infof("name = %s", c.Name)
	if c.ForceNewCluster {
		plog.Infof("force new cluster")
	}
	plog.Infof("data dir = %s", c.DataDir)
	plog.Infof("member dir = %s", c.MemberDir())
	if c.DedicatedWALDir != "" {
		plog.Infof("dedicated WAL dir = %s", c.DedicatedWALDir)
	}
	plog.Infof("heartbeat = %dms", c.TickMs)
	plog.Infof("election = %dms", c.ElectionTicks*int(c.TickMs))
	plog.Infof("snapshot count = %d", c.SnapCount)
	if len(c.DiscoveryURL) != 0 {
		plog.Infof("discovery URL= %s", c.DiscoveryURL)
		if len(c.DiscoveryProxy) != 0 {
			plog.Infof("discovery proxy = %s", c.DiscoveryProxy)
		}
	}
	plog.Infof("advertise client URLs = %s", c.ClientURLs)
	if initial {
		plog.Infof("initial advertise peer URLs = %s", c.PeerURLs)
		plog.Infof("initial cluster = %s", c.InitialPeerURLsMap)
	}
}

func checkDuplicateURL(urlsmap types.URLsMap) bool {
	um := make(map[string]bool)
	for _, urls := range urlsmap {
		for _, url := range urls {
			u := url.String()
			if um[u] {
				return true
			}
			um[u] = true
		}
	}
	return false
}
