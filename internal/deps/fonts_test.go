package deps_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pinOf(t *testing.T, srv *fonttest.Server, family string) lockfile.Font {
	t.Helper()
	pin, err := srv.Source().Resolve(context.Background(), family)
	require.NoError(t, err)
	result, err := fonts.Dir{Root: t.TempDir()}.Ensure(context.Background(), srv.Source(), pin, false)
	require.NoError(t, err)
	return result.Pin
}

func TestEnsureFonts_InstallsAndReportsWhatChanged(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	srv.Publish("ofl", "inter", "Inter.ttf")
	dir, err := fonts.OpenDir()
	require.NoError(t, err)
	mine := filepath.Join(dir.Path("inter"), "Mine.ttf")
	require.NoError(t, paths.EnsureDir(filepath.Dir(mine)))
	require.NoError(t, os.WriteFile(mine, []byte("mine"), paths.FilePerm))
	pins := []lockfile.Font{pinOf(t, srv, "Lato"), pinOf(t, srv, "Inter")}

	results, err := deps.EnsureFonts(pins, false, discardLogger())

	require.NoError(t, err)
	assert.Equal(t, 1, results.Report(), "the hand-installed family is skipped, not counted")
	again, err := deps.EnsureFonts(pins, false, discardLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, again.Report())
}

func TestEnsureFonts_WithoutPinsDoesNothing(t *testing.T) {
	t.Parallel()

	results, err := deps.EnsureFonts(nil, false, discardLogger())

	require.NoError(t, err)
	assert.Equal(t, 0, results.Report())
}

func TestEnsureFonts_FailsWithoutADataDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("APPDATA", "")

	_, err := deps.EnsureFonts([]lockfile.Font{{Family: "Lato"}}, false, discardLogger())

	require.Error(t, err)
}
