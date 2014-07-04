## Libswarm Makefile

## Defines a place to put downloaded dependencies in addition to acting as
## an installation sandbox for development.
workspace = $(CURDIR)/build

## Export the GOPATH to include our workspace.
export GOPATH = $(workspace)


## Init and build
all: init get


## Initializes the libswarm workspace.
init: init-dirs


## Creates workspace directories and links.
init-dirs:
	test -d $(workspace) || mkdir $(workspace)
	test -d $(workspace)/pkg || mkdir $(workspace)/pkg
	test -d $(workspace)/bin || mkdir $(workspace)/bin
	test -d $(workspace)/src || (mkdir -p $(workspace)/src/github.com/docker/ && ln -s $(CURDIR) $(workspace)/src/github.com/docker/libswarm)


## Get uses "go get" since this will sandbox the built binary to the
## bin directory of the workspace as well as grab all relevant dependencies.
get: clean-pkg
	go get github.com/docker/libswarm/swarmd


## Clean the swarmd object code
clean-pkg:
	rm -rf $(workspace)/pkg/github.com/docker/libswarm


## Remove the workspace - ezpz.
clean:
	rm -rf $(workspace)
