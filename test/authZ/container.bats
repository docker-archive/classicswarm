#!/usr/bin/env bats

setup() {
	Host='tcp://127.0.0.1:2379'
	user1='/home/stack/user1/'
	user2='/home/stack/user2/'   
}

@test "Create..." {
    docker --config $user1 -H $Host run -tid --name name1 busybox
    docker --config $user1 -H $Host run -tid --name name2 busybox
    docker --config $user2 -H $Host run -tid --name name3 busybox
}

@test "Test tenant can see his containers..." {
    run docker --config $user1 -H $Host ps -a
    echo $output
    [[ "$output" == *"name1"* ]]
    [[ "$output" == *"name2"* ]]
}

@test "Test tenant can not see other tenant's containers..." {
    run docker --config $user1 -H $Host ps -a
    echo $output
    [[ "$output" != *"name3"* ]]
}

@test "Test tenant can stop his container..." {
    run docker --config $user1 -H $Host stop name1
    echo $output
    [[ "$output" == *"name1"* ]]
    [[ "$output" != *"error"* ]]
}

@test "Test tenant can not stop other tenant's containers..." {
    run docker --config $user1 -H $Host stop name3
    echo $output
    [[ "$output" == *"Error response from daemon: Not Authorized!"* ]]
    [[ "$output" == *"Error: failed to stop containers: [name3]"* ]]
}

@test "Test tenant can start his container..." {
    run docker --config $user1 -H $Host start name1
    echo $output
    [[ "$output" == *"name1"* ]]
    [[ "$output" != *"error"* ]]
}

@test "Test tenant can not start other tenant's containers..." {
    run docker --config $user1 -H $Host start name3
    echo $output
    [[ "$output" == *"Error response from daemon: Not Authorized!"* ]]
    [[ "$output" == *"Error: failed to start containers: [name3]"* ]]
}

@test "Test tenant can pause his container..." {
    run docker --config $user1 -H $Host pause name1
    echo $output
    [[ "$output" == *"name1"* ]]
    [[ "$output" != *"error"* ]]
}

@test "Test tenant can not pause other tenant's containers..." {
    run docker --config $user1 -H $Host pause name3
    echo $output
    [[ "$output" == *"Error response from daemon: Not Authorized!"* ]]
    [[ "$output" == *"Error: failed to pause containers: [name3]"* ]]
}

@test "Test tenant can unpause his container..." {
    run docker --config $user1 -H $Host unpause name1
    echo $output
    [[ "$output" == *"name1"* ]]
    [[ "$output" != *"error"* ]]
}

@test "Test tenant can not unpause other tenant's containers..." {
    run docker --config $user1 -H $Host unpause name3
    echo $output
    [[ "$output" == *"Error response from daemon: Not Authorized!"* ]]
    [[ "$output" == *"Error: failed to unpause containers: [name3]"* ]]
}

@test "Remove..." {
    docker --config $user1 -H $Host stop name1 name2
    docker --config $user1 -H $Host rm name1 name2
    docker --config $user2 -H $Host stop name3
    docker --config $user2 -H $Host rm name3
}
