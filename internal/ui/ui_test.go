package ui_test

import (
	"errors"
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errBoom stands in for whatever the wrapped work failed with.
var errBoom = errors.New("boom")

func TestPackage(t *testing.T) {
	t.Parallel()
	for _, ref := range []string{"@preview/my-pkg:0.1.0", "@local/foo:1.2.3", "@preview/bar:*.*.*"} {
		assert.Contains(t, ui.Package(ref), ref, "Package(%q) must contain the reference", ref)
	}
}

func TestSpinner_DefaultSuffix(t *testing.T) {
	t.Parallel()
	assert.Contains(t, ui.Spinner("").Suffix, "Loading")
	assert.Contains(t, ui.Spinner(" Cloning...").Suffix, "Cloning")
}

func TestWithSpinner_PassesTheResultThrough(t *testing.T) {
	t.Parallel()
	value, err := ui.WithSpinner("working", func() (string, error) { return "done", nil })

	require.NoError(t, err)
	assert.Equal(t, "done", value)
}

func TestWithSpinner_PassesTheErrorThrough(t *testing.T) {
	t.Parallel()
	value, err := ui.WithSpinner("working", func() (string, error) { return "half", errBoom })

	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, "half", value, "a failing call keeps whatever it returned alongside the error")
}

func TestSpin_PassesTheErrorThrough(t *testing.T) {
	t.Parallel()
	require.NoError(t, ui.Spin("working", func() error { return nil }))
	require.ErrorIs(t, ui.Spin("working", func() error { return errBoom }), errBoom)
}
