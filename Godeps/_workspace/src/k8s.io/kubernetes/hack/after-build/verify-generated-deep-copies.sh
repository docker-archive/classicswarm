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

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/../..
source "${KUBE_ROOT}/hack/lib/init.sh"

kube::golang::setup_env

gendeepcopy=$(kube::util::find-binary "gendeepcopy")

APIROOTS=${APIROOTS:-pkg/api pkg/apis/experimental}
_tmp="${KUBE_ROOT}/_tmp"

cleanup() {
	rm -rf "${_tmp}"
}

trap "cleanup" EXIT SIGINT

for APIROOT in ${APIROOTS}; do
	mkdir -p "${_tmp}/${APIROOT%/*}"
	cp -a "${KUBE_ROOT}/${APIROOT}" "${_tmp}/${APIROOT}"
done

"${KUBE_ROOT}/hack/after-build/update-generated-deep-copies.sh"

for APIROOT in ${APIROOTS}; do
	TMP_APIROOT="${_tmp}/${APIROOT}"
	echo "diffing ${APIROOT} against freshly generated deep copies"
	ret=0
	diff -Naupr -I 'Auto generated by' "${KUBE_ROOT}/${APIROOT}" "${TMP_APIROOT}" || ret=$?
	cp -a ${TMP_APIROOT} "${KUBE_ROOT}/${APIROOT%/*}"
	if [[ $ret -eq 0 ]]; then
		echo "${APIROOT} up to date."
	else
		echo "${APIROOT} is out of date. Please run hack/update-generated-deep-copies.sh"
		exit 1
	fi
done
