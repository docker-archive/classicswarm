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

# A set of helpers for tests

readonly reset=$(tput sgr0)
readonly  bold=$(tput bold)
readonly black=$(tput setaf 0)
readonly   red=$(tput setaf 1)
readonly green=$(tput setaf 2)

kube::test::clear_all() {
  kubectl delete "${kube_flags[@]}" rc,pods --all --grace-period=0
}

kube::test::get_object_assert() {
  local object=$1
  local request=$2
  local expected=$3

  res=$(eval kubectl get "${kube_flags[@]}" $object -o go-template=\"$request\")

  if [[ "$res" =~ ^$expected$ ]]; then
      echo -n ${green}
      echo "Successful get $object $request: $res"
      echo -n ${reset}
      return 0
  else
      echo ${bold}${red}
      echo "FAIL!"
      echo "Get $object $request"
      echo "  Expected: $expected"
      echo "  Got:      $res"
      echo ${reset}${red}
      caller
      echo ${reset}
      return 1
  fi
}

kube::test::get_object_jsonpath_assert() {
  local object=$1
  local request=$2
  local expected=$3

  res=$(eval kubectl get "${kube_flags[@]}" $object -o jsonpath=\"$request\")

  if [[ "$res" =~ ^$expected$ ]]; then
      echo -n ${green}
      echo "Successful get $object $request: $res"
      echo -n ${reset}
      return 0
  else
      echo ${bold}${red}
      echo "FAIL!"
      echo "Get $object $request"
      echo "  Expected: $expected"
      echo "  Got:      $res"
      echo ${reset}${red}
      caller
      echo ${reset}
      return 1
  fi
}

kube::test::describe_object_assert() {
  local resource=$1
  local object=$2
  local matches=${@:3}

  result=$(eval kubectl describe "${kube_flags[@]}" $resource $object)

  for match in ${matches}; do
    if [[ ! $(echo "$result" | grep ${match}) ]]; then
      echo ${bold}${red}
      echo "FAIL!"
      echo "Describe $resource $object"
      echo "  Expected Match: $match"
      echo "  Not found in:"
      echo "$result"
      echo ${reset}${red}
      caller
      echo ${reset}
      return 1
    fi
  done

  echo -n ${green}
  echo "Successful describe $resource $object:"
  echo "$result"
  echo -n ${reset}
  return 0
}

kube::test::describe_resource_assert() {
  local resource=$1
  local matches=${@:2}

  result=$(eval kubectl describe "${kube_flags[@]}" $resource)

  for match in ${matches}; do
    if [[ ! $(echo "$result" | grep ${match}) ]]; then
      echo ${bold}${red}
      echo "FAIL!"
      echo "Describe $resource"
      echo "  Expected Match: $match"
      echo "  Not found in:"
      echo "$result"
      echo ${reset}${red}
      caller
      echo ${reset}
      return 1
    fi
  done

  echo -n ${green}
  echo "Successful describe $resource:"
  echo "$result"
  echo -n ${reset}
  return 0
}

kube::test::if_has_string() {
  local message=$1
  local match=$2

  if [[ $(echo "$message" | grep "$match") ]]; then
    echo "Successful"
    echo "message:$message"
    echo "has:$match"
    return 0
  else
    echo "FAIL!"
    echo "message:$message"
    echo "has not:$match"
    caller
    return 1
  fi
}
