package ui_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timeline records spinner starts and stops and terminal writes in the order
// they happen.
type timeline struct {
	events []string
}

func (tl *timeline) Write(p []byte) (int, error) {
	tl.events = append(tl.events, "write "+string(p))
	return len(p), nil
}

func (tl *timeline) indicator(suffix string) ui.Indicator { //nolint: ireturn // matches the factory the package swaps in
	tl.events = append(tl.events, "new"+suffix)
	return fakeIndicator{tl}
}

type fakeIndicator struct{ tl *timeline }

func (f fakeIndicator) Start() { f.tl.events = append(f.tl.events, "start") }
func (f fakeIndicator) Stop()  { f.tl.events = append(f.tl.events, "stop") }

//nolint:paralleltest // swaps the package's spinner
func TestPausing_ReplacesTheSpinnerAroundAWrite(t *testing.T) {
	tl := &timeline{}
	ui.UseIndicator(t, tl.indicator)
	out := ui.Pausing(tl)

	require.NoError(t, ui.Spin("working", func() error {
		_, err := out.Write([]byte("warn"))
		return err //nolint: wrapcheck // the fake writer never fails
	}))

	assert.Equal(t, []string{
		"new working", "start",
		"stop", "write warn", "new working", "start",
		"stop",
	}, tl.events, "a stopped spinner is not restarted: a fresh one takes its place")
}

//nolint:paralleltest // swaps the package's spinner
func TestPausing_WritesStraightThroughWithoutASpinner(t *testing.T) {
	tl := &timeline{}
	ui.UseIndicator(t, tl.indicator)

	_, err := ui.Pausing(tl).Write([]byte("warn"))

	require.NoError(t, err)
	assert.Equal(t, []string{"write warn"}, tl.events)
}

//nolint:paralleltest // swaps the package's spinner
func TestSpin_InsideASpinKeepsTheOuterSpinner(t *testing.T) {
	tl := &timeline{}
	ui.UseIndicator(t, tl.indicator)

	require.NoError(t, ui.Spin("outer", func() error {
		return ui.Spin("inner", func() error { return nil })
	}))

	assert.Equal(t, []string{"new outer", "start", "stop"}, tl.events)
}

//nolint:paralleltest // swaps the package's spinner
func TestSpin_FallsBackToAGenericSuffix(t *testing.T) {
	tl := &timeline{}
	ui.UseIndicator(t, tl.indicator)

	require.NoError(t, ui.Spin("", func() error { return nil }))

	assert.Equal(t, []string{"new Loading...", "start", "stop"}, tl.events)
}
