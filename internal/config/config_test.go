package config_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-moma/internal/config"
)

const testConfig = `# Config
debug = true
verbosity = 2
sharedPath = "/tmp"
[temporal]
address = "host:port"
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"
[worker]
maxConcurrentSessions = 1
[removeFiles]
removeNames = ".DS_Store, Thumbs.db"
`

func TestConfig(t *testing.T) {
	tmpDir := fs.NewDir(
		t, "",
		fs.WithFile(
			"preprocessing.toml",
			testConfig,
		),
	)
	configFile := tmpDir.Join("preprocessing.toml")

	var c config.Configuration
	found, configFileUsed, err := config.Read(&c, configFile)

	assert.NilError(t, err)
	assert.Equal(t, found, true)
	assert.Equal(t, configFileUsed, configFile)

	assert.Equal(t, c.Debug, true)
	assert.Equal(t, c.Verbosity, 2)
	assert.Equal(t, c.SharedPath, "/tmp")

	assert.Equal(t, c.Temporal.Address, "host:port")
	assert.Equal(t, c.Temporal.Namespace, "default")
	assert.Equal(t, c.Temporal.TaskQueue, "preprocessing")
	assert.Equal(t, c.Temporal.WorkflowName, "preprocessing")

	assert.Equal(t, c.Worker.MaxConcurrentSessions, 1)

	assert.Equal(t, c.RemoveFiles.RemoveNames, ".DS_Store, Thumbs.db")
}
