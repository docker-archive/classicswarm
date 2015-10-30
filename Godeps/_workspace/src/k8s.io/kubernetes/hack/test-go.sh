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

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

kube::golang::setup_env

kube::test::find_dirs() {
  (
    cd ${KUBE_ROOT}
    find . -not \( \
        \( \
          -path './_artifacts/*' \
          -o -path './_output/*' \
          -o -path './_gopath/*' \
          -o -path './Godeps/*' \
          -o -path './contrib/podex/*' \
          -o -path './output/*' \
          -o -path './release/*' \
          -o -path './target/*' \
          -o -path './test/e2e/*' \
          -o -path './test/integration/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | sed 's|^\./||' | sort -u
  )
}

# -covermode=atomic becomes default with -race in Go >=1.3
KUBE_TIMEOUT=${KUBE_TIMEOUT:--timeout 120s}
KUBE_COVER=${KUBE_COVER:-n} # set to 'y' to enable coverage collection
KUBE_COVERMODE=${KUBE_COVERMODE:-atomic}
# How many 'go test' instances to run simultaneously when running tests in
# coverage mode.
KUBE_COVERPROCS=${KUBE_COVERPROCS:-4}
KUBE_RACE=${KUBE_RACE:-}   # use KUBE_RACE="-race" to enable race testing
# Set to the goveralls binary path to report coverage results to Coveralls.io.
KUBE_GOVERALLS_BIN=${KUBE_GOVERALLS_BIN:-}
# Lists of API Versions of each groups that should be tested, groups are
# separated by comma, lists are separated by semicolon. e.g.,
# "v1,compute/v1alpha1,experimental/v1alpha2;v1,compute/v2,experimental/v1alpha3"
# TODO: It's going to be:
# KUBE_TEST_API_VERSIONS=${KUBE_TEST_API_VERSIONS:-"v1,experimental/v1alpha1"}
KUBE_TEST_API_VERSIONS=${KUBE_TEST_API_VERSIONS:-"v1,experimental/v1alpha1"}
# once we have multiple group supports
# Run tests with the standard (registry) and a custom etcd prefix
# (kubernetes.io/registry).
KUBE_TEST_ETCD_PREFIXES=${KUBE_TEST_ETCD_PREFIXES:-"registry,kubernetes.io/registry"}
# Create a junit-style XML test report in this directory if set.
KUBE_JUNIT_REPORT_DIR=${KUBE_JUNIT_REPORT_DIR:-}

kube::test::usage() {
  kube::log::usage_from_stdin <<EOF
usage: $0 [OPTIONS] [TARGETS]

OPTIONS:
  -i <number>   : number of times to run each test, must be >= 1
EOF
}

isnum() {
  [[ "$1" =~ ^[0-9]+$ ]]
}

iterations=1
while getopts "hi:" opt ; do
  case $opt in
    h)
      kube::test::usage
      exit 0
      ;;
    i)
      iterations="$OPTARG"
      if ! isnum "${iterations}" || [[ "${iterations}" -le 0 ]]; then
        kube::log::usage "'$0': argument to -i must be numeric and greater than 0"
        kube::test::usage
        exit 1
      fi
      ;;
    ?)
      kube::test::usage
      exit 1
      ;;
    :)
      kube::log::usage "Option -$OPTARG <value>"
      kube::test::usage
      exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

# Use eval to preserve embedded quoted strings.
eval "goflags=(${KUBE_GOFLAGS:-})"
eval "testargs=(${KUBE_TEST_ARGS:-})"

# Used to filter verbose test output.
go_test_grep_pattern=".*"

# The go-junit-report tool needs full test case information to produce a
# meaningful report.
if [[ -n "${KUBE_JUNIT_REPORT_DIR}" ]] ; then
  goflags+=(-v)
  # Show only summary lines by matching lines like "status package/test"
  go_test_grep_pattern="^[^[:space:]]\+[[:space:]]\+[^[:space:]]\+/[^[[:space:]]\+"
fi

# Filter out arguments that start with "-" and move them to goflags.
testcases=()
for arg; do
  if [[ "${arg}" == -* ]]; then
    goflags+=("${arg}")
  else
    testcases+=("${arg}")
  fi
done
if [[ ${#testcases[@]} -eq 0 ]]; then
  testcases=($(kube::test::find_dirs))
fi
set -- "${testcases[@]+${testcases[@]}}"

junitFilenamePrefix() {
  if [[ -z "${KUBE_JUNIT_REPORT_DIR}" ]]; then
    echo ""
    return
  fi
  mkdir -p "${KUBE_JUNIT_REPORT_DIR}"
  local KUBE_TEST_API_NO_SLASH="${KUBE_TEST_API//\//-}"
  echo "${KUBE_JUNIT_REPORT_DIR}/junit_${KUBE_TEST_API_NO_SLASH}_$(kube::util::sortable_date)"
}

produceJUnitXMLReport() {
  local -r junit_filename_prefix=$1
  if [[ -z "${junit_filename_prefix}" ]]; then
    return
  fi

  local test_stdout_filenames
  local junit_xml_filename
  test_stdout_filenames=$(ls ${junit_filename_prefix}*.stdout)
  junit_xml_filename="${junit_filename_prefix}.xml"
  if ! command -v go-junit-report >/dev/null 2>&1; then
    kube::log::error "go-junit-report not found; please install with " \
      "go get -u github.com/jstemmer/go-junit-report"
    return
  fi
  cat ${test_stdout_filenames} | go-junit-report > "${junit_xml_filename}"
  rm ${test_stdout_filenames}
  kube::log::status "Saved JUnit XML test report to ${junit_xml_filename}"
}

runTests() {
  # TODO: this should probably be refactored to avoid code duplication with the
  # coverage version.
  if [[ $iterations -gt 1 ]]; then
    if [[ $# -eq 0 ]]; then
      set -- $(kube::test::find_dirs)
    fi
    kube::log::status "Running ${iterations} times"
    fails=0
    for arg; do
      trap 'exit 1' SIGINT
      pkg=${KUBE_GO_PACKAGE}/${arg}
      kube::log::status "${pkg}"
      # keep going, even if there are failures
      pass=0
      count=0
      for i in $(seq 1 ${iterations}); do
        if go test "${goflags[@]:+${goflags[@]}}" \
            ${KUBE_RACE} ${KUBE_TIMEOUT} "${pkg}" \
            "${testargs[@]:+${testargs[@]}}"; then
          pass=$((pass + 1))
        else
          fails=$((fails + 1))
        fi
        count=$((count + 1))
      done 2>&1
      kube::log::status "${pass} / ${count} passed"
    done
    if [[ ${fails} -gt 0 ]]; then
      return 1
    else
      return 0
    fi
  fi

  local junit_filename_prefix
  junit_filename_prefix=$(junitFilenamePrefix)

  # If we're not collecting coverage, run all requested tests with one 'go test'
  # command, which is much faster.
  if [[ ! ${KUBE_COVER} =~ ^[yY]$ ]]; then
    kube::log::status "Running tests without code coverage"
    go test "${goflags[@]:+${goflags[@]}}" \
      ${KUBE_RACE} ${KUBE_TIMEOUT} "${@+${@/#/${KUBE_GO_PACKAGE}/}}" \
     "${testargs[@]:+${testargs[@]}}" \
     | tee ${junit_filename_prefix:+"${junit_filename_prefix}.stdout"} \
     | grep "${go_test_grep_pattern}" && rc=$? || rc=$?
    produceJUnitXMLReport "${junit_filename_prefix}"
    return ${rc}
  fi

  # Create coverage report directories.
  cover_report_dir="/tmp/k8s_coverage/${KUBE_TEST_API}/$(kube::util::sortable_date)"
  cover_profile="coverage.out"  # Name for each individual coverage profile
  kube::log::status "Saving coverage output in '${cover_report_dir}'"
  mkdir -p "${@+${@/#/${cover_report_dir}/}}"

  # Run all specified tests, collecting coverage results. Go currently doesn't
  # support collecting coverage across multiple packages at once, so we must issue
  # separate 'go test' commands for each package and then combine at the end.
  # To speed things up considerably, we can at least use xargs -P to run multiple
  # 'go test' commands at once.
  # To properly parse the test results if generating a JUnit test report, we
  # must make sure the output from parallel runs is not mixed. To achieve this,
  # we spawn a subshell for each parallel process, redirecting the output to
  # separate files.
  printf "%s\n" "${@}" | xargs -I{} -n1 -P${KUBE_COVERPROCS} \
    bash -c "set -o pipefail; _pkg=\"{}\"; _pkg_out=\${_pkg//\//_}; \
        go test ${goflags[@]:+${goflags[@]}} \
          ${KUBE_RACE} \
          ${KUBE_TIMEOUT} \
          -cover -covermode=\"${KUBE_COVERMODE}\" \
          -coverprofile=\"${cover_report_dir}/\${_pkg}/${cover_profile}\" \
          \"${KUBE_GO_PACKAGE}/\${_pkg}\" \
          ${testargs[@]:+${testargs[@]}} \
        | tee ${junit_filename_prefix:+\"${junit_filename_prefix}-\$_pkg_out.stdout\"} \
        | grep \"${go_test_grep_pattern}\"" \
      && test_result=$? || test_result=$?

  produceJUnitXMLReport "${junit_filename_prefix}"

  COMBINED_COVER_PROFILE="${cover_report_dir}/combined-coverage.out"
  {
    # The combined coverage profile needs to start with a line indicating which
    # coverage mode was used (set, count, or atomic). This line is included in
    # each of the coverage profiles generated when running 'go test -cover', but
    # we strip these lines out when combining so that there's only one.
    echo "mode: ${KUBE_COVERMODE}"

    # Include all coverage reach data in the combined profile, but exclude the
    # 'mode' lines, as there should be only one.
    for x in `find "${cover_report_dir}" -name "${cover_profile}"`; do
      cat $x | grep -h -v "^mode:" || true
    done
  } >"${COMBINED_COVER_PROFILE}"

  coverage_html_file="${cover_report_dir}/combined-coverage.html"
  go tool cover -html="${COMBINED_COVER_PROFILE}" -o="${coverage_html_file}"
  kube::log::status "Combined coverage report: ${coverage_html_file}"

  return ${test_result}
}

reportCoverageToCoveralls() {
  if [[ ${KUBE_COVER} =~ ^[yY]$ ]] && [[ -x "${KUBE_GOVERALLS_BIN}" ]]; then
    kube::log::status "Reporting coverage results to Coveralls for service ${CI_NAME:-}"
    ${KUBE_GOVERALLS_BIN} -coverprofile="${COMBINED_COVER_PROFILE}" \
    ${CI_NAME:+"-service=${CI_NAME}"} \
    ${COVERALLS_REPO_TOKEN:+"-repotoken=${COVERALLS_REPO_TOKEN}"} \
      || true
  fi
}

# Convert the CSVs to arrays.
IFS=';' read -a apiVersions <<< "${KUBE_TEST_API_VERSIONS}"
IFS=',' read -a etcdPrefixes <<< "${KUBE_TEST_ETCD_PREFIXES}"
apiVersionsCount=${#apiVersions[@]}
etcdPrefixesCount=${#etcdPrefixes[@]}
for (( i=0, j=0; ; )); do
  apiVersion=${apiVersions[i]}
  etcdPrefix=${etcdPrefixes[j]}
  echo "Running tests for APIVersion: $apiVersion with etcdPrefix: $etcdPrefix"
  # KUBE_TEST_API sets the version of each group to be tested. KUBE_API_VERSIONS
  # register the groups/versions as supported by k8s. So KUBE_API_VERSIONS
  # needs to be the superset of KUBE_TEST_API.
  KUBE_TEST_API="${apiVersion}" KUBE_API_VERSIONS="v1,experimental/v1alpha1" ETCD_PREFIX=${etcdPrefix} runTests "$@"
  i=${i}+1
  j=${j}+1
  if [[ i -eq ${apiVersionsCount} ]] && [[ j -eq ${etcdPrefixesCount} ]]; then
    # All api versions and etcd prefixes tested.
    break
  fi
  if [[ i -eq ${apiVersionsCount} ]]; then
    # Use the last api version for remaining etcd prefixes.
    i=${i}-1
  fi
   if [[ j -eq ${etcdPrefixesCount} ]]; then
     # Use the last etcd prefix for remaining api versions.
    j=${j}-1
  fi
done

# We might run the tests for multiple versions, but we want to report only
# one of them to coveralls. Here we report coverage from the last run.
reportCoverageToCoveralls
