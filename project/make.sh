#!/usr/bin/env bash
set -e

export SWARM_PKG='github.com/docker/swarm'

cd "$GOPATH/src/$SWARM_PKG"
VERSION=$(cat ./VERSION)
if command -v git &> /dev/null && git rev-parse &> /dev/null; then
	GITCOMMIT=$(git rev-parse --short HEAD)
	if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
		GITCOMMIT="$GITCOMMIT-dirty"
	fi
elif [ "$SWARM_GITCOMMIT" ]; then
	GITCOMMIT="$SWARM_GITCOMMIT"
else
	echo >&2 'error: .git directory missing and SWARM_GITCOMMIT not specified'
	echo >&2 '  Please either build with the .git directory accessible, or specify the'
	echo >&2 '  exact (--short) commit hash you are building using SWARM_GITCOMMIT for'
	echo >&2 '  future accountability in diagnosing build issues.  Thanks!'
	exit 1
fi

LDFLAGS='
	-X '$SWARM_PKG'/swarmversion.GITCOMMIT "'$GITCOMMIT'"
	-X '$SWARM_PKG'/swarmversion.VERSION "'$VERSION'"
'

(cd $GOPATH && go build   -ldflags "$LDFLAGS" $SWARM_PKG)
(cd $GOPATH && go install -ldflags "$LDFLAGS" $SWARM_PKG)
