FROM docs/base:latest
MAINTAINER Mary <mary@docker.com> (@moxiegirl)

# to get the git info for this repo
COPY . /src

# Reset the /docs dir so we can replace the theme meta with the new repo's git info
RUN git reset --hard

RUN grep "VERSION =" /src/version/version.go  | sed 's/.*"\(.*\)".*/\1/' > /docs/VERSION


COPY * /docs/sources/swarm/
COPY scheduler/* /docs/sources/swarm/scheduler/
COPY api/* /docs/sources/swarm/api/
COPY mkdocs.yml /docs/mkdocs-swarm.yml

# Then build everything together, ready for mkdocs
RUN /docs/build.sh
