package run_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger { return log.New(io.Discard) }

// record makes the launcher record launches instead of replacing the process.
func record(t *testing.T) *[]run.Launch {
	t.Helper()
	var launches []run.Launch
	t.Cleanup(run.UseLauncher(func(l run.Launch) error {
		launches = append(launches, l)
		return nil
	}))
	return &launches
}

func compilerDir(t *testing.T) compilers.Dir {
	t.Helper()
	dir, err := compilers.OpenDir()
	require.NoError(t, err)
	return dir
}

func binary(t *testing.T, version string) string {
	t.Helper()
	path, err := compilerDir(t).Binary(version)
	require.NoError(t, err)
	return path
}

// outside makes an empty directory, belonging to no project, the working
// directory.
func outside(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

func install(t *testing.T, srv *compilertest.Server, versions ...string) {
	t.Helper()
	for _, version := range versions {
		_, err := compilerDir(t).Ensure(context.Background(), srv.Source(), version)
		require.NoError(t, err)
	}
}

func TestRun_OutsideAProjectLaunchesTheNewestInstalledStableCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	for _, version := range []string{"0.12.0", "0.13.1", "0.15.0"} {
		srv.Publish(version)
	}
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())
	install(t, srv, "0.12.0", "0.13.1", "0.14.0-rc.1")
	before := len(srv.Requests())
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "", []string{"compile", "notes.typ"}, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary,
		"neither a pre-release nor a newer release that is not installed")
	assert.Len(t, srv.Requests(), before, "an installed compiler needs no network")
}

func TestRun_OutsideAProjectInstallsTheLatestStableReleaseWhenNoneIsInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	home := os.Getenv("HOME")
	cwd := outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	srv.Publish("0.15.0-rc.1", compilertest.Prerelease())
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "", []string{"compile", "notes.typ"}, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.14.0"), (*launches)[0].Binary)
	installed, err := compilerDir(t).Installed()
	require.NoError(t, err)
	assert.Equal(t, []string{"0.14.0"}, installed)
	assert.Empty(t, entries(t, cwd), "outside a project nothing is written to the working directory")
	assert.Empty(t, filesOutside(t, home, compilerDir(t).Root), "only the compiler directory is written")
}

func TestRun_HandsEveryArgumentAndTheEnvironmentToTheCompiler(t *testing.T) {
	testrepo.Isolate(t)
	outside(t)
	t.Setenv("GOTPM_RUN_TEST", "kept")
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	launches := record(t)
	args := []string{"compile", "--help", "-v", "--", "my notes.typ"}

	require.NoError(t, run.Run(context.Background(), "", args, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, args, (*launches)[0].Args)
	assert.Contains(t, (*launches)[0].Env, "GOTPM_RUN_TEST=kept")
}

func TestRun_NeverUsesATypstOnThePath(t *testing.T) {
	testrepo.Isolate(t)
	outside(t)
	bin := t.TempDir()
	require.NoError(t, paths.WriteFile(filepath.Join(bin, "typst"), []byte("#!/bin/sh\n")))
	t.Setenv("PATH", bin)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "", []string{"--version"}, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.14.0"), (*launches)[0].Binary)
}

func TestRun_ReturnsWhatTheLauncherReturns(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	failed := assert.AnError
	t.Cleanup(run.UseLauncher(func(run.Launch) error { return failed }))

	require.ErrorIs(t, run.Run(context.Background(), "", nil, io.Discard, discardLogger()), failed)
}

func TestRun_FailsWhenNoStableReleaseIsPublished(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.15.0-rc.1", compilertest.Prerelease())
	launches := record(t)

	require.ErrorIs(t, run.Run(context.Background(), "", nil, io.Discard, discardLogger()), compilers.ErrNoRelease)
	assert.Empty(t, *launches)
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	found, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(found))
	for _, entry := range found {
		names = append(names, entry.Name())
	}
	return names
}

// filesOutside lists the files under root that are not inside keep.
func filesOutside(t *testing.T, root, keep string) []string {
	t.Helper()
	var found []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if rel, _ := filepath.Rel(keep, path); !filepath.IsLocal(rel) {
			found = append(found, path)
		}
		return nil
	}))
	return found
}
