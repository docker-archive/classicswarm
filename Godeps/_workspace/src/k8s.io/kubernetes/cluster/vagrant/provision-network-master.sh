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

# provision-network-master configures flannel on the master
function provision-network-master {

  echo "Provisioning network on master"

  FLANNEL_ETCD_URL="http://${MASTER_IP}:4379"

  # Install etcd for flannel data
  if ! which etcd >/dev/null 2>&1; then

    yum install -y etcd

    # Modify etcd configuration for flannel data
    cat <<EOF >/etc/etcd/etcd.conf
ETCD_NAME=flannel
ETCD_DATA_DIR="/var/lib/etcd/flannel.etcd"
ETCD_LISTEN_PEER_URLS="http://${MASTER_IP}:4380"
ETCD_LISTEN_CLIENT_URLS="http://${MASTER_IP}:4379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="http://${MASTER_IP}:4380"
ETCD_INITIAL_CLUSTER="flannel=http://${MASTER_IP}:4380"
ETCD_ADVERTISE_CLIENT_URLS="${FLANNEL_ETCD_URL}"
EOF
    # Enable and start etcd
    systemctl enable etcd
    systemctl start etcd

  fi

  # Install flannel for overlay
  if ! which flanneld >/dev/null 2>&1; then

    yum install -y flannel

    cat <<EOF >/etc/flannel-config.json
{
    "Network": "${CONTAINER_SUBNET}",
    "SubnetLen": 24,
    "Backend": {
        "Type": "udp",
        "Port": 8285
     }
}
EOF

    # Import default configuration into etcd for master setup
    etcdctl -C ${FLANNEL_ETCD_URL} set /coreos.com/network/config < /etc/flannel-config.json

    # Configure local daemon to speak to master
    NETWORK_CONF_PATH=/etc/sysconfig/network-scripts/
    if_to_edit=$( find ${NETWORK_CONF_PATH}ifcfg-* | xargs grep -l VAGRANT-BEGIN )
    NETWORK_IF_NAME=`echo ${if_to_edit} | awk -F- '{ print $3 }'`
    cat <<EOF >/etc/sysconfig/flanneld
FLANNEL_ETCD="${FLANNEL_ETCD_URL}"
FLANNEL_ETCD_KEY="/coreos.com/network"
FLANNEL_OPTIONS="-iface=${NETWORK_IF_NAME}"
EOF

    # Start flannel
    systemctl enable flanneld
    systemctl start flanneld
  fi

  echo "Network configuration verified"
}
