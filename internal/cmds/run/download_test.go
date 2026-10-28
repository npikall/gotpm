package run_test

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_InstallsACompilerWithoutDigestAndWarnsNamingIt(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.10.0", compilertest.NoDigest())
	var out bytes.Buffer
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "0.10.0", nil, &out, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.10.0"), (*launches)[0].Binary)
	assert.Contains(t, out.String(), "warning")
	assert.Contains(t, out.String(), "0.10.0")
	assert.Contains(t, out.String(), "no digest")
}

func TestRun_StaysQuietAboutAVerifiedCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	var out bytes.Buffer
	record(t)

	require.NoError(t, run.Run(context.Background(), "0.14.0", nil, &out, discardLogger()))

	assert.Empty(t, out.String())
}

func TestRun_RefusesACompilerWhoseDigestDoesNotMatch(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0", compilertest.WrongDigest())
	launches := record(t)

	err := run.Run(context.Background(), "", nil, io.Discard, discardLogger())

	require.ErrorIs(t, err, compilers.ErrChecksum)
	assert.Empty(t, *launches)
	installed, err := compilerDir(t).Installed()
	require.NoError(t, err)
	assert.Empty(t, installed)
}

func TestRun_OfflineFailsNamingTheMissingCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project(t, "0.13.0", "0.13.1")
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Unreachable()
	launches := record(t)

	err := run.Run(context.Background(), "", nil, io.Discard, discardLogger())

	require.ErrorIs(t, err, compilers.ErrDownload)
	require.ErrorContains(t, err, "could not download compiler 0.13.1")
	assert.Empty(t, *launches)
}

func TestRun_OfflineWithNothingInstalledSaysTheReleasesCouldNotBeLookedUp(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Unreachable()
	launches := record(t)

	err := run.Run(context.Background(), "", nil, io.Discard, discardLogger())

	require.ErrorContains(t, err, "could not look up compiler releases")
	assert.Empty(t, *launches)
}

func TestRun_OfflineLaunchesAnInstalledCompiler(t *testing.T) { //nolint: paralleltest
	for name, setup := range map[string]func(t *testing.T){ //nolint: paralleltest
		"outside a project":    func(t *testing.T) { t.Helper(); outside(t) },
		"inside a project":     func(t *testing.T) { t.Helper(); project(t, "0.13.0", "0.13.1") },
		"pinning in a project": func(t *testing.T) { t.Helper(); project(t, "0.13.0", "") },
	} {
		t.Run(name, func(t *testing.T) {
			testrepo.Isolate(t)
			setup(t)
			srv := compilertest.Serve(t)
			srv.Publish("0.13.1")
			install(t, srv, "0.13.1")
			srv.Unreachable()
			launches := record(t)

			require.NoError(t, run.Run(context.Background(), "", nil, io.Discard, discardLogger()))

			require.Len(t, *launches, 1)
			assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary)
		})
	}
}

func TestRun_FailsNamingThePlatformTypstShipsNoCompilerFor(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0", compilertest.NoAssetForThisPlatform())
	launches := record(t)

	err := run.Run(context.Background(), "", nil, io.Discard, discardLogger())

	require.ErrorIs(t, err, compilers.ErrUnsupportedPlatform)
	require.ErrorContains(t, err, runtime.GOOS+"/"+runtime.GOARCH)
	assert.Empty(t, *launches)
}
