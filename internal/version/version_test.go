package version_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-moma/internal/version"
)

func TestVersion(t *testing.T) {
	// The results below lack vcs build info due to bug:
	// https://github.com/golang/go/issues/33976, and the output should include
	// the expected git data when the bug is fixed.
	assert.Equal(t, version.Short, "0.0.0-dev")
	assert.Equal(t, version.Long, "0.0.0-dev-t")
	assert.Equal(t, version.Info("testapp"), "testapp version 0.0.0-dev-t")
}
