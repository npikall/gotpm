package cmd //nolint: testpackage

import (
	"testing"

	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHandsFlagsAfterTheFirstArgumentToTypst(t *testing.T) { //nolint: paralleltest // rootCmd is shared
	launches := runThroughCLI(t, "run", "-v", "compile", "--help", "-v", "main.typ")

	require.Len(t, launches, 1)
	assert.Equal(t, []string{"compile", "--help", "-v", "main.typ"}, launches[0].Args)
}

func TestRunHandsFlagsAfterADoubleDashToTypst(t *testing.T) { //nolint: paralleltest // rootCmd is shared
	launches := runThroughCLI(t, "run", "--", "--version")

	require.Len(t, launches, 1)
	assert.Equal(t, []string{"--version"}, launches[0].Args)
}

func TestRunTakesTheCompilerFromTypstBeforeTheFirstArgument(t *testing.T) { //nolint: paralleltest // rootCmd is shared
	launches := runThroughCLI(t, "run", "--typst", "0.13.1", "compile", "--typst", "x", "main.typ")

	require.Len(t, launches, 1)
	assert.Contains(t, launches[0].Binary, "0.13.1", "not the latest release, 0.14.0")
	assert.Equal(t, []string{"compile", "--typst", "x", "main.typ"}, launches[0].Args,
		"a --typst after the first argument is typst's")
}

func TestRunRefusesAnEmptyTypst(t *testing.T) {
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())
	t.Cleanup(run.UseLauncher(func(run.Launch) error { return nil }))
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		runCmd.Flags().Lookup("typst").Changed = false
	})
	rootCmd.SetArgs([]string{"run", "--typst=", "compile", "main.typ"})

	require.ErrorIs(t, rootCmd.Execute(), errNoCompilerNamed)
}

// runThroughCLI executes the root command with args, outside any project,
// and returns what the launcher was asked to run.
func runThroughCLI(t *testing.T, args ...string) []run.Launch {
	t.Helper()
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	var launches []run.Launch
	t.Cleanup(run.UseLauncher(func(l run.Launch) error {
		launches = append(launches, l)
		return nil
	}))
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		require.NoError(t, runCmd.Flags().Set("typst", ""))
		runCmd.Flags().Lookup("typst").Changed = false
	})

	require.NoError(t, rootCmd.Execute())
	return launches
}
