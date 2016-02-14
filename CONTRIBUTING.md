# Contributing to Swarm

Want to hack on Swarm? Awesome! Here are instructions to get you
started.

Swarm is a part of the [Docker](https://www.docker.com) project, and follows
the same rules and principles. If you're already familiar with the way
Docker does things, you'll feel right at home.

Otherwise, go read Docker's
[contributions guidelines](https://github.com/docker/docker/blob/master/CONTRIBUTING.md),
[issue triaging](https://github.com/docker/docker/blob/master/project/ISSUE-TRIAGE.md),
[review process](https://github.com/docker/docker/blob/master/project/REVIEWING.md) and
[branches and tags](https://github.com/docker/docker/blob/master/project/BRANCHES-AND-TAGS.md).

### Development Environment Setup

Swarm is written in [the Go programming language](http://golang.org) and manages its dependencies using [Godep](http://github.com/tools/godep).  This guide will walk you through installing Go, forking the Swarm repo, building Swarm from source and contributing pull requests (PRs) back to the Swarm project.

#### Install git, Go and Godep
If you don't already have `git` installed, you should install it.  For example, on Ubuntu:
```sh
sudo apt-get install git
```

You also need Go 1.5 or higher.  Download Go from [https://golang.org/dl/](https://golang.org/dl/).  To install on Linux:
```sh
tar xzvf go1.5.3.linux-amd64.tar.gz
sudo mv go /usr/local
```

> **Note**: On Ubuntu, do not use `apt-get` to install Go.  Its repositories tend
> to include older versions of Go.  Instead, install the latest Go manually using the
> instructions provided on the Go site.

Create a go project directory in your home directory:
```sh
mkdir ~/gocode    # any name is fine
```

Add these to your `.bashrc`:
```sh
export GOROOT=/usr/local/go
export GOPATH=~/gocode
export PATH=$PATH:$GOPATH/bin:$GOROOT/bin
```

Close and reopen your terminal.

Install Godep:

```sh
go get github.com/tools/godep
```

Install golint:

```sh
go get github.com/golang/lint/golint
```

#### Fork the Swarm repo

You will create a Github fork of the Swarm repository.  You will make changes to this fork.  When your work is ready for review, open a pull request to contribute the work back to the main Swarm repo.

Go to the [`docker/swarm` repo](https://github.com/docker/swarm) in a browser and click the "Fork" button in the top right corner.  You'll end up with your form of Swarm in `https://github.com/<username>/swarm`.

(Throughout this document, `<username>` refers to your Github username.)

#### Clone Swarm and set remotes

Now clone the main Swarm repo:

```sh
mkdir -p $GOPATH/src/github.com/docker
cd $GOPATH/src/github.com/docker
git clone https://github.com/docker/swarm.git
cd swarm
```

For the easiest contribution workflow, we will set up remotes as follows:
  * `origin` is your personal fork
  * `upstream` is `docker/swarm`

Run these commands:

```sh
git remote remove origin
git remote add origin https://github.com/<username>/swarm.git
git remote add upstream https://github.com/docker/swarm.git
git remote set-url --push upstream no-pushing
```

You can check your configuration like this:
```sh
$ git remote -v
origin     https://github.com/mgoelzer/swarm.git (fetch)
origin     https://github.com/mgoelzer/swarm.git (push)
upstream   https://github.com/docker/swarm.git (fetch)
upstream   no-pushing (push)
```

#### Pull `docker/swarm` changes into your fork

As the `docker/swarm` moves forward, it will accumulate commits that are not in your personal fork.  To keep your fork up to date, you need to periodically rebase:

```sh
git fetch upstream master
git rebase upstream/master
git push -f origin master
```

Don't worry:  fetching the upstream and rebasing will not overwrite your local changes.

### Build the Swarm binary

Build the binary, installing it to `$GOPATH/bin/swarm`:

```sh
cd $GOPATH/src/github.com/docker/swarm
godep go install .
```
 
Run the binary you just created:

```sh
$GOPATH/bin/swarm help
$GOPATH/bin/swarm create
$GOPATH/bin/swarm manage token://<cluster_id>
$GOPATH/bin/swarm join --addr=<node_ip:2375> token://<cluster_id>
```

#### Create a Swarm image for deployment

Swarm is distributed as a Docker image with a single `swarm` binary inside.  To create the image:

```sh
docker build -t my_swarm .
```

Now you can run the same commands as above using the container:

```sh
docker run my_swarm help
docker run -d my_swarm join --addr=<node_ip:2375> token://<cluster_id>
docker run -d -p <manager_port>:2375 my_swarm manage token://<cluster_id>
```

For complete documentation on how to use Swarm, refer to the Swarm section of [docs.docker.com](http://docs.docker.com/).


### Run tests

To run unit tests:

```sh
godep go test -race ./...
```

To run integration tests:

```sh
./test/integration/run.sh
```

You can use this command to check if `*.bats` files are formatted correctly:

```sh
find test/ -type f \( -name "*.sh" -or -name "*.bash" -or -name "*.bats" \) -exec grep -Hn -e "^ " {} \;
```

You can use this command to check if `*.go` files are formatted correctly:

```sh
golint ./...
```

### Submit a pull request (PR)

Swarm welcomes your contributions back to the project!  We follow the same contribution process and principles as other Docker projects.  This tutorial describes the [Docker contribution workflow](https://docs.docker.com/opensource/workflow/make-a-contribution/).

In general, your contribution workflow for Swarm will look like this:

```sh
cd $GOPATH/src/github.com/docker/swarm
git checkout master                     # Make sure you have a clean master
git pull upstream master                #   ...before you create your branch.
git checkout -b issue-9999              # Create a branch for your work
vi README.md                            # Edit a file in your branch
git add README.md                       # Git add your changed files
git commit -s -m "Edited README"        # Make a signed commit to your local branch
git push origin issue-9999              # Push your local branch to your fork
```

To open a PR, browse to your fork at `http://github.com/<username>/swarm` and select the `issue-9999` branch from "Branch" drop down list.  Click the green "New Pull Request" button and submit your PR.  Maintainers will review your PR and merge it or provide feedback on how to improve it.


### Revise your PR

You may receive suggested changes on your PR before it can be merged.  If you do, make the changes in the same branch, then do something like:

```sh
git checkout issue-9999
git add <changed files>
git commit -s --amend --no-edit
git push -f origin issue-9999
```

Github will magically update the open PR to add these changes.

### Talk to us

The best way to chat with other Swarm developers is on freenode `#docker-swarm`.  ([See here](https://docs.docker.com/opensource/get-help/) for IRC instructions.)

You can also find maintainers' email addresses in `MAINTAINERS`.

### Advanced:  Adding New Dependencies

To make sure other will not miss dependencies you've added to Swarm, you'll need to call `godep save` to make changes to the config file, `Godep/Godeps.json`. An important thing is that `godep` will replace the config file by the dependency information it learnt from your local machine. This step will mess the upstream config. So, changes to `Godep/Godeps.json` must be performed with care.

```sh
$GOBIN/godep save ./...
$GOBIN/godep update <an updated package>
git diff # check what added or removed in Godep/Godeps.json
         # then manually add missing dependencies
```

To make sure you newly added codes will make the build process happy, you can try building Swarm in the same way as defined in `Dockerfile`.

```sh
$GOBIN/godep go install
```
Then you should find the `swarm` binary under the `$GOBIN` directory.

Happy hacking!
