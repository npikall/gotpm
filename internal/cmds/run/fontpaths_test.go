package run_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fontPathsKey = "TYPST_FONT_PATHS"

const sep = string(os.PathListSeparator)

func fontDir(t *testing.T) string {
	t.Helper()
	dir, err := fonts.OpenDir()
	require.NoError(t, err)
	return dir.Root
}

// unsetFontPaths unsets TYPST_FONT_PATHS until the test ends.
func unsetFontPaths(t *testing.T) {
	t.Helper()
	t.Setenv(fontPathsKey, "")
	require.NoError(t, os.Unsetenv(fontPathsKey))
}

// launchEnv runs the compiler outside a project and returns the environment
// it was launched with.
func launchEnv(t *testing.T) []string {
	t.Helper()
	outside(t)
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	launches := record(t)
	require.NoError(t, run.Run(context.Background(), "", []string{"compile", "main.typ"}, io.Discard, discardLogger()))
	require.Len(t, *launches, 1)
	return (*launches)[0].Env
}

// fontPaths is every TYPST_FONT_PATHS entry in env.
func fontPaths(env []string) []string {
	var found []string
	for _, pair := range env {
		if value, ok := strings.CutPrefix(pair, fontPathsKey+"="); ok {
			found = append(found, value)
		}
	}
	return found
}

func TestRun_SetsTheFontPathsToTheFontDirectoryWhenUnset(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	unsetFontPaths(t)

	env := launchEnv(t)

	assert.Equal(t, []string{fontDir(t)}, fontPaths(env))
}

func TestRun_SetsTheFontPathsToTheFontDirectoryWhenEmpty(t *testing.T) {
	testrepo.Isolate(t)
	t.Setenv(fontPathsKey, "")

	env := launchEnv(t)

	assert.Equal(t, []string{fontDir(t)}, fontPaths(env))
}

func TestRun_AppendsTheFontDirectoryToTheUsersFontPaths(t *testing.T) {
	testrepo.Isolate(t)
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	t.Setenv(fontPathsKey, a+sep+b+sep)

	env := launchEnv(t)

	assert.Equal(t, []string{a + sep + b + sep + fontDir(t)}, fontPaths(env), "with no empty entry")
}

func TestRun_DoesNotAppendTheFontDirectoryTwice(t *testing.T) {
	testrepo.Isolate(t)
	mine := t.TempDir() + sep + fontDir(t) + string(filepath.Separator)
	t.Setenv(fontPathsKey, mine)

	env := launchEnv(t)

	assert.Equal(t, []string{mine}, fontPaths(env))
}

func TestRun_PassesTheRestOfTheEnvironmentThroughUnchanged(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	unsetFontPaths(t)

	env := launchEnv(t)

	assert.ElementsMatch(t, append(os.Environ(), fontPathsKey+"="+fontDir(t)), env)
}

func TestRun_FetchesNoPackageOrFontFamily(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	missing := testrepo.New(t, "missing", "1.0.0").Release().Import()
	file := project(t, "", "0.14.0", missing)
	require.NoError(t, manifest.SetFonts(file, []string{"Lato"}))
	families := fonttest.Serve(t)
	families.Publish("ofl", "lato", "Lato-Regular.ttf")
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	install(t, srv, "0.14.0")

	runIn(t)

	assert.Empty(t, families.Requests(), "no font family is fetched")
	assert.NoDirExists(t, fontDir(t))
	assert.NoDirExists(t, packages, "no package is fetched")
}
