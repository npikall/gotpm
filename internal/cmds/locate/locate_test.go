package locate_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

var errNoManifest = errors.New("no typst.toml found")

func TestRenderAlignsPathsAndExplainsFailures(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	locate.Render(&out, []locate.Group{
		{Name: "machine", Entries: []locate.Entry{
			{Key: "packages", Path: "/env/packages", Note: "TYPST_PACKAGE_PATH"},
			{Key: "cache", Path: "/data/cache"},
		}},
		{Name: "project", Entries: []locate.Entry{
			{Key: "lock", Err: errNoManifest},
		}},
	})

	assert.Equal(t, `machine
  packages /env/packages (TYPST_PACKAGE_PATH)
  cache    /data/cache

project
  lock     unresolved
           no typst.toml found
`, out.String())
}

// inProject runs the rest of the test from inside a fresh project.
func inProject(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	manifest := "[package]\nname = \"doc\"\nversion = \"0.1.0\"\nentrypoint = \"main.typ\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "typst.toml"), []byte(manifest), 0o600))
	t.Chdir(root)
}

func TestRunListsProjectPathsInsideAProject(t *testing.T) { //nolint: paralleltest // t.Chdir
	inProject(t)

	require.NoError(t, locate.Run("", logger.Setup(0)))
}

func TestRunListsMachinePathsOutsideAProject(t *testing.T) { //nolint: paralleltest // t.Chdir
	t.Chdir(t.TempDir())

	require.NoError(t, locate.Run("", logger.Setup(0)))
}

func TestRunProjectKeyInsideAProject(t *testing.T) { //nolint: paralleltest // t.Chdir
	inProject(t)

	require.NoError(t, locate.Run("lock", logger.Setup(0)))
}

func TestRunProjectKeyOutsideAProject(t *testing.T) { //nolint: paralleltest // t.Chdir
	t.Chdir(t.TempDir())

	require.Error(t, locate.Run("lock", logger.Setup(0)))
}
