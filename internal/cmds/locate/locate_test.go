package locate_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/cmds/locate"
	"github.com/npikall/gotpm/internal/logger"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Not parallel: t.Setenv cannot be used from a parallel test.
func TestRunKnownKey(t *testing.T) {
	t.Setenv(paths.InstallDirEnvVar, "/env/packages")

	require.NoError(t, locate.Run("packages", logger.Setup(0)))
}

func TestRunUnknownKey(t *testing.T) {
	t.Parallel()

	err := locate.Run("bogus", logger.Setup(0))

	require.ErrorIs(t, err, locate.ErrUnknownKey)
	assert.Contains(t, err.Error(), `"bogus"`)
	for _, key := range locate.Keys() {
		assert.Contains(t, err.Error(), key)
	}
}
