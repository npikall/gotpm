// Package hooks runs the shell commands a manifest attaches to a command, such
// as the pre- and post-publish hooks. Commands are interpreted by a portable
// POSIX shell, so they behave the same on every platform.
package hooks

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/npikall/gotpm/internal/ui"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Run runs cmds one after another in dir, writing each command as a shell
// prompt (see ui.Command) and then its output to out. It stops at the first
// command that fails.
func Run(ctx context.Context, dir string, cmds []string, out io.Writer) error {
	for _, cmd := range cmds {
		if err := runOne(ctx, dir, cmd, out); err != nil {
			return fmt.Errorf("%q: %w", cmd, err)
		}
	}
	return nil
}

func runOne(ctx context.Context, dir, cmd string, out io.Writer) error {
	if err := echoCmd(cmd, out); err != nil {
		return err
	}
	file, runner, err := parseCmd(dir, cmd, out)
	if err != nil {
		return err
	}

	if err := runner.Run(ctx, file); err != nil {
		return fmt.Errorf("running: %w", err)
	}
	return nil
}

func echoCmd(cmd string, out io.Writer) error {
	if _, err := fmt.Fprintln(out, ui.Command(cmd)); err != nil {
		return fmt.Errorf("echoing: %w", err)
	}
	return nil
}

func parseCmd(dir, cmd string, out io.Writer) (*syntax.File, *interp.Runner, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return nil, nil, fmt.Errorf("parsing: %w", err)
	}
	runner, err := interp.New(interp.Dir(dir), interp.StdIO(nil, out, out))
	if err != nil {
		return nil, nil, fmt.Errorf("creating shell: %w", err)
	}
	return file, runner, nil
}
