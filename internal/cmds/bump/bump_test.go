package bump_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/bump"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeManifest creates a package directory holding bumpManifest.
func writeManifest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(bumpManifest), paths.FilePerm))
	return dir
}

func discardLogger() *log.Logger {
	return log.New(io.Discard)
}

const bumpManifest = `[package]
name = "my-pkg"
version = "1.2.3"
entrypoint = "lib.typ"
`

func TestRun_MissingArgument(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	err := bump.Run("", &bump.Options{}, discardLogger())
	assert.ErrorIs(t, err, bump.ErrMissingArgument)
}

func TestRun_NoManifest(t *testing.T) { //nolint: paralleltest
	t.Chdir(t.TempDir())
	err := bump.Run("patch", &bump.Options{}, discardLogger())
	assert.ErrorIs(t, err, manifest.ErrManifestNotFound)
}

func TestRun_ShowCurrent(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	err := bump.Run("", &bump.Options{ShowCur: true}, discardLogger())
	require.NoError(t, err)
	// typst.toml must not be modified
	content, _ := os.ReadFile(filepath.Join(dir, "typst.toml"))
	assert.Contains(t, string(content), "1.2.3")
}

func TestRun_Patch(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("patch", &bump.Options{}, discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.4", m.Package.Version)
}

func TestRun_Minor(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("minor", &bump.Options{}, discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.3.0", m.Package.Version)
}

func TestRun_Major(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("major", &bump.Options{}, discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", m.Package.Version)
}

func TestRun_ExactVersion(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("9.8.7", &bump.Options{}, discardLogger()))

	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "9.8.7", m.Package.Version)
}

func TestRun_InvalidIncrement(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	err := bump.Run("bogus", &bump.Options{}, discardLogger())
	assert.ErrorIs(t, err, semver.ErrInvalidIncrement)
}

func TestRun_DryRun_DoesNotWriteFile(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("patch", &bump.Options{DryRun: true}, discardLogger()))

	// File must still contain original version
	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", m.Package.Version)
}

func TestRun_ShowNext_DoesNotWriteFile(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t)
	t.Chdir(dir)
	require.NoError(t, bump.Run("patch", &bump.Options{ShowNext: true}, discardLogger()))

	// File must still contain original version
	m, err := manifest.LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", m.Package.Version)
}
