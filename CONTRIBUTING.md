# Contributing to Swarm

Want to hack on Swarm? Awesome! Here are instructions to get you
started.

Swarm is a part of the [Docker](https://www.docker.com) project, and follows
the same rules and principles. If you're already familiar with the way
Docker does things, you'll feel right at home.

Otherwise, go read
[Docker's contributions guidelines](https://github.com/docker/docker/blob/master/CONTRIBUTING.md).

### Development Environment Setup

Swarm is written in [the Go programming language](http://golang.org) and manages its dependencies using [Godep](http://github.com/tools/godep). Go and Godep also use `git` to fetch dependencies.
So, you are required to install the Go compiler, and the `git` client (e.g. `apt-get install golang git` on Ubuntu).

You may need to setup your path and set the `$GOPATH` and `$GOBIN` variables, e.g:
```sh
mkdir -p ~/projects/swarm
cd ~/projects/swarm
export GOPATH=$PWD
export GOBIN=$GOPATH/bin
```

As previously mentioned, you need `godep` to manage dependencies for Swarm. The `godep` command can be installed using the following command:
```sh
go get github.com/tools/godep
```
and you will find the `godep` binary under `$GOBIN`.

Then you can fetch `swarm` code to your `$GOPATH`.
```sh
go get -d github.com/docker/swarm
```

#### Start hacking

First, prepare dependencies for `swarm` and try to compile it.
```sh
cd src/github.com/docker/swarm
$GOBIN/godep restore
go test
go install
```

#### Adding New Dependencies

To make sure other will not miss dependencies you've added to Swarm, you'll need to call `godep save` to make changes to the config file, `Godep/Godeps.json`. An important thing is that `godep` will replace the config file by the dependency information it learnt from your local machine. This step will mess the upstream config. So, changes to `Godep/Godeps.json` must be performed with care.

```sh
$GOBIN/godep save ./...
git diff # check what added or removed in Godep/Godeps.json
         # then manually add missing dependencies
```

To make sure you newly added codes will make the build process happy, you can try building Swarm in the same way as defined in `Dockerfile`.

```sh
$GOBIN/godep go install
```
Then you should find the `swarm` binary under the `$GOBIN` directory.

Happy hacking!
