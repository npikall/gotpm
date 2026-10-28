//go:build unix

package run //nolint: testpackage // launchChild backs the windows launcher, which tests otherwise swap out

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

const childHelperEnv = "GOTPM_RUN_LAUNCH_CHILD_HELPER"

// TestLaunchChild_SurvivesAnInterruptAndExitsWithTheChildsCode re-runs the
// test binary as a helper process that launches a shell as its child. The
// shell reports the environment it was handed on stdout and stderr, then waits
// for a line on stdin. The helper is interrupted meanwhile; that it still
// exits with the shell's code shows gotpm waited for the compiler to decide
// how to stop instead of dying first.
func TestLaunchChild_SurvivesAnInterruptAndExitsWithTheChildsCode(t *testing.T) {
	if os.Getenv(childHelperEnv) != "" {
		_ = launchChild(Launch{
			Binary: "/bin/sh",
			Args:   []string{"-c", `echo "ready $GREETING"; echo oops >&2; read line; exit $CODE`},
			Env:    []string{"GREETING=hi", "CODE=7"},
		})
		os.Exit(1)
	}
	t.Parallel()
	helper := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLaunchChild_SurvivesAnInterruptAndExitsWithTheChildsCode$")
	helper.Env = append(os.Environ(), childHelperEnv+"=1")
	stdin, err := helper.StdinPipe()
	require.NoError(t, err)
	stdout, err := helper.StdoutPipe()
	require.NoError(t, err)
	stderr, err := helper.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, helper.Start())

	ready, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready hi\n", ready, "the child writes to gotpm's stdout with the environment it was handed")
	require.NoError(t, helper.Process.Signal(syscall.SIGINT))
	_, err = io.WriteString(stdin, "go on\n")
	require.NoError(t, err)
	errOut, err := io.ReadAll(stderr)
	require.NoError(t, err)
	err = helper.Wait()

	require.Equal(t, "oops\n", string(errOut), "the child writes to gotpm's stderr")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "the helper exits with the shell's code")
	require.Equal(t, 7, exitErr.ExitCode())
}

func TestLaunchChild_FailsForAMissingBinary(t *testing.T) {
	t.Parallel()
	require.Error(t, launchChild(Launch{Binary: "/nonexistent/typst"}))
}
