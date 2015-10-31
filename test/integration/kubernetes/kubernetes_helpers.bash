#!/bin/bash

load ../../helpers

KUBERNETES_IMAGE="b.kuar.io/kuar/kubernetes:1.2.0"
ETCD_IMAGE="b.gcr.io/kuar/etcd:2.2.0"

# Start kubernetes controller components and kubelet.
function start_kubernetes() {
    ETCD=$(
      docker_host run -d \
        --name etcd \
        --net=host \
        $ETCD_IMAGE \
        --advertise-client-urls http://127.0.0.1:2379
        --data-dir /var/lib/etcd
        --listen-client-urls http://127.0.0.1:2379
        --listen-peer-urls http://127.0.0.1:2380
        --name etcd0
    )

    retry 10 1 eval "docker_host ps | grep 'etcd'"

    KUBE_APISERVER=$(
      docker_host run -d \
        --name kube-apiserver \
        --net=host \
        $KUBERNETES_IMAGE kube-apiserver \
        --etcd-servers=http://127.0.0.1:2379 \
        --insecure-bind-address=127.0.0.1 \
        --insecure-port=8080 \
        --logtostderr=true \
        --service-cluster-ip-range=10.200.20.0/24 \
        --v=2
    )

    retry 10 1 eval "docker_host ps | grep 'kube-apiserver'"

    KUBE_CONTROLLER_MANAGER=$(
      docker_host run -d \
        --name kube-controller-manager \
        --net=host \
        $KUBERNETES_IMAGE kube-controller-manager \
        --logtostderr=true \
        --master=http://127.0.0.1:8080 \
        --v=2
    )

    retry 10 1 eval "docker_host ps | grep 'kube-controller-manager'"

    KUBE_SCHEDULER=$(
      docker_host run -d \
        --name=kube-scheduler \
        --net=host \
        $KUBERNETES_IMAGE kube-scheduler \
        --logtostderr=true \
        --master=http://127.0.0.1:8080 \
        --v=2
    )

    retry 10 1 eval "docker_host ps | grep 'kube-scheduler'"

    KUBELET=$(
      docker_host run -d \
        --name=kubelet \
        --net=host \
        $KUBERNETES_IMAGE kubelet \
        --address=0.0.0.0 \
        --api-servers=http://127.0.0.1:8080 \
        --containerized \
        --enable-server \
        --hostname-override=127.0.0.1 \
        --v=2
    )

    retry 10 1 eval "docker_host ps | grep 'kubelet'"
}

# Stop Kubernetes controller components and Kubelet.
function stop_kubernetes() {
    echo "Stopping $KUBE_CONTROLLER_MANAGER"
    docker_host rm -f -v $KUBE_CONTROLLER_MANAGER > /dev/null;
    echo "Stopping $KUBE_SCHEDULER"
    docker_host rm -f -v $KUBE_SCHEDULER > /dev/null;
    echo "Stopping $KUBELET"
    docker_host rm -f -v $KUBELET > /dev/null;
    echo "Stopping $KUBE_APISERVER"
    docker_host rm -f -v $KUBE_APISERVER > /dev/null;
    echo "Stopping $ETCD"
    docker_host rm -f -v $ETCD > /dev/null;
}
