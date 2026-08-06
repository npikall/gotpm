package ui_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestImport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		namespace, name, version string
	}{
		{"preview", "my-pkg", "0.1.0"},
		{"local", "foo", "1.2.3"},
		{"preview", "bar", "0.0.1"},
	}
	for _, tt := range tests {
		got := ui.Import(tt.namespace, tt.name, tt.version)
		expected := "@" + tt.namespace + "/" + tt.name + ":" + tt.version
		assert.Contains(t, got, expected,
			"Import(%q, %q, %q) = %q, must contain %q",
			tt.namespace, tt.name, tt.version, got, expected)
	}
}

func TestSpinner_DefaultSuffix(t *testing.T) {
	t.Parallel()
	assert.Contains(t, ui.Spinner("").Suffix, "Loading")
	assert.Contains(t, ui.Spinner(" Cloning...").Suffix, "Cloning")
}
