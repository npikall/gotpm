package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatImportStmt(t *testing.T) {
	tests := []struct {
		namespace, name, version string
	}{
		{"preview", "my-pkg", "0.1.0"},
		{"local", "foo", "1.2.3"},
		{"preview", "bar", "0.0.1"},
	}
	for _, tt := range tests {
		got := FormatImportStmt(tt.namespace, tt.name, tt.version)
		expected := "@" + tt.namespace + "/" + tt.name + ":" + tt.version
		assert.Contains(t, got, expected,
			"FormatImportStmt(%q, %q, %q) = %q, must contain %q",
			tt.namespace, tt.name, tt.version, got, expected)
	}
}
