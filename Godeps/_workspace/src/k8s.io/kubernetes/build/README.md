# Building Kubernetes

Building Kubernetes is easy if you take advantage of the containerized build environment. This document will help guide you through understanding this build process.

## Requirements

1. Docker, using one of the two following configurations:
  1. **Mac OS X** You can either use docker-machine or boot2docker. See installation instructions [here](https://docs.docker.com/installation/mac/).  
  * **Note**: You will want to set the boot2docker vm to have at least 3GB of initial memory or building will likely fail. (See: [#11852]( http://issue.k8s.io/11852)) and do not `make quick-release` from `/tmp/` (See: [#14773]( https://github.com/kubernetes/kubernetes/issues/14773))
  2. **Linux with local Docker**  Install Docker according to the [instructions](https://docs.docker.com/installation/#installation) for your OS.  The scripts here assume that they are using a local Docker server and that they can "reach around" docker and grab results directly from the file system.
2. [Python](https://www.python.org)
3. **Optional** [Google Cloud SDK](https://developers.google.com/cloud/sdk/)

You must install and configure Google Cloud SDK if you want to upload your release to Google Cloud Storage and may safely omit this otherwise.

## Overview

While it is possible to build Kubernetes using a local golang installation, we have a build process that runs in a Docker container.  This simplifies initial set up and provides for a very consistent build and test environment.

There is also early support for building Docker "run" containers

## Key scripts

* `run.sh`: Run a command in a build docker container.  Common invocations:
  *  `run.sh hack/build-go.sh`: Build just linux binaries in the container.  Pass options and packages as necessary.
  *  `run.sh hack/build-cross.sh`: Build all binaries for all platforms
  *  `run.sh hack/test-go.sh`: Run all unit tests
  *  `run.sh hack/test-integration.sh`: Run integration test
* `copy-output.sh`: This will copy the contents of `_output/dockerized/bin` from any remote Docker container to the local `_output/dockerized/bin`.  Right now this is only necessary on Mac OS X with `boot2docker` when your git repo isn't under `/Users`.
* `make-clean.sh`: Clean out the contents of `_output/dockerized` and remove any local built container images.
* `shell.sh`: Drop into a `bash` shell in a build container with a snapshot of the current repo code.
* `release.sh`: Build everything, test it, and (optionally) upload the results to a GCS bucket.

## Releasing

The `release.sh` script will build a release.  It will build binaries, run tests, (optionally) build runtime Docker images and then (optionally) upload all build artifacts to a GCS bucket.

The main output is a tar file: `kubernetes.tar.gz`.  This includes:
* Cross compiled client utilities.
* Script (`kubectl`) for picking and running the right client binary based on platform.
* Examples
* Cluster deployment scripts for various clouds
* Tar file containing all server binaries
* Tar file containing salt deployment tree shared across multiple cloud deployments.

In addition, there are some other tar files that are created:
* `kubernetes-client-*.tar.gz` Client binaries for a specific platform.
* `kubernetes-server-*.tar.gz` Server binaries for a specific platform.
* `kubernetes-salt.tar.gz` The salt script/tree shared across multiple deployment scripts.

The release utilities grab a set of environment variables to modify behavior.  Arguably, these should be command line flags:

Env Variable | Default | Description
-------------|---------|------------
`KUBE_SKIP_CONFIRMATIONS` | `n` | If `y` then no questions are asked and the scripts just continue.
`KUBE_GCS_UPLOAD_RELEASE` | `n` | Upload release artifacts to GCS
`KUBE_GCS_RELEASE_BUCKET` | `kubernetes-releases-${project_hash}` | The bucket to upload releases to
`KUBE_GCS_RELEASE_PREFIX` | `devel` | The path under the release bucket to put releases
`KUBE_GCS_MAKE_PUBLIC` | `y` | Make GCS links readable from anywhere
`KUBE_GCS_NO_CACHING` | `y` | Disable HTTP caching of GCS release artifacts.  By default GCS will cache public objects for up to an hour.  When doing "devel" releases this can cause problems.
`KUBE_GCS_DOCKER_REG_PREFIX` | `docker-reg` | *Experimental* When uploading docker images, the bucket that backs the registry.

## Basic Flow

The scripts directly under `build/` are used to build and test.  They will ensure that the `kube-build` Docker image is built (based on `build/build-image/Dockerfile`) and then execute the appropriate command in that container.  If necessary (for Mac OS X), the scripts will also copy results out.

The `kube-build` container image is built by first creating a "context" directory in `_output/images/build-image`.  It is done there instead of at the root of the Kubernetes repo to minimize the amount of data we need to package up when building the image.

Everything in `build/build-image/` is meant to be run inside of the container.  If it doesn't think it is running in the container it'll throw a warning.  While you can run some of that stuff outside of the container, it wasn't built to do so.

When building final release tars, they are first staged into `_output/release-stage` before being tar'd up and put into `_output/release-tars`.

## Proxy Settings


If you are behind a proxy, you need to edit `build/build-image/Dockerfile` and add proxy settings to execute command in that container correctly.

example:

`ENV http_proxy http://username:password@proxyaddr:proxyport`

`ENV https_proxy http://username:password@proxyaddr:proxyport`

Besides, to avoid integration test touch the proxy while connecting to local etcd service, you need to set

`ENV no_proxy 127.0.0.1`

## TODOs

These are in no particular order

* [X] Harmonize with scripts in `hack/`.  How much do we support building outside of Docker and these scripts?
* [X] Deprecate/replace most of the stuff in the hack/
* [ ] Finish support for the Dockerized runtime. Issue [#19](http://issue.k8s.io/19).  A key issue here is to make this fast/light enough that we can use it for development workflows.


[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/build/README.md?pixel)]()
