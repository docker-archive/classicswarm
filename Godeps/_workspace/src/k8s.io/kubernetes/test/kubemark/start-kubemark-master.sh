#!/bin/bash

# Copyright 2015 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# TODO: figure out how to get etcd tag from some real configuration and put it here.
sudo docker run --net=host -d gcr.io/google_containers/etcd:2.0.12 /usr/local/bin/etcd --addr=127.0.0.1:4001 --bind-addr=0.0.0.0:4001 --data-dir=/var/etcd/data

# Increase the allowed number of open file descriptors
ulimit -n 65536

tar xzf kubernetes-server-linux-amd64.tar.gz

kubernetes/server/bin/kube-controller-manager --master=127.0.0.1:8080 --service-account-private-key-file=/srv/kubernetes/server.key --root-ca-file=/srv/kubernetes/ca.crt --v=2 &> /tmp/kube-controller-manager.log &

kubernetes/server/bin/kube-scheduler --master=127.0.0.1:8080 --v=2 &> /tmp/kube-scheduler.log &

kubernetes/server/bin/kube-apiserver \
	--portal-net=10.0.0.1/24 \
	--address=0.0.0.0 \
	--etcd-servers=http://127.0.0.1:4001 \
	--cluster-name=hollow-kubernetes \
	--v=4 \
	--tls-cert-file=/srv/kubernetes/server.cert \
	--tls-private-key-file=/srv/kubernetes/server.key \
	--client-ca-file=/srv/kubernetes/ca.crt \
	--token-auth-file=/srv/kubernetes/known_tokens.csv \
	--secure-port=443 \
	--basic-auth-file=/srv/kubernetes/basic_auth.csv &> /tmp/kube-apiserver.log &

rm -rf kubernetes
