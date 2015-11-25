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

kube::util::sortable_date() {
  date "+%Y%m%d-%H%M%S"
}

# this mimics the behavior of linux realpath which is not shipped by default with
# mac OS X 
kube::util::realpath() {
  [[ $1 = /* ]] && echo "$1" | sed 's/\/$//' || echo "$PWD/${1#./}"  | sed 's/\/$//'
}

kube::util::wait_for_url() {
  local url=$1
  local prefix=${2:-}
  local wait=${3:-0.5}
  local times=${4:-25}

  which curl >/dev/null || {
    kube::log::usage "curl must be installed"
    exit 1
  }

  local i
  for i in $(seq 1 $times); do
    local out
    if out=$(curl -fs $url 2>/dev/null); then
      kube::log::status "On try ${i}, ${prefix}: ${out}"
      return 0
    fi
    sleep ${wait}
  done
  kube::log::error "Timed out waiting for ${prefix} to answer at ${url}; tried ${times} waiting ${wait} between each"
  return 1
}

# Example:  kube::util::trap_add 'echo "in trap DEBUG"' DEBUG
# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
kube::util::trap_add() {
  local trap_add_cmd
  trap_add_cmd=$1
  shift

  for trap_add_name in "$@"; do
    local existing_cmd
    local new_cmd

    # Grab the currently defined trap commands for this trap
    existing_cmd=`trap -p "${trap_add_name}" |  awk -F"'" '{print $2}'`

    if [[ -z "${existing_cmd}" ]]; then
      new_cmd="${trap_add_cmd}"
    else
      new_cmd="${existing_cmd};${trap_add_cmd}"
    fi

    # Assign the test
    trap "${new_cmd}" "${trap_add_name}"
  done
}

# Opposite of kube::util::ensure-temp-dir()
kube::util::cleanup-temp-dir() {
  rm -rf "${KUBE_TEMP}"
}

# Create a temp dir that'll be deleted at the end of this bash session.
#
# Vars set:
#   KUBE_TEMP
kube::util::ensure-temp-dir() {
  if [[ -z ${KUBE_TEMP-} ]]; then
    KUBE_TEMP=$(mktemp -d 2>/dev/null || mktemp -d -t kubernetes.XXXXXX)
    kube::util::trap_add kube::util::cleanup-temp-dir EXIT
  fi
}

# This figures out the host platform without relying on golang.  We need this as
# we don't want a golang install to be a prerequisite to building yet we need
# this info to figure out where the final binaries are placed.
kube::util::host_platform() {
  local host_os
  local host_arch
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      kube::log::error "Unsupported host OS.  Must be Linux or Mac OS X."
      exit 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    arm*)
      host_arch=arm
      ;;
    i?86*)
      host_arch=x86
      ;;
    *)
      kube::log::error "Unsupported host arch. Must be x86_64, 386 or arm."
      exit 1
      ;;
  esac
  echo "${host_os}/${host_arch}"
}

kube::util::find-binary() {
  local lookfor="${1}"
  local host_platform="$(kube::util::host_platform)"
  local locations=(
    "${KUBE_ROOT}/_output/dockerized/bin/${host_platform}/${lookfor}"
    "${KUBE_ROOT}/_output/local/bin/${host_platform}/${lookfor}"
    "${KUBE_ROOT}/platforms/${host_platform}/${lookfor}"
  )
  local bin=$( (ls -t "${locations[@]}" 2>/dev/null || true) | head -1 )
  echo -n "${bin}"
}

# Wait for background jobs to finish. Return with
# an error status if any of the jobs failed.
kube::util::wait-for-jobs() {
  local fail=0
  local job
  for job in $(jobs -p); do
    wait "${job}" || fail=$((fail + 1))
  done
  return ${fail}
}

# Run all known doc generators (today gendocs, genman, and genbashcomp for kubectl)
# $1 is the directory to put those generated documents
kube::util::gen-docs() {
  local dest="$1"

  # Find binary
  gendocs=$(kube::util::find-binary "gendocs")
  genman=$(kube::util::find-binary "genman")
  genbashcomp=$(kube::util::find-binary "genbashcomp")

  mkdir -p "${dest}/docs/user-guide/kubectl/"
  "${gendocs}" "${dest}/docs/user-guide/kubectl/"
  mkdir -p "${dest}/docs/man/man1/"
  "${genman}" "${dest}/docs/man/man1/"
  mkdir -p "${dest}/contrib/completions/bash/"
  "${genbashcomp}" "${dest}/contrib/completions/bash/"

  # create the list of generated files
  pushd "${dest}" > /dev/null
  touch .generated_docs
  find . -type f | cut -sd / -f 2- | LC_ALL=C sort > .generated_docs
  popd > /dev/null

  while read file; do
    # Copy out of KUBE_ROOT if we didn't really change anything
    if [[ -e "${dest}/${file}" && -e "${KUBE_ROOT}/${file}" ]]; then
      # Filter all munges from original content.
      original=$(cat "${KUBE_ROOT}/${file}" | sed '/^<!-- BEGIN MUNGE:.*/,/^<!-- END MUNGE:.*/d')
      generated=$(cat "${dest}/${file}")

      # Filter out meaningless lines with timestamps
      original=$(echo "${original}" | grep -v "Auto generated by spf13/cobra" || :)
      generated=$(echo "${generated}" | grep -v "Auto generated by spf13/cobra" || :)

      # By now, the contents should be normalized and stripped of any
      # auto-managed content.  We also ignore whitespace here because of
      # markdown strictness fixups later in the pipeline.
      if diff -Bw >/dev/null <(echo "${original}") <(echo "${generated}"); then
        # actual contents same, overwrite generated with original.
        cp "${KUBE_ROOT}/${file}" "${dest}/${file}"
      fi
    fi
  done <"${KUBE_ROOT}/.generated_docs"
}

# Takes a path $1 to traverse for md files to append the ga-beacon tracking
# link to, if needed. If $2 is set, just print files that are missing
# the link.
kube::util::gen-analytics() {
  local path="$1"
  local dryrun="${2:-}"
  local mdfiles dir link
  # find has some strange inconsistencies between darwin/linux. The
  # path to search must end in '/' for linux, but darwin will put an extra
  # slash in results if there is a trailing '/'.
  if [[ $( uname ) == 'Linux' ]]; then
    dir="${path}/"
  else
    dir="${path}"
  fi
  # We don't touch files in Godeps|third_party, and the kubectl
  # docs are autogenerated by gendocs.
  mdfiles=($( find "${dir}" -name "*.md" -type f \
              -not -path "${path}/Godeps/*" \
              -not -path "${path}/third_party/*" \
              -not -path "${path}/_output/*" \
              -not -path "${path}/docs/user-guide/kubectl/kubectl*" ))
  for f in "${mdfiles[@]}"; do
    link=$(kube::util::analytics-link "${f#${path}/}")
    if grep -q -F -x "${link}" "${f}"; then
      continue
    elif [[ -z "${dryrun}" ]]; then
      echo -e "\n\n${link}" >> "${f}"
    else
      echo "$f"
    fi
  done
}

# Prints analytics link to append to a file at path $1.
kube::util::analytics-link() {
  local path="$1"
  echo "[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/${path}?pixel)]()"
}

# Takes a group/version and returns the path to its location on disk, sans
# "pkg". E.g.:
# * default behavior: experimental/v1alpha1 -> apis/experimental/v1alpha1
# * default behavior for only a group: experimental -> apis/experimental
# * Special handling for empty group: v1 -> api/v1, unversioned -> api/unversioned
# * Very special handling for when both group and version are "": / -> api
kube::util::group-version-to-pkg-path() {
  local group_version="$1"
  # Special cases first.
  # TODO(lavalamp): Simplify this by moving pkg/api/v1 and splitting pkg/api,
  # moving the results to pkg/apis/api.
  case "${group_version}" in
    # both group and version are "", this occurs when we generate deep copies for internal objects of the legacy v1 API.
    /)
      echo "api"
      ;;
    v1)
      echo "api/v1"
      ;;
    unversioned)
      echo "api/unversioned"
      ;;
    *)
      echo "apis/${group_version}"
      ;;
  esac
}

# ex: ts=2 sw=2 et filetype=sh
