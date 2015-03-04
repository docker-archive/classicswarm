#!/bin/bash

SWARM_ROOT=${BATS_TEST_DIRNAME}/../..

function swarm() {
	${SWARM_ROOT}/swarm $@
}
