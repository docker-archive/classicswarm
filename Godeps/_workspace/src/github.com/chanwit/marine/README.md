# Marine

`Marine` is a functional testing framework designed mainly to test [Swarm](http://github.com/docker/swarm).
But we can generally use `Marine` to test several kinds of cluster-based application.

Marine contains a set of GO APIs to prepare a cluster, initialize network for the cluster nodes, allow us to install software, e.g., Docker and Swarm on them.
As `Marine` is desinged to test `Swarm` and `Docker`, it directly uses VirtualBox to manage cluster machines.

### Installation

#### Image Preparation

Marine comes with a little utility to help us prepare a ready-to-use VM image that contains `docker` and `golang`. The following command imports Ubuntu 14.10 and performs the preparation.

```
$ go get github.com/chanwit/marine/cmd/marine
$ $GOBIN/marine prepare                   \
  -i=files/ubuntu-14.10-server-amd64.ova  \
  -o=files/ubuntu-docker-golang.ova
INFO[0015] Imported "ubuntu"
INFO[0015] Modified nic2 for "ubuntu"
INFO[0015] Started "ubuntu"
INFO[0035] VM "ubuntu" ready to connect
INFO[0035] Sudo: "wget -qO- --no-check-certificate https://get.docker.com/ | bash"
INFO[0141] Sudo: "service docker start"
INFO[0141] Sudo: "docker pull golang:1.3"
INFO[0275] Stopping "ubuntu"
INFO[0290] VM "ubuntu" is now poweroff
INFO[0290] Snapshot "ubuntu/origin" taken
INFO[0407] Exported "ubuntu" to files/ubuntu-docker-golang.ova
INFO[0409] Removed "ubuntu"
```

#### Functional Test

`Marine` has been designed to use in Go test.
Just create a test file and

```go
import github.com/chanwit/marine
```

This example shows how to setup a pre-check that import the prepared image.
```go
func checkFunctional(t *testing.T) {
	if os.Getenv("SWARM_FUNC_TEST") != "" {
		// you need to import a VM first.
		if exist, err := vm.Exist("ubuntu"); err == nil && !exist {
			ubuntu, err := vm.Import(
				path.Join(os.Getenv("GOPATH"), "files/ubuntu-docker-golang.ova"),
				512)
			assert.NoError(t, err)
			assert.NotNil(t, ubuntu)
		}
	} else {
		t.Skip(`Enable this test with "export SWARM_FUNC_TEST=true"`)
	}
}
```

This test case uses the above VM, compiles `Swarm` from its `HEAD`.
Then it tests that the `Swarm`'s version should have a proper format.
```go
func TestVersionCommitId(t *testing.T) {
	checkFunctional(t)

	ubuntu := &vm.Machine{Name: "ubuntu"}
	box, err := ubuntu.Clone("box")
	box.StartAndWait()
	defer func() {
		box.Stop()
		box.Remove()
	}()

	// build swarm from master/HEAD
	err = box.BuildSwarm("docker/swarm")
	assert.NoError(t, err)

	v := strings.Replace(version.VERSION, ".", `\.`, 2)
	pattern, err := regexp.Compile(`^swarm version ` + v + ` \([0-9a-f]{7}\)$`)
	assert.NoError(t, err)

	out, err := box.Sudo("docker run --rm -t swarm:build --version")
	assert.NoError(t, err)
	assert.True(t, pattern.MatchString(strings.TrimSpace(string(out))))
}
```

#### VM Clones
To clone the above image and prepare 4 VMs, we can call the following command.

```go
	ubuntu := &marine.Machine{Name:"ubuntu"}
	boxes, err := ubuntu.CloneN(4, "box")
```

Please note that all VMs will share an auto-config `host-only` network.
We'll have `box001`, `box002`, `box003` and `box004` if the command runs successfully.

Each node will have its own port-forwarding. VM `box001` can be connected via `127.0.0.1:52201` and so on.

### Creator

`Marine` is created by

Chanwit Kaewkasi,
Copyright 2015 Suranaree University of Technology.

`Marine` code is made available under Apache Software License 2.0.
Its document is available under Creative Commons.