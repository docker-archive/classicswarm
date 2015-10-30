#!/bin/bash

# Copyright 2014 The Kubernetes Authors All rights reserved.
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

set -o errexit
set -o nounset
set -o pipefail

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..

source "${KUBE_ROOT}/cluster/common.sh"
source "${KUBE_ROOT}/hack/lib/init.sh"

# Ginkgo will build the e2e tests, so we need to make sure that the environment
# is set up correctly (including Godeps, etc).
kube::golang::setup_env
# Find the ginkgo binary build as part of the release.
ginkgo=$(kube::util::find-binary "ginkgo")
e2e_test=$(kube::util::find-binary "e2e.test")

# --- Setup some env vars.

: ${KUBE_VERSION_ROOT:=${KUBE_ROOT}}
: ${KUBECTL:="${KUBE_VERSION_ROOT}/cluster/kubectl.sh"}
: ${KUBE_CONFIG_FILE:="config-test.sh"}

export KUBECTL KUBE_CONFIG_FILE

source "${KUBE_ROOT}/cluster/kube-env.sh"

# ---- Do cloud-provider-specific setup
if [[ -n "${KUBERNETES_CONFORMANCE_TEST:-}" ]]; then
    echo "Conformance test: not doing test setup."
    KUBERNETES_PROVIDER=""

    detect-master-from-kubeconfig

    auth_config=(
      "--kubeconfig=${KUBECONFIG}"
    )
else
    echo "Setting up for KUBERNETES_PROVIDER=\"${KUBERNETES_PROVIDER}\"."

    source "${KUBE_VERSION_ROOT}/cluster/${KUBERNETES_PROVIDER}/util.sh"

    prepare-e2e

    detect-master >/dev/null
    KUBE_MASTER_URL="${KUBE_MASTER_URL:-https://${KUBE_MASTER_IP:-}}"

    auth_config=(
      "--kubeconfig=${KUBECONFIG:-$DEFAULT_KUBECONFIG}"
    )
fi

if [[ -n "${NODE_INSTANCE_PREFIX:-}" ]]; then
  NODE_INSTANCE_GROUP="${NODE_INSTANCE_PREFIX}-group"
else
  NODE_INSTANCE_GROUP=""
fi

if [[ "${KUBERNETES_PROVIDER}" == "gke" ]]; then
  detect-node-instance-group
fi

ginkgo_args=()
if [[ -n "${CONFORMANCE_TEST_SKIP_REGEX:-}" ]]; then
  ginkgo_args+=("--skip=${CONFORMANCE_TEST_SKIP_REGEX}")
  ginkgo_args+=("--seed=1436380640")
fi
if [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
  ginkgo_args+=("-p")
fi


# The --host setting is used only when providing --auth_config
# If --kubeconfig is used, the host to use is retrieved from the .kubeconfig
# file and the one provided with --host is ignored.
# Add path for things like running kubectl binary.
export PATH=$(dirname "${e2e_test}"):"${PATH}"
"${ginkgo}" "${ginkgo_args[@]:+${ginkgo_args[@]}}" "${e2e_test}" -- \
  "${auth_config[@]:+${auth_config[@]}}" \
  --host="${KUBE_MASTER_URL}" \
  --provider="${KUBERNETES_PROVIDER}" \
  --gce-project="${PROJECT:-}" \
  --gce-zone="${ZONE:-}" \
  --gke-cluster="${CLUSTER_NAME:-}" \
  --kube-master="${KUBE_MASTER:-}" \
  --cluster-tag="${CLUSTER_ID:-}" \
  --repo-root="${KUBE_VERSION_ROOT}" \
  --node-instance-group="${NODE_INSTANCE_GROUP:-}" \
  --num-nodes="${NUM_MINIONS:-}" \
  --prefix="${KUBE_GCE_INSTANCE_PREFIX:-e2e}" \
  ${E2E_MIN_STARTUP_PODS:+"--minStartupPods=${E2E_MIN_STARTUP_PODS}"} \
  ${E2E_REPORT_DIR:+"--report-dir=${E2E_REPORT_DIR}"} \
  "${@:-}"
