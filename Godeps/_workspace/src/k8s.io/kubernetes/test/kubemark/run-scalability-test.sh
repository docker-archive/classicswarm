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

export KUBERNETES_PROVIDER="kubemark"
export KUBE_CONFIG_FILE="config-default.sh"

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/../..

# We need an absolute path to KUBE_ROOT
ABSOLUTE_ROOT=$(readlink -f ${KUBE_ROOT})

source ${KUBE_ROOT}/cluster/kubemark/util.sh
source ${KUBE_ROOT}/cluster/kubemark/config-default.sh

echo ${KUBERNETES_PROVIDER}
echo ${MASTER_NAME}

detect-master

export KUBE_MASTER_URL="https://${KUBE_MASTER_IP}"
export KUBECONFIG="${ABSOLUTE_ROOT}/test/kubemark/kubeconfig.loc"
export E2E_MIN_STARTUP_PODS=0

${KUBE_ROOT}/hack/ginkgo-e2e.sh --e2e-verify-service-account=false --ginkgo.focus="should\sallow\sstarting\s30\spods\sper\snode"
