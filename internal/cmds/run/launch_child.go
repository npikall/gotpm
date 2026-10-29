package run

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

// launchChild runs the compiler as a child process attached to gotpm's stdin,
// stdout and stderr, and exits with its exit code. The child shares gotpm's
// console, so Ctrl-C reaches it directly; gotpm ignores the interrupt and
// waits, leaving it to the compiler how to stop, as when it runs on its own.
func launchChild(l Launch) error {
	cmd := exec.Command(l.Binary, l.Args...) //nolint: gosec,noctx // running the chosen compiler is the point; only an interrupt may stop it
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = l.Env

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	err := cmd.Run()
	signal.Stop(interrupts)

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return fmt.Errorf("could not run compiler %q: %w", l.Binary, err)
	}
	code := cmd.ProcessState.ExitCode()
	if code < 0 {
		code = 1 // killed by a signal
	}
	os.Exit(code)
	return nil
}
