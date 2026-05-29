package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/cmd"
	"github.com/npikall/gotpm/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBumpCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().CountP("verbose", "v", "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().BoolP("show-current", "c", false, "")
	cmd.Flags().BoolP("show-next", "n", false, "")
	cmd.Flags().BoolP("indent", "i", false, "")
	return cmd
}

const bumpManifest = `[package]
name = "my-pkg"
version = "1.2.3"
entrypoint = "lib.typ"
`

func TestBumpRunner_MissingArgument(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	err := BumpRunner(newBumpCmd(t), []string{})
	assert.ErrorIs(t, err, ErrMissingArgument)
}

func TestBumpRunner_NoManifest(t *testing.T) { //nolint: paralleltest
	t.Chdir(t.TempDir())
	err := BumpRunner(newBumpCmd(t), []string{"patch"})
	assert.ErrorIs(t, err, internal.ErrManifestNotFound)
}

func TestBumpRunner_ShowCurrent(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	cmd := newBumpCmd(t)
	check(cmd.Flags().Set("show-current", "true"))
	err := BumpRunner(cmd, []string{})
	require.NoError(t, err)
	// typst.toml must not be modified
	content, _ := os.ReadFile(filepath.Join(dir, "typst.toml"))
	assert.Contains(t, string(content), "1.2.3")
}

func TestBumpRunner_Patch(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	require.NoError(t, BumpRunner(newBumpCmd(t), []string{"patch"}))

	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.4", m.Package.Version)
}

func TestBumpRunner_Minor(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	require.NoError(t, BumpRunner(newBumpCmd(t), []string{"minor"}))

	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.3.0", m.Package.Version)
}

func TestBumpRunner_Major(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	require.NoError(t, BumpRunner(newBumpCmd(t), []string{"major"}))

	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", m.Package.Version)
}

func TestBumpRunner_ExactVersion(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	require.NoError(t, BumpRunner(newBumpCmd(t), []string{"9.8.7"}))

	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "9.8.7", m.Package.Version)
}

func TestBumpRunner_InvalidIncrement(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	err := BumpRunner(newBumpCmd(t), []string{"bogus"})
	assert.ErrorIs(t, err, internal.ErrInvalidIncrement)
}

func TestBumpRunner_DryRun_DoesNotWriteFile(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	cmd := newBumpCmd(t)
	check(cmd.Flags().Set("dry-run", "true"))
	require.NoError(t, BumpRunner(cmd, []string{"patch"}))

	// File must still contain original version
	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", m.Package.Version)
}

func TestBumpRunner_ShowNext_DoesNotWriteFile(t *testing.T) { //nolint: paralleltest
	dir := writeManifest(t, bumpManifest)
	t.Chdir(dir)
	cmd := newBumpCmd(t)
	check(cmd.Flags().Set("show-next", "true"))
	require.NoError(t, BumpRunner(cmd, []string{"patch"}))

	// File must still contain original version
	m, err := internal.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", m.Package.Version)
}
