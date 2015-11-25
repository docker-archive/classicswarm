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

#### HACK ####
# Sometimes godep just can't handle things. This lets use manually put
# some deps in place first, so godep won't fall over.
preload-dep() {
  org="$1"
  project="$2"
  sha="$3"

  org_dir="${GOPATH}/src/${org}"
  mkdir -p "${org_dir}"
  pushd "${org_dir}" > /dev/null
    git clone "https://${org}/${project}.git" > /dev/null 2>&1
    pushd "${org_dir}/${project}" > /dev/null
      git checkout "${sha}" > /dev/null 2>&1
    popd > /dev/null
  popd > /dev/null
}

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

branch="${1:-master}"
# notice this uses ... to find the first shared ancestor
if ! git diff origin/"${branch}"...HEAD | grep 'Godeps/' > /dev/null; then
  exit 0
fi

# create a nice clean place to put our new godeps
_tmpdir="$(mktemp -d -t gopath.XXXXXX)"
function cleanup {
  echo "Removing ${_tmpdir}"
  rm -rf "${_tmpdir}"
}
trap cleanup EXIT

# build the godep tool
export GOPATH="${_tmpdir}"
go get -u github.com/tools/godep 2>/dev/null
go install github.com/tools/godep 2>/dev/null
GODEP="${_tmpdir}/bin/godep"

# fill out that nice clean place with the kube godeps
echo "Starting to download all kubernetes godeps. This takes a while"

# github.com/prometheus/client_golang removed the model and extraction directory.
# thus go get fails and thus godep fails. So load it by hand.
preload-dep "github.com/prometheus" "client_golang" "692492e54b553a81013254cc1fba4b6dd76fad30"

"${GODEP}" restore
echo "Download finished"

# copy the contents of your kube directory into the nice clean place
_kubetmp="${_tmpdir}/src/k8s.io"
mkdir -p "${_kubetmp}"
#should create ${_kubectmp}/kubernetes
git archive --format=tar --prefix=kubernetes/ $(git write-tree) | (cd "${_kubetmp}" && tar xf -)
_kubetmp="${_kubetmp}/kubernetes"

# destroy godeps in our COPY of the kube tree
pushd "${_kubetmp}" > /dev/null
  rm -rf ./Godeps

  # for some reason the kube tree needs to be a git repo for the godep tool to run. Doesn't make sense
  git init > /dev/null 2>&1

  # recreate the Godeps using the nice clean set we just downloaded
  "${GODEP}" save -t ./...
popd > /dev/null

if ! _out="$(diff -Naupr --ignore-matching-lines='^\s*\"GoVersion\":' --ignore-matching-lines='^\s*\"Comment\":' ${KUBE_ROOT}/Godeps/Godeps.json ${_kubetmp}/Godeps/Godeps.json)"; then
  echo "Your Godeps.json is different:"
  echo "${_out}"
  exit 1
fi

# Godeps/_workstapces/src/github.com/fsouza/go-dockerclient/testing/data/symlink'
# is an intentionally broken symlink. Linux can use --no-dereference. OS X cannot.
# So we --exclude='symlink' so diff -r doesn't die following a bad symlink.
if ! _out="$(diff -Naupr --exclude='symlink' ${KUBE_ROOT}/Godeps/_workspace/src ${_kubetmp}/Godeps/_workspace/src)"; then
  echo "Your godeps changes are not reproducable"
  echo "${_out}"
  exit 1
fi

# ex: ts=2 sw=2 et filetype=sh
