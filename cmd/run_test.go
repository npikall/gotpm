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

// runThroughCLI executes the root command with args, outside any project,
// and returns what the launcher was asked to run.
func runThroughCLI(t *testing.T, args ...string) []run.Launch {
	t.Helper()
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())
	srv := compilertest.Serve(t)
	srv.Publish("0.14.0")
	var launches []run.Launch
	t.Cleanup(run.UseLauncher(func(l run.Launch) error {
		launches = append(launches, l)
		return nil
	}))
	rootCmd.SetArgs(args)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	require.NoError(t, rootCmd.Execute())
	return launches
}
