package fontname_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/fontname"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"Open Sans":           "opensans",
		"opensans":            "opensans",
		"IBM Plex Mono":       "ibmplexmono",
		"Source Serif 4":      "sourceserif4",
		"M PLUS 1p":           "mplus1p",
		"Zen Kaku Gothic-New": "zenkakugothicnew",
		"Café":                "caf",
		"  -- ":               "",
	}
	for name, want := range tests {
		assert.Equal(t, want, fontname.Key(name), name)
	}
}
