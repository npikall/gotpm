package compilers_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDir(t *testing.T) compilers.Dir {
	t.Helper()
	return compilers.Dir{Root: filepath.Join(t.TempDir(), "compilers")}
}

func readBinary(t *testing.T, dir compilers.Dir, version string) string {
	t.Helper()
	path, err := dir.Binary(version)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestStable_LeavesOutPreReleases(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.12.0")
	srv.Publish("0.13.1")
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())

	releases, err := compilers.Stable(context.Background(), srv.Source())
	require.NoError(t, err)

	versions := make([]string, 0, len(releases))
	for _, r := range releases {
		versions = append(versions, r.Version)
	}
	assert.Equal(t, []string{"0.13.1", "0.12.0"}, versions)
}

func TestLatest_IsTheNewestStableRelease(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	srv.Publish("0.9.0")
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())

	latest, err := compilers.Latest(context.Background(), srv.Source())
	require.NoError(t, err)

	assert.Equal(t, "0.13.1", latest.Version)
}

func TestLatest_FailsWithoutAnyStableRelease(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())

	_, err := compilers.Latest(context.Background(), srv.Source())

	require.ErrorIs(t, err, compilers.ErrNoRelease)
}

func TestEnsure_InstallsTheBinaryOfTheVersion(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	dir := newDir(t)

	installed, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")
	require.NoError(t, err)

	assert.True(t, installed)
	assert.Equal(t, compilertest.BinaryContent("0.13.1"), readBinary(t, dir, "0.13.1"))
}

func TestEnsure_MakesTheBinaryExecutable(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("windows has no executable bit")
	}
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	dir := newDir(t)

	_, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")
	require.NoError(t, err)

	path, err := dir.Binary("0.13.1")
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode().Perm()&0o100, "binary is not executable: %s", info.Mode())
}

func TestEnsure_InstallsANamedPreRelease(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.14.0-rc.1", compilertest.Prerelease())
	dir := newDir(t)

	_, err := dir.Ensure(context.Background(), srv.Source(), "0.14.0-rc.1")
	require.NoError(t, err)

	assert.Equal(t, compilertest.BinaryContent("0.14.0-rc.1"), readBinary(t, dir, "0.14.0-rc.1"))
}

func TestEnsure_RefusesADigestMismatchAndLeavesNothingBehind(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1", compilertest.WrongDigest())
	dir := newDir(t)

	_, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")

	require.ErrorIs(t, err, compilers.ErrChecksum)
	entries, _ := os.ReadDir(dir.Root)
	assert.Empty(t, entries)
	_, err = dir.Binary("0.13.1")
	require.ErrorIs(t, err, compilers.ErrNotInstalled)
}

func TestEnsure_RefusesAnArchiveWithoutDigest(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.10.0", compilertest.NoDigest())
	dir := newDir(t)

	_, err := dir.Ensure(context.Background(), srv.Source(), "0.10.0")

	require.ErrorIs(t, err, compilers.ErrNoDigest)
	entries, _ := os.ReadDir(dir.Root)
	assert.Empty(t, entries)
}

func TestStable_LeavesOutAPreReleaseTagNotFlaggedAsOne(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0-rc.1")

	latest, err := compilers.Latest(context.Background(), srv.Source())
	require.NoError(t, err)

	assert.Equal(t, "0.13.1", latest.Version)
}

func TestEnsure_MakesNoRequestForAnInstalledVersion(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	dir := newDir(t)
	_, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")
	require.NoError(t, err)
	before := len(srv.Requests())

	installed, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")
	require.NoError(t, err)

	assert.False(t, installed)
	assert.Len(t, srv.Requests(), before)
}

func TestEnsure_FailsForAVersionThatWasNeverReleased(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	dir := newDir(t)

	_, err := dir.Ensure(context.Background(), srv.Source(), "0.99.0")

	require.ErrorIs(t, err, compilers.ErrNoRelease)
	assert.ErrorContains(t, err, "0.99.0")
}

func TestEnsure_RefusesAnInvalidVersion(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	dir := newDir(t)

	for _, version := range []string{"", "../0.13.1", "latest", "0.13", "v0.13.1"} {
		_, err := dir.Ensure(context.Background(), srv.Source(), version)
		require.ErrorIs(t, err, compilers.ErrInvalidVersion, version)
	}
	assert.Empty(t, srv.Requests())
}

func TestEnsure_ConcurrentInstallsOfOneVersionBothSucceed(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	srv.Publish("0.13.1")
	dir := newDir(t)

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := dir.Ensure(context.Background(), srv.Source(), "0.13.1")
			errs <- err
		}()
	}
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	assert.Equal(t, compilertest.BinaryContent("0.13.1"), readBinary(t, dir, "0.13.1"))
	versions, err := dir.Installed()
	require.NoError(t, err)
	assert.Equal(t, []string{"0.13.1"}, versions, "a staging directory was left behind")
}

func TestInstalled_ListsInstalledVersionsInVersionOrder(t *testing.T) {
	t.Parallel()
	srv := compilertest.New(t)
	for _, v := range []string{"0.10.0", "0.9.0", "0.14.0-rc.10", "0.14.0-rc.2", "0.14.0"} {
		srv.Publish(v)
	}
	dir := newDir(t)
	for _, v := range []string{"0.10.0", "0.14.0", "0.9.0", "0.14.0-rc.10", "0.14.0-rc.2"} {
		_, err := dir.Ensure(context.Background(), srv.Source(), v)
		require.NoError(t, err)
	}

	versions, err := dir.Installed()
	require.NoError(t, err)

	assert.Equal(t, []string{"0.9.0", "0.10.0", "0.14.0-rc.2", "0.14.0-rc.10", "0.14.0"}, versions)
}

func TestInstalled_IsEmptyWithoutACompilerDirectory(t *testing.T) {
	t.Parallel()
	versions, err := newDir(t).Installed()

	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestInstalled_IgnoresDirectoriesWithoutABinary(t *testing.T) {
	t.Parallel()
	dir := newDir(t)
	require.NoError(t, os.MkdirAll(dir.Path("0.13.1"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(dir.Root, ".0.13.1-123"), 0o750))

	versions, err := dir.Installed()
	require.NoError(t, err)

	assert.Empty(t, versions)
}

func TestBinary_FailsForAVersionNotInstalled(t *testing.T) {
	t.Parallel()
	_, err := newDir(t).Binary("0.13.1")

	require.ErrorIs(t, err, compilers.ErrNotInstalled)
}

//nolint:paralleltest // Isolate sets the process environment
func TestOpenDir_LivesBesideTheFontDirectory(t *testing.T) {
	data := filepath.Dir(testrepo.Isolate(t))
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")

	dir, err := compilers.OpenDir()
	require.NoError(t, err)
	_, err = dir.Ensure(context.Background(), compilers.Default(), "0.13.1")
	require.NoError(t, err)

	fontsDir, err := paths.GotpmFontsDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(filepath.Dir(fontsDir), "compilers"), dir.Root)
	assert.True(t, strings.HasPrefix(dir.Root, data), "%q is outside the isolated data directory", dir.Root)
	assert.Equal(t, compilertest.BinaryContent("0.13.1"), readBinary(t, dir, "0.13.1"))
}

func TestNewestStable(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		versions []string
		want     string
	}{
		"nothing":            {nil, ""},
		"only pre-releases":  {[]string{"0.14.0-rc.1", "0.15.0-beta.2"}, ""},
		"newest wins":        {[]string{"0.9.0", "0.13.1", "0.10.0"}, "0.13.1"},
		"pre-release passed": {[]string{"0.13.1", "0.14.0-rc.1"}, "0.13.1"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, compilers.NewestStable(tt.versions))
		})
	}
}
