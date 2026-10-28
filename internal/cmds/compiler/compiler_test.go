package compiler_test

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/compiler"
	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

func compilerDir(t *testing.T) compilers.Dir {
	t.Helper()
	dir, err := compilers.OpenDir()
	require.NoError(t, err)
	return dir
}

// install publishes versions on a fake release source and installs them.
func install(t *testing.T, srv *compilertest.Server, versions ...string) {
	t.Helper()
	for _, version := range versions {
		srv.Publish(version)
		_, err := compilerDir(t).Ensure(context.Background(), srv.Source(), version)
		require.NoError(t, err)
	}
}

// outside makes an empty directory, belonging to no project, the working
// directory.
func outside(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

// project writes a manifest pinning pin and makes a subdirectory of the
// project the working directory.
func project(t *testing.T, pin string) {
	t.Helper()
	root := t.TempDir()
	content := `[package]
name = "notes"
version = "0.1.0"
entrypoint = "main.typ"

[tool.gotpm]
compiler-pin = "` + pin + `"
`
	require.NoError(t, paths.WriteFile(filepath.Join(root, manifest.FileName), []byte(content)))
	sub := filepath.Join(root, "chapters")
	require.NoError(t, paths.EnsureDir(sub))
	t.Chdir(sub)
}

func list(t *testing.T) []string {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, compiler.List(&out, discardLogger()))
	return strings.Split(strings.TrimSpace(out.String()), "\n")
}

func TestList_SaysSoWhenNoCompilerIsInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)

	lines := list(t)

	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "no compiler installed")
}

func TestList_ShowsInstalledCompilersInVersionOrderMarkingTheNewestStable(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	install(t, srv, "0.13.1", "0.9.0", "0.10.0")
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())
	install(t, srv, "0.14.0-rc.1")

	assert.Equal(t, []string{"0.9.0", "0.10.0", "0.13.1 (newest)", "0.14.0-rc.1"}, list(t))
}

func TestList_InsideAProjectMarksThePinnedCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := compilertest.Serve(t)
	install(t, srv, "0.12.0", "0.13.1")
	project(t, "0.12.0")

	assert.Equal(t, []string{"0.12.0 (pinned)", "0.13.1 (newest)"}, list(t))
}

func TestList_ShowsAPinThatIsNotInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := compilertest.Serve(t)
	install(t, srv, "0.12.0", "0.14.0")
	project(t, "0.13.1")

	assert.Equal(t, []string{"0.12.0", "0.13.1 (pinned, not installed)", "0.14.0 (newest)"}, list(t))
}

func TestList_ShowsAPinThatIsNotInstalledWhenNothingIs(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project(t, "0.13.1")

	lines := list(t)

	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "no compiler installed")
	assert.Equal(t, "0.13.1 (pinned, not installed)", lines[1])
}

func TestUninstall_RemovesTheCompilerAndReportsIt(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	install(t, srv, "0.13.1", "0.14.0")
	var out bytes.Buffer

	require.NoError(t, compiler.Uninstall("0.13.1", &out, discardLogger()))

	installed, err := compilerDir(t).Installed()
	require.NoError(t, err)
	assert.Equal(t, []string{"0.14.0"}, installed)
	assert.Contains(t, out.String(), "uninstalled compiler 0.13.1")
}

func TestUninstall_FailsNamingAVersionThatIsNotInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)

	err := compiler.Uninstall("0.13.1", io.Discard, discardLogger())

	require.ErrorIs(t, err, compilers.ErrNotInstalled)
	assert.Contains(t, err.Error(), "0.13.1")
}

func TestUninstall_OfThePinnedCompilerIsUndoneByTheNextRun(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := compilertest.Serve(t)
	install(t, srv, "0.13.1")
	project(t, "0.13.1")
	t.Cleanup(run.UseLauncher(func(run.Launch) error { return nil }))

	require.NoError(t, compiler.Uninstall("0.13.1", io.Discard, discardLogger()))
	_, err := compilerDir(t).Binary("0.13.1")
	require.ErrorIs(t, err, compilers.ErrNotInstalled)

	require.NoError(t, run.Run(context.Background(), "", []string{"compile", "main.typ"}, io.Discard, discardLogger()))

	_, err = compilerDir(t).Binary("0.13.1")
	assert.NoError(t, err, "run installs the pinned compiler again")
}
