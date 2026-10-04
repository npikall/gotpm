package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// answer makes content what Confirm reads from stdin for the rest of the test.
func answer(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdin")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	stdin, err := os.Open(path)
	require.NoError(t, err)
	previous := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() {
		os.Stdin = previous
		_ = stdin.Close()
	})
}

func TestConfirm(t *testing.T) { //nolint: paralleltest
	for content, want := range map[string]bool{
		"yes\n": true,
		"y":     true, // no newline: the answer ends at EOF
		"\n":    false,
		"":      false,
	} {
		answer(t, content)
		got, err := ui.Confirm("proceed?")
		require.NoError(t, err)
		assert.Equal(t, want, got, "answer %q", content)
	}
}

func TestConfirm_ReportsAnUnreadableStdin(t *testing.T) { //nolint: paralleltest
	answer(t, "")
	require.NoError(t, os.Stdin.Close())

	_, err := ui.Confirm("proceed?")

	require.Error(t, err)
}
