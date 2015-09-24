.PHONY: build binary shell build_image


BUILD_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DOCKER_IMAGE := swarm-dev:$(BUILD_ID)

VOLUMES := \
	-v $(CURDIR):/go/src/github.com/docker/swarm \
	-v $(CURDIR)/dist/bin:/go/bin \
	-v $(CURDIR)/dist/pkg:/go/pkg

all: binary

build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile.build .

dist:
	mkdir dist/

binary: dist build
	docker run --rm -e VERSION=$(BUILD_ID) $(VOLUMES) $(DOCKER_IMAGE)

shell: dist build
	docker run --rm -ti $(VOLUMES) $(DOCKER_IMAGE) bash

build_image: binary
	docker build -t swarm:$(BUILD_ID) .
