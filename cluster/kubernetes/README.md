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