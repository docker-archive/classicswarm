package marine

import (
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewAPIs(t *testing.T) {
	t.Skip()
	log.Info("Start testing")
	base, err := Import(os.Getenv("GOPATH")+"/files/ubuntu-14.10-server-amd64.ova", 512)
	assert.NoError(t, err)
	boxes, err := base.CloneN(4, "box")
	assert.NoError(t, err)

	defer func() {
		boxes[0].Stop()
		for _, box := range boxes {
			assert.NoError(t, box.Remove())
		}
		assert.NoError(t, base.Remove())
	}()

	boxes[0].StartAndWait()
	if out, err := boxes[0].Run("/usr/bin/whoami"); err == nil {
		assert.Equal(t, "ubuntu", strings.TrimSpace(string(out)))
	}
	if out, err := boxes[0].Sudo("whoami"); err == nil {
		assert.Equal(t, "root", strings.TrimSpace(out))
	}
}

func TestListingIP(t *testing.T) {
	t.Skip()

	log.Info("Start testing")
	base, err := Import(os.Getenv("GOPATH")+"/files/ubuntu-14.10-server-amd64.ova", 512)
	assert.NoError(t, err)
	box, err := base.Clone("box")
	assert.NoError(t, err)

	defer func() {
		box.Stop()
		box.Remove()
		base.Remove()
	}()

	box.StartAndWait()
	if out, err := box.Sudo(`sed -i "\$aauto eth1\niface eth1 inet dhcp\n" /etc/network/interfaces`); err == nil {
		log.Info(string(out))
	}
	if out, err := box.Sudo("cat /etc/network/interfaces"); err == nil {
		log.Info(string(out))
	}
	if out, err := box.Sudo("ifup eth1"); err == nil {
		log.Info(string(out))
	}
	time.Sleep(3 * time.Second)
	if out, err := box.Run("/sbin/ip addr show dev eth1"); err == nil {
		log.Info("\n" + string(out))
	}

}

func TestGettingIP(t *testing.T) {
	t.Skip()
	log.Info("Start testing")
	base, err := Import(os.Getenv("GOPATH")+"/files/ubuntu-14.10-server-amd64.ova", 512)
	assert.NoError(t, err)
	boxes, err := base.CloneN(1, "box")
	assert.NoError(t, err)

	defer func() {
		boxes[0].Stop()
		boxes[0].Remove()
		base.Remove()
	}()

	boxes[0].StartAndWait()
	assert.NoError(t, boxes[0].SetupIPAddr())
	ip, err := boxes[0].GetIPAddr()
	log.Infof("IP Address (eth1): %s", ip)
	assert.NoError(t, err)
	assert.Equal(t, true, strings.HasPrefix(ip, "192.168.99."))
}

func prepareBoxWithDocker() *Machine {
	base, _ := Import(os.Getenv("GOPATH")+"/files/ubuntu-14.10-server-amd64.ova", 512,
		"docker",
		"golang")
	return base
}

func TestInstallDocker(t *testing.T) {
	t.Skip()
	log.Info("Start testing")
	base := &Machine{Name: "base"}
	if !base.Exist() {
		base = prepareBoxWithDocker()
	}
	boxes, err := base.CloneN(1, "box")
	assert.NoError(t, err)

	defer func() {
		boxes[0].Stop()
		boxes[0].Remove()
	}()

	boxes[0].StartAndWait()
	out, err := boxes[0].Sudo("docker version")
	if err != nil {
		log.Error(err)
	}
	log.Info(out)
}

func TestRunningSwarm(t *testing.T) {
	t.Skip()
	base := &Machine{Name: "base"}
	boxes, err := base.CloneN(1, "box")
	assert.NoError(t, err)

	defer func() {
		boxes[0].Stop()
		boxes[0].Remove()
	}()

	boxes[0].StartAndWait()
	out, err := boxes[0].Sudo("docker run --rm swarm create")
	assert.NoError(t, err)
	hash := strings.TrimSpace(out)
	log.Info(hash)
	assert.Equal(t, 32, len(hash))
}

func TestCompilingSwarm(t *testing.T) {
	t.Skip()
	base := &Machine{Name: "base"}
	box, err := base.Clone("box")
	assert.NoError(t, err)

	defer func() {
		box.Stop()
		box.Remove()
	}()

	box.StartAndWait()
	err = box.BuildSwarm("chanwit/swarm", "plugin-system")
	assert.NoError(t, err)

	out, err := box.Sudo("docker run --rm -t swarm:build --version")
	assert.NoError(t, err)
	log.Info(out)
}

func TestExisting(t *testing.T) {
	exist, err := Exist("ubuntu")
	assert.NoError(t, err)
	assert.False(t, exist)
}

func TestCompilingSwarmHEAD(t *testing.T) {
	exist, err := Exist("ubuntu")
	assert.NoError(t, err)
	assert.False(t, exist)

	base, err := Import(os.Getenv("GOPATH")+"/files/ubuntu-14.10-amd64-docker-1.5.0.ova", 512)

	exist, err = Exist("ubuntu")
	assert.NoError(t, err)
	assert.True(t, exist)

	box, err := base.Clone("box")
	assert.NoError(t, err)

	defer func() {
		box.Stop()
		box.Remove()
		base.Remove()
	}()

	box.StartAndWait()
	err = box.BuildSwarm("docker/swarm")
	assert.NoError(t, err)

	out, err := box.Sudo("docker run --rm -t swarm:build --version")
	assert.NoError(t, err)
	log.Info(out)
}

/*
func TestMain(m *testing.M) {
	// check if "base" vm exists?
	base := &Machine{Name: "base"}
	if !base.Exist() {
		prepareBoxWithDocker()
	}
	os.Exit(m.Run())
}
*/
