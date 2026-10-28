//go:build unix

package run

import (
	"fmt"
	"syscall"
)

// launch replaces the gotpm process with the compiler, so signals, the
// terminal and the exit code are the compiler's own.
func launch(l Launch) error {
	argv := append([]string{l.Binary}, l.Args...)
	if err := syscall.Exec(l.Binary, argv, l.Env); err != nil { //nolint: gosec // running the chosen compiler is the point
		return fmt.Errorf("could not run compiler %q: %w", l.Binary, err)
	}
	return nil
}
