//go:build unix

package run //nolint: testpackage // launch is the real launcher tests otherwise swap out

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

const helperEnv = "GOTPM_RUN_LAUNCH_HELPER"

// TestLaunch_ReplacesTheProcess re-runs the test binary as a helper process
// that launches a shell. The helper exiting with the shell's code, computed
// from the arguments and environment it was handed, shows the process was
// replaced: had launch returned, the helper would exit with another code.
func TestLaunch_ReplacesTheProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "" {
		_ = launch(Launch{Binary: "/bin/sh", Args: []string{"-c", "exit $CODE"}, Env: []string{"CODE=7"}})
		os.Exit(1)
	}
	t.Parallel()
	helper := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLaunch_ReplacesTheProcess$")
	helper.Env = append(os.Environ(), helperEnv+"=1")

	err := helper.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "the helper exits with the shell's code")
	require.Equal(t, 7, exitErr.ExitCode())
}

func TestLaunch_FailsForAMissingBinary(t *testing.T) {
	t.Parallel()
	require.Error(t, launch(Launch{Binary: "/nonexistent/typst"}))
}
