package ui

import (
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

const defaultSpinnerSuffix = " Loading..."

// Spinner returns a spinner ready to start. An empty suffix falls back to a
// generic loading message.
func Spinner(suffix string) *spinner.Spinner {
	if suffix == "" {
		suffix = defaultSpinnerSuffix
	}
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) //nolint: mnd
	s.Suffix = Muted.Render(suffix)
	_ = s.Color("cyan")
	return s
}

// WithSpinner runs fn with a spinner showing, and stops the spinner whichever
// way fn leaves.
//
// The spinner owns the terminal while it runs, so anything written from inside
// fn is garbled by it: work that has something to say has to collect it and say
// it afterwards.
func WithSpinner[T any](suffix string, fn func() (T, error)) (T, error) { //nolint: ireturn // T is the caller's own result type, not an abstraction being returned
	s := Spinner(" " + strings.TrimSpace(suffix))
	defer s.Stop()
	s.Start()
	return fn()
}

// Spin is WithSpinner for work that only reports whether it succeeded.
func Spin(suffix string, fn func() error) error {
	_, err := WithSpinner(suffix, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
