FROM docs/base:latest
MAINTAINER Sven Dowideit <SvenDowideit@docker.com> (@SvenDowideit)

# to get the git info for this repo
COPY . /src

# Reset the /docs dir so we can replace the theme meta with the new repo's git info
RUN git reset --hard

RUN grep "VERSION =" /src/version/version.go  | sed 's/.*"\(.*\)".*/\1/' > /docs/VERSION
COPY docs/mkdocs.yml /docs/mkdocs-swarm.yml

COPY ./scheduler/strategy/README.md /docs/sources/swarm/scheduler/strategy.md
COPY ./scheduler/filter/README.md /docs/sources/swarm/scheduler/filter.md
COPY ./api/README.md /docs/sources/swarm/API.md
COPY ./discovery/README.md /docs/sources/swarm/discovery.md
#COPY ./discovery/token/README.md /docs/sources/
COPY ./docs/index.md /docs/sources/swarm/index.md

# Then build everything together, ready for mkdocs
RUN /docs/build.sh
