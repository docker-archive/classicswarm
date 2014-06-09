## Libswarm Makefile

## Defines a place to put downloaded dependencies in addition to acting as
## an installation sandbox for development.
workspace = $(CURDIR)/workspace

## Export the GOPATH to include our workspace.
export GOPATH = $(workspace)


## Init, build and test
all: init build


## Initializes the libswarm workspace.
init: init-dirs get-deps


## Creates workspace directories and links.
init-dirs:
	test -d $(workspace) || mkdir $(workspace)
	test -d $(workspace)/pkg || mkdir $(workspace)/pkg
	test -d $(workspace)/bin || mkdir $(workspace)/bin
	test -d $(workspace)/src || (mkdir -p $(workspace)/src/github.com/docker/ && ln -s $(CURDIR) $(workspace)/src/github.com/docker/libswarm)


## Gets dependencies listed within the following packages.
get-deps:
	go get -d github.com/docker/libswarm/backends
	go get -d github.com/docker/libswarm/beam
	go get -d github.com/docker/libswarm/dockerclient
	go get -d github.com/docker/libswarm/swarmd

## Build uses "go install" since this will sandbox the built binary to the
## bin directory of the workspace.
build:
	go install github.com/docker/libswarm/swarmd

## Remove the workspace - ezpz.
clean:
	rm -rf $(workspace)
