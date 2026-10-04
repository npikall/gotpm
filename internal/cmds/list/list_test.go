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

func TestRun_WithoutAPackageDirectory(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)

	require.ErrorIs(t, list.Run(discardLogger()), list.ErrNoPackages)
}

func TestRun_AnEmptyPackageDirectory(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	packages := testrepo.Isolate(t)
	require.NoError(t, os.MkdirAll(packages, 0o755))

	require.NoError(t, list.Run(discardLogger()))
}

func TestRun_ManyAndEditableVersions(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	packages := testrepo.Isolate(t)
	for _, version := range []string{"0.1.0", "0.2.0", "0.3.0", "0.4.0", "0.5.0", "0.6.0", "0.7.0"} {
		install(t, packages, "cetz", version)
	}
	linked := filepath.Join(packages, "local", "mine")
	require.NoError(t, os.MkdirAll(linked, 0o755))
	require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(linked, "0.1.0")))

	require.NoError(t, list.Run(discardLogger()))
}
