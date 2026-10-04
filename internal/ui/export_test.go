package ui

import "testing"

// Indicator is the part of a spinner the package drives, exported for tests.
type Indicator = indicator

// UseIndicator makes every spinner started during the test come from build,
// and restores the real spinner when the test ends.
func UseIndicator(t *testing.T, build func(suffix string) Indicator) {
	t.Helper()
	previous := newIndicator
	newIndicator = build
	t.Cleanup(func() { newIndicator = previous })
}
