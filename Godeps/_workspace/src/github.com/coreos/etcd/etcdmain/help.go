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

package etcdmain

var (
	usageline = `usage: etcd [flags]
       start an etcd server

       etcd --version
       show the version of etcd

       etcd -h | --help
       show the help information about etcd
	`
	flagsline = `
member flags:

	--name 'default'
		human-readable name for this member.
	--data-dir '${name}.etcd'
		path to the data directory.
	--wal-dir ''
		path to the dedicated wal directory.
	--snapshot-count '10000'
		number of committed transactions to trigger a snapshot to disk.
	--heartbeat-interval '100'
		time (in milliseconds) of a heartbeat interval.
	--election-timeout '1000'
		time (in milliseconds) for an election to timeout. See tuning documentation for details.
	--listen-peer-urls 'http://localhost:2380,http://localhost:7001'
		list of URLs to listen on for peer traffic.
	--listen-client-urls 'http://localhost:2379,http://localhost:4001'
		list of URLs to listen on for client traffic.
	-cors ''
		comma-separated whitelist of origins for CORS (cross-origin resource sharing).


clustering flags:

	--initial-advertise-peer-urls 'http://localhost:2380,http://localhost:7001'
		list of this member's peer URLs to advertise to the rest of the cluster.
	--initial-cluster 'default=http://localhost:2380,default=http://localhost:7001'
		initial cluster configuration for bootstrapping.
	--initial-cluster-state 'new'
		initial cluster state ('new' or 'existing').
	--initial-cluster-token 'etcd-cluster'
		initial cluster token for the etcd cluster during bootstrap.
	--advertise-client-urls 'http://localhost:2379,http://localhost:4001'
		list of this member's client URLs to advertise to the public.
		The client URLs advertised should be accessible to machines that talk to etcd cluster. etcd client libraries parse these URLs to connect to the cluster.
	--discovery ''
		discovery URL used to bootstrap the cluster.
	--discovery-fallback 'proxy'
		expected behavior ('exit' or 'proxy') when discovery services fails.
	--discovery-proxy ''
		HTTP proxy to use for traffic to discovery service.
	--discovery-srv ''
		dns srv domain used to bootstrap the cluster.
	--strict-reconfig-check
		reject reconfiguration requests that would cause quorum loss.

proxy flags:

	--proxy 'off'
		proxy mode setting ('off', 'readonly' or 'on').
	--proxy-failure-wait 5000
		time (in milliseconds) an endpoint will be held in a failed state.
	--proxy-refresh-interval 30000
		time (in milliseconds) of the endpoints refresh interval.
	--proxy-dial-timeout 1000
		time (in milliseconds) for a dial to timeout.
	--proxy-write-timeout 5000
		time (in milliseconds) for a write to timeout.
	--proxy-read-timeout 0
		time (in milliseconds) for a read to timeout.


security flags:

	--ca-file '' [DEPRECATED]
		path to the client server TLS CA file. '-ca-file ca.crt' could be replaced by '-trusted-ca-file ca.crt -client-cert-auth' and etcd will perform the same.
	--cert-file ''
		path to the client server TLS cert file.
	--key-file ''
		path to the client server TLS key file.
	--client-cert-auth 'false'
		enable client cert authentication.
	--trusted-ca-file ''
		path to the client server TLS trusted CA key file.
	--peer-ca-file '' [DEPRECATED]
		path to the peer server TLS CA file. '-peer-ca-file ca.crt' could be replaced by '-peer-trusted-ca-file ca.crt -peer-client-cert-auth' and etcd will perform the same.
	--peer-cert-file ''
		path to the peer server TLS cert file.
	--peer-key-file ''
		path to the peer server TLS key file.
	--peer-client-cert-auth 'false'
		enable peer client cert authentication.
	--peer-trusted-ca-file ''
		path to the peer server TLS trusted CA file.

logging flags

	--debug 'false'
		enable debug-level logging for etcd.
	--log-package-levels ''
		set individual packages to various log levels (eg: 'etcdmain=CRITICAL,etcdserver=DEBUG')

unsafe flags:

Please be CAUTIOUS when using unsafe flags because it will break the guarantees
given by the consensus protocol.

	--force-new-cluster 'false'
		force to create a new one-member cluster.


experimental flags:

	--experimental-v3demo 'false'
		enable experimental v3 demo API
`
)
