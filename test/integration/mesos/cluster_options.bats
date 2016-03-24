#!/usr/bin/env bats

load ../../helpers
load ../mesos_helpers

function teardown() {
	swarm_manage_cleanup
	stop_mesos
	stop_docker
}

@test "swarm - cluster options" {
	start_docker 2
	start_mesos

	# mesos.port in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.port=123asd 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Failed to parse mesos.port in Uint"* ]]

	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.port=0 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"mesos.port cannot be 0"* ]]

	# mesos.address in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.address=127.0.0.1oooo 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"invalid IP address for cluster-opt mesos.address:"* ]]

	# mesos.checkpointfailover in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.checkpointfailover=123asd 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Failed to parse mesos.checkpointfailover in Bool"* ]]

	# mesos.tasktimeout in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.tasktimeout=123asd 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Failed to parse mesos.tasktimeout in Duration"* ]]

	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.tasktimeout=-1 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"mesos.taskCreationTimeout cannot be a negative number"* ]]

	# mesos.offertimeout in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.offertimeout=123asd 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Failed to parse mesos.offertimeout in Duration"* ]]

	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.offertimeout=-1 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"mesos.offerTimeout cannot be a negative number"* ]]

	# mesos.offerrefusetimeout in cluster-opt
	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.offerrefusetimeout=123asd 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"Failed to parse mesos.offerrefusetimeout in Duration"* ]]

	run swarm manage --cluster-driver mesos-experimental --cluster-opt mesos.offerrefusetimeout=-1 127.0.0.1:$MESOS_MASTER_PORT
	[ "$status" -ne 0 ]
	[[ "${output}" == *"mesos.offerrefusetimeout cannot be a negative number"* ]]
}
