package list_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/list"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

// install writes a package into the package directory by hand, which is all
// list needs to find it.
func install(t *testing.T, root, name, version string) {
	t.Helper()
	dir := filepath.Join(root, "local", name, version)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "typst.toml"), nil, 0o644))
}

func TestRun_ListsThePackageDirectoryDespiteAnInstallDirOverride(t *testing.T) {
	packages := testrepo.Isolate(t)
	install(t, packages, "cetz", "0.3.1")
	t.Setenv(paths.InstallDirEnvVar, filepath.Join(t.TempDir(), "stale"))

	err := list.Run(discardLogger())

	require.NoError(t, err,
		"an install dir left set in the shell must not make an installed package invisible")
}
