package strategy

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginInitialization(t *testing.T) {
	// Default plugin number is 2.
	InitializePlugins("./")
	assert.Equal(t, 2, len(strategies))

	// Place a new plugin, got it in the strategies map.
	dir, err := ioutil.TempDir("", "swarm-plugins-")
	assert.NoError(t, err)
	filename := path.Join(dir, "swarm-strategy-temp")
	err = ioutil.WriteFile(filename, []byte{0, 1, 2, 3, 4}, os.FileMode(644))
	assert.NoError(t, err)

	plugins := InitializePlugins(dir)
	assert.Equal(t, 3, len(strategies))
	assert.Equal(t, 3, len(plugins))
	assert.Contains(t, plugins, "temp")
	plugin, exist := strategies["temp"]
	assert.NotNil(t, plugin)
	assert.True(t, exist)
	_, ok := plugin.(*StrategyPlugin)
	assert.True(t, ok)
}

func TestPluginOverride(t *testing.T) {
	// Place a new plugin with the existing name,
	// it should override the old one in the strategies map.
	dir, err := ioutil.TempDir("", "swarm-plugins-")
	assert.NoError(t, err)
	filename := path.Join(dir, "swarm-strategy-random")
	err = ioutil.WriteFile(filename, []byte{0, 1, 2, 3, 4}, os.FileMode(644))
	assert.NoError(t, err)

	InitializePlugins(dir)
	assert.Equal(t, 2, len(strategies))
	plugin, exist := strategies["random"]
	assert.NotNil(t, plugin)
	assert.True(t, exist)
	_, ok := plugin.(*StrategyPlugin)
	assert.True(t, ok)
}
