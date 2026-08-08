package ui

import (
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
