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

# A library of helper functions that each provider hosting Kubernetes must implement to use cluster/kube-*.sh scripts.

# exit on any error
set -e

SSH_OPTS="-oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -oLogLevel=ERROR"

# Use the config file specified in $KUBE_CONFIG_FILE, or default to
# config-default.sh.
KUBE_ROOT=$(dirname "${BASH_SOURCE}")/../..
readonly ROOT=$(dirname "${BASH_SOURCE}")
source "${ROOT}/${KUBE_CONFIG_FILE:-"config-default.sh"}"
source "$KUBE_ROOT/cluster/common.sh"


KUBECTL_PATH=${KUBE_ROOT}/cluster/centos/binaries/kubectl

# Directory to be used for master and minion provisioning.
KUBE_TEMP="~/kube_temp"


# Must ensure that the following ENV vars are set
function detect-master() {
  KUBE_MASTER=$MASTER
  KUBE_MASTER_IP=${MASTER#*@}
  echo "KUBE_MASTER_IP: ${KUBE_MASTER_IP}" 1>&2
  echo "KUBE_MASTER: ${MASTER}" 1>&2
}

# Get minion IP addresses and store in KUBE_MINION_IP_ADDRESSES[]
function detect-minions() {
  KUBE_MINION_IP_ADDRESSES=()
  for minion in ${MINIONS}; do
    KUBE_MINION_IP_ADDRESSES+=("${minion#*@}")
  done
  echo "KUBE_MINION_IP_ADDRESSES: [${KUBE_MINION_IP_ADDRESSES[*]}]" 1>&2
}

# Verify prereqs on host machine
function verify-prereqs() {
  local rc
  rc=0
  ssh-add -L 1> /dev/null 2> /dev/null || rc="$?"
  # "Could not open a connection to your authentication agent."
  if [[ "${rc}" -eq 2 ]]; then
    eval "$(ssh-agent)" > /dev/null
    trap-add "kill ${SSH_AGENT_PID}" EXIT
  fi
  rc=0
  ssh-add -L 1> /dev/null 2> /dev/null || rc="$?"
  # "The agent has no identities."
  if [[ "${rc}" -eq 1 ]]; then
    # Try adding one of the default identities, with or without passphrase.
    ssh-add || true
  fi
  rc=0
  # Expect at least one identity to be available.
  if ! ssh-add -L 1> /dev/null 2> /dev/null; then
    echo "Could not find or add an SSH identity."
    echo "Please start ssh-agent, add your identity, and retry."
    exit 1
  fi
}

# Install handler for signal trap
function trap-add {
  local handler="$1"
  local signal="${2-EXIT}"
  local cur

  cur="$(eval "sh -c 'echo \$3' -- $(trap -p ${signal})")"
  if [[ -n "${cur}" ]]; then
    handler="${cur}; ${handler}"
  fi

  trap "${handler}" ${signal}
}

# Validate a kubernetes cluster
function validate-cluster() {
  # by default call the generic validate-cluster.sh script, customizable by
  # any cluster provider if this does not fit.
  "${KUBE_ROOT}/cluster/validate-cluster.sh"
}

# Instantiate a kubernetes cluster
function kube-up() {
  provision-master

  for minion in ${MINIONS}; do
    provision-minion ${minion}
  done

  verify-master
  for minion in ${MINIONS}; do
    verify-minion ${minion}
  done

  detect-master

  # set CONTEXT and KUBE_SERVER values for create-kubeconfig() and get-password()
  export CONTEXT="centos"
  export KUBE_SERVER="http://${KUBE_MASTER_IP}:8080"
  source "${KUBE_ROOT}/cluster/common.sh"

  # set kubernetes user and password
  get-password
  create-kubeconfig
}

# Delete a kubernetes cluster
function kube-down() {
  tear-down-master
  for minion in ${MINIONS}; do
    tear-down-minion ${minion}
  done
}


function verify-master() {
  # verify master has all required daemons
  printf "[INFO] Validating master ${MASTER}"
  local -a required_daemon=("kube-apiserver" "kube-controller-manager" "kube-scheduler")
  local validated="1"
  local try_count=0
  until [[ "$validated" == "0" ]]; do
    validated="0"
    local daemon
    for daemon in "${required_daemon[@]}"; do
      local rc=0
      kube-ssh "${MASTER}" "sudo pgrep -f ${daemon}" >/dev/null 2>&1 || rc="$?"
      if [[ "${rc}" -ne "0" ]]; then
        printf "."
        validated="1"
        ((try_count=try_count+2))
        if [[ ${try_count} -gt ${PROCESS_CHECK_TIMEOUT} ]]; then
          printf "\nWarning: Process \"${daemon}\" failed to run on ${MASTER}, please check.\n"
          exit 1
        fi
        sleep 2
      fi
    done
  done
  printf "\n"

}

function verify-minion() {
  # verify minion has all required daemons
  printf "[INFO] Validating minion ${1}"
  local -a required_daemon=("kube-proxy" "kubelet" "docker")
  local validated="1"
  local try_count=0
  until [[ "$validated" == "0" ]]; do
    validated="0"
    local daemon
    for daemon in "${required_daemon[@]}"; do
      local rc=0
      kube-ssh "${1}" "sudo pgrep -f ${daemon}" >/dev/null 2>&1 || rc="$?"
      if [[ "${rc}" -ne "0" ]]; then
        printf "."
        validated="1"
        ((try_count=try_count+2))
        if [[ ${try_count} -gt ${PROCESS_CHECK_TIMEOUT} ]] ; then
          printf "\nWarning: Process \"${daemon}\" failed to run on ${1}, please check.\n"
          exit 1
        fi
        sleep 2
      fi
    done
  done
  printf "\n"
}

# Clean up on master
function tear-down-master() {
echo "[INFO] tear-down-master on ${MASTER}"
  for service_name in etcd kube-apiserver kube-controller-manager kube-scheduler ; do
      service_file="/usr/lib/systemd/system/${service_name}.service"
      kube-ssh "$MASTER" " \
        if [[ -f $service_file ]]; then \
          sudo systemctl stop $service_name; \
          sudo systemctl disable $service_name; \
          sudo rm -f $service_file; \
        fi"
  done
  kube-ssh "${MASTER}" "sudo rm -rf /opt/kubernetes"
  kube-ssh "${MASTER}" "sudo rm -rf ${KUBE_TEMP}"
  kube-ssh "${MASTER}" "sudo rm -rf /var/lib/etcd"
}

# Clean up on minion
function tear-down-minion() {
echo "[INFO] tear-down-minion on $1"
  for service_name in kube-proxy kubelet docker flannel ; do
      service_file="/usr/lib/systemd/system/${service_name}.service"
      kube-ssh "$1" " \
        if [[ -f $service_file ]]; then \
          sudo systemctl stop $service_name; \
          sudo systemctl disable $service_name; \
          sudo rm -f $service_file; \
        fi"
  done
  kube-ssh "$1" "sudo rm -rf /run/flannel"
  kube-ssh "$1" "sudo rm -rf /opt/kubernetes"
  kube-ssh "$1" "sudo rm -rf ${KUBE_TEMP}"
}

# Provision master
#
# Assumed vars:
#   MASTER
#   KUBE_TEMP
#   ETCD_SERVERS
#   SERVICE_CLUSTER_IP_RANGE
function provision-master() {
  echo "[INFO] Provision master on ${MASTER}"
  local master_ip=${MASTER#*@}
  ensure-setup-dir ${MASTER}

  # scp -r ${SSH_OPTS} master config-default.sh copy-files.sh util.sh "${MASTER}:${KUBE_TEMP}" 
  kube-scp ${MASTER} "${ROOT}/../saltbase/salt/generate-cert/make-ca-cert.sh ${ROOT}/binaries/master ${ROOT}/master ${ROOT}/config-default.sh ${ROOT}/util.sh" "${KUBE_TEMP}" 
  kube-ssh "${MASTER}" " \
    sudo cp -r ${KUBE_TEMP}/master/bin /opt/kubernetes; \
    sudo chmod -R +x /opt/kubernetes/bin; \
    sudo bash ${KUBE_TEMP}/make-ca-cert.sh ${master_ip} IP:${master_ip},IP:${SERVICE_CLUSTER_IP_RANGE%.*}.1,DNS:kubernetes,DNS:kubernetes.default,DNS:kubernetes.default.svc,DNS:kubernetes.default.svc.cluster.local; \
    sudo bash ${KUBE_TEMP}/master/scripts/etcd.sh; \
    sudo bash ${KUBE_TEMP}/master/scripts/apiserver.sh ${master_ip} ${ETCD_SERVERS} ${SERVICE_CLUSTER_IP_RANGE} ${ADMISSION_CONTROL}; \
    sudo bash ${KUBE_TEMP}/master/scripts/controller-manager.sh ${master_ip}; \
    sudo bash ${KUBE_TEMP}/master/scripts/scheduler.sh ${master_ip}"
}


# Provision minion
#
# Assumed vars:
#   $1 (minion)
#   MASTER
#   KUBE_TEMP
#   ETCD_SERVERS
#   FLANNEL_NET
#   DOCKER_OPTS
function provision-minion() {
  echo "[INFO] Provision minion on $1"
  local master_ip=${MASTER#*@}
  local minion=$1
  local minion_ip=${minion#*@}
  ensure-setup-dir ${minion}

  # scp -r ${SSH_OPTS} minion config-default.sh copy-files.sh util.sh "${minion_ip}:${KUBE_TEMP}" 
  kube-scp ${minion} "${ROOT}/binaries/node ${ROOT}/node ${ROOT}/config-default.sh ${ROOT}/util.sh" ${KUBE_TEMP}
  kube-ssh "${minion}" " \
    sudo cp -r ${KUBE_TEMP}/node/bin /opt/kubernetes; \
    sudo chmod -R +x /opt/kubernetes/bin; \
    sudo bash ${KUBE_TEMP}/node/scripts/flannel.sh ${ETCD_SERVERS} ${FLANNEL_NET}; \
    sudo bash ${KUBE_TEMP}/node/scripts/docker.sh \"${DOCKER_OPTS}\"; \
    sudo bash ${KUBE_TEMP}/node/scripts/kubelet.sh ${master_ip} ${minion_ip}; \
    sudo bash ${KUBE_TEMP}/node/scripts/proxy.sh ${master_ip}"
}

# Create dirs that'll be used during setup on target machine.
#
# Assumed vars:
#   KUBE_TEMP
function ensure-setup-dir() {
  kube-ssh "${1}" "mkdir -p ${KUBE_TEMP}; \
                   sudo mkdir -p /opt/kubernetes/bin; \
                   sudo mkdir -p /opt/kubernetes/cfg"
}

# Run command over ssh
function kube-ssh() {
  local host="$1"
  shift
  ssh ${SSH_OPTS} -t "${host}" "$@" >/dev/null 2>&1
}

# Copy file recursively over ssh
function kube-scp() {
  local host="$1"
  local src=($2)
  local dst="$3"
  scp -r ${SSH_OPTS} ${src[*]} "${host}:${dst}"
}

# Ensure that we have a password created for validating to the master. Will
# read from kubeconfig if available.
#
# Vars set:
#   KUBE_USER
#   KUBE_PASSWORD
function get-password {
  get-kubeconfig-basicauth
  if [[ -z "${KUBE_USER}" || -z "${KUBE_PASSWORD}" ]]; then
    KUBE_USER=admin
    KUBE_PASSWORD=$(python -c 'import string,random; \
      print "".join(random.SystemRandom().choice(string.ascii_letters + string.digits) for _ in range(16))')
  fi
}
