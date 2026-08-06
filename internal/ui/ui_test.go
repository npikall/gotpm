package ui_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
)

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
