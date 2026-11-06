package run_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_TypstOutsideAProjectLaunchesExactlyThatCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	cwd := outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.11.1")
	srv.Publish("0.14.0")
	install(t, srv, "0.14.0")
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "0.11.1", []string{"compile", "notes.typ"}, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.11.1"), (*launches)[0].Binary, "installed first, though a newer one is installed")
	assert.Equal(t, []string{"compile", "notes.typ"}, (*launches)[0].Args)
	assert.Empty(t, entries(t, cwd))
}

func TestRun_TypstInsideAProjectLeavesTheManifestAlone(t *testing.T) { //nolint: paralleltest
	for name, pin := range map[string]string{"with a pin": "0.13.1", "without a pin": ""} { //nolint: paralleltest
		t.Run(name, func(t *testing.T) {
			testrepo.Isolate(t)
			file := project(t, "0.13.0", pin)
			before := readFile(t, file)
			srv := compilertest.Serve(t)
			srv.Publish("0.11.1")
			srv.Publish("0.13.1")
			install(t, srv, "0.13.1")
			var out bytes.Buffer
			launches := record(t)

			require.NoError(t, run.Run(context.Background(), "0.11.1", nil, &out, discardLogger()))

			require.Len(t, *launches, 1)
			assert.Equal(t, binary(t, "0.11.1"), (*launches)[0].Binary, "used though below the floor")
			assert.Equal(t, before, readFile(t, file), "the manifest is not touched")
			assert.Empty(t, out.String())
		})
	}
}

func TestRun_TypstNamingAPrereleaseRunsIt(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	srv.Publish("0.15.0-rc.1", compilertest.Prerelease())
	launches := record(t)

	require.NoError(t, run.Run(context.Background(), "0.15.0-rc.1", nil, io.Discard, discardLogger()))

	require.Len(t, *launches, 1)
	assert.Equal(t, binary(t, "0.15.0-rc.1"), (*launches)[0].Binary)
}

func TestRun_TypstNamingAVersionWithoutAReleaseFails(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	launches := record(t)

	err := run.Run(context.Background(), "0.9.9", nil, io.Discard, discardLogger())

	require.ErrorIs(t, err, compilers.ErrNoRelease)
	assert.Contains(t, err.Error(), "0.9.9")
	assert.Empty(t, *launches)
	installed, err := compilerDir(t).Installed()
	require.NoError(t, err)
	assert.Empty(t, installed)
}
