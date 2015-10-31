#!/bin/bash

load ../../helpers

KUBERNETES_API_PORT=8080
KUBERNETES_IMAGE="gcr.io/google_containers/hyperkube:v1.0.6"
ETCD_IMAGE="gcr.io/google_containers/etcd:2.0.12"

# Start kubernetes controller components and kubelet.
function start_kubernetes() {
    ETCD=$(
      docker_host run -d \
        --name=etcd \
        --net=host \
        $ETCD_IMAGE /usr/local/bin/etcd \
        --addr=127.0.0.1:4001 \
        --bind-addr=0.0.0.0:4001 \
        --data-dir=/var/etcd/data
    )

    retry 10 1 eval "docker_host ps | grep 'etcd'"

    KUBERNETES=$(
      docker run -d \
        --name=kubernetes \
        --volume=/:/rootfs:ro \
        --volume=/sys:/sys:ro \
        --volume=/dev:/dev \
        --volume=/var/lib/docker/:/var/lib/docker:rw \
        --volume=/var/lib/kubelet/:/var/lib/kubelet:rw \
        --volume=/var/run:/var/run:rw \
        --net=host \
        --pid=host \
        --privileged=true \
        $KUBERNETES_IMAGE \
        /hyperkube kubelet \
        --containerized \
        --hostname-override="127.0.0.1" \
        --address="0.0.0.0" \
        --api-servers=http://127.0.0.1:8080 \
        --config=/etc/kubernetes/manifests
    )

    retry 10 1 eval "docker_host ps | grep 'kubernetes'"
}

# Stop Kubernetes controller components and Kubelet.
function stop_kubernetes() {
    echo "Stopping $KUBERNETES"
    docker_host rm -f -v $KUBERNETES > /dev/null;
    echo "Stopping $ETCD"
    docker_host rm -f -v $ETCD > /dev/null;
}
