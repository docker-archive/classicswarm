# Using Docker Swarm and Kubernetes

## Deploying Docker Swarm on Kubernetes

### Create the Docker Swarm Replication Controller

In the following example the Docker Swarm pod will be managed using a Kubernetes
replication controller to ensure it's always running somewhere in the cluster.

```
kubectl create -f swarm-rc.yaml
```

The Swarm Kubernetes driver leverages the default service account to locate and authenticate
with the Kubernetes API.

### Create the Docker Swarm Service

The following command will create a Kubernetes service for the Docker Swarm pod.

```
kubectl create -f swarm-svc.yaml
```

### Expose The Docker Endpoints

Docker swarms works by communicating with the Docker engine API. This means we must expose the
Docker API across all Kubernetes worker nodes.

```
--host=unix:///var/run/docker.sock
--host=tcp://0.0.0.0:2375
```

## Cluster Options

The Kubernetes cluster backend supports the following options:

- kubernetes.apiversion - The API Version to use.
- kubernetes.certificateauthority - The path to a cert file for the certificate authority. 
- kubernetes.clientcertificate - The path to a client cert file for TLS.
- kubernetes.clientkey - The path to a client key file for TLS.
- kubernetes.insecureskiptlsverify - Skips the validity check for the server's certificate.
- kubernetes.password - The password for basic authentication to the kubernetes cluster.
- kubernetes.token - The bearer token for authentication to the kubernetes cluster.
- kubernetes.username - The username for basic authentication to the kubernetes cluster.

### Example Usage

```
$ docker run -d -p <swarm_port>:2375 -p 3375:3375 \
  swarm manage \
  -c kubernetes-experimental \
  --cluster-opt kubernetes.apiversion=v1 \
  --cluster-opt kubernetes.username=swarm \
  --cluster-opt kubernetes.password=swarmPassword \
  127.0.0.1:8080
```