package ui_test

import (
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestAffirmative(t *testing.T) {
	t.Parallel()
	answers := map[string]bool{
		"y\n":       true,
		"Y\n":       true,
		"yes\n":     true,
		"  YES  \n": true,
		"n\n":       false,
		"no\n":      false,
		"\n":        false,
		"":          false,
		"yeah":      false,
		"ye":        false,
	}

	for answer, want := range answers {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, want, ui.Affirmative(answer), "answer %q", answer)
		})
	}
}
