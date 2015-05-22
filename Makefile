# Currently this just contains the existing docs build cmds from docker/docker
# Later, we will move this into an import
DOCS_MOUNT := $(if $(DOCSDIR),-v $(CURDIR)/$(DOCSDIR):/$(DOCSDIR))
DOCSPORT := 8000
DOCKER_DOCS_IMAGE := docker-docs-$(VERSION)
DOCKER_RUN_DOCS := docker run --rm -it $(DOCS_MOUNT) -e AWS_S3_BUCKET -e NOCACHE

docs: docs-build
	$(DOCKER_RUN_DOCS) -p $(DOCSPORT):8000 "$(DOCKER_DOCS_IMAGE)" mkdocs serve

docs-shell: docs-build
	$(DOCKER_RUN_DOCS) -p $(DOCSPORT):8000 "$(DOCKER_DOCS_IMAGE)" bash

docs-build:
	docker build -t "$(DOCKER_DOCS_IMAGE)" -f docs/Dockerfile .