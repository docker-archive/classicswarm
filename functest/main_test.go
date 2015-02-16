package functest

import (
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	vm "github.com/chanwit/marine"
	"github.com/docker/swarm/version"
	"github.com/stretchr/testify/assert"
)

//
// Test if swarm returns version in a proper format
//
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

func checkFunctional(t *testing.T) {
	// Functional tests are slow,
	// run only when needed.
	if os.Getenv("SWARM_FUNC_TEST") != "" {
		// You need to import a VM first.
		if exist, err := vm.Exist("ubuntu"); err == nil && !exist {
			// Prepare this VM with:
			// $ marine prepare -i=input.ova -o=output.ova
			// See github.com/chanwit/marine for more information.
			imagefile := os.Getenv("SWARM_FUNC_TEST_IMAGE")
			if imagefile == "" {
				t.Skip(`SWARM_FUNC_TEST_IMAGE not defined.
Set SWARM_FUNC_TEST_IMAGE to an image file.
You can prepare an image using command "marine prepare"`)
			}

			ubuntu, err := vm.Import(
				path.Join(imagefile),
				512)
			assert.NoError(t, err)
			assert.NotNil(t, ubuntu)
		}
	} else {
		t.Skip(`Enable this test with "export SWARM_FUNC_TEST=true"`)
	}
}
