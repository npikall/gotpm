package ui

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

const defaultSpinnerSuffix = "Loading..."

// indicator is the part of a spinner this package drives.
type indicator interface {
	Start()
	Stop()
}

var (
	// terminal guards active and every write that goes through Pausing, so a
	// write never lands while the spinner is drawing.
	terminal sync.Mutex
	// active is the spinner currently showing, or nil. Only one shows at a time.
	active indicator
	// activeSuffix is what active shows, so a paused spinner can be rebuilt.
	activeSuffix string
	// newIndicator builds the spinner WithSpinner shows; tests swap it.
	newIndicator = func(suffix string) indicator { return newSpinner(suffix) }
)

func newSpinner(suffix string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriterFile(os.Stderr)) //nolint: mnd
	s.Suffix = Muted.Render(suffix)
	_ = s.Color("cyan")
	return s
}

// WithSpinner runs fn with a spinner showing, and stops the spinner whichever way
// fn leaves. An empty suffix falls back to a generic loading message. Inside a
// spinner that is already showing, fn just runs under the outer one.
func WithSpinner[T any](suffix string, fn func() (T, error)) (T, error) { //nolint: ireturn // T is the caller's own result type, not an abstraction being returned
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = defaultSpinnerSuffix
	}

	terminal.Lock()
	if active != nil {
		terminal.Unlock()
		return fn()
	}
	activeSuffix = " " + suffix
	active = newIndicator(activeSuffix)
	active.Start()
	terminal.Unlock()

	defer func() {
		terminal.Lock()
		defer terminal.Unlock()
		active.Stop()
		active = nil
	}()
	return fn()
}

// Spin is WithSpinner for work that only reports whether it succeeded.
func Spin(suffix string, fn func() error) error {
	_, err := WithSpinner(suffix, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

// Pausing wraps w so that each write takes the spinner off the screen, if one
// is showing, and puts it back afterwards. Everything that reaches the terminal
// while a spinner may be showing must go through it.
func Pausing(w io.Writer) io.Writer {
	return pausingWriter{w}
}

type pausingWriter struct {
	w io.Writer
}

func (p pausingWriter) Write(b []byte) (int, error) {
	terminal.Lock()
	defer terminal.Unlock()
	if active == nil {
		return p.w.Write(b) //nolint: wrapcheck // a writer passes its target's errors through
	}
	active.Stop()
	n, err := p.w.Write(b)
	// A stopped spinner is not started again: its drawing goroutine may still
	// be winding down and would swallow the new one's stop signal. A fresh
	// spinner has its own.
	active = newIndicator(activeSuffix)
	active.Start()
	return n, err //nolint: wrapcheck // a writer passes its target's errors through
}
