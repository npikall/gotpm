package hooks_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/npikall/gotpm/internal/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExecutesCommandsInDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var out bytes.Buffer

	err := hooks.Run(t.Context(), dir, []string{"echo hi > a.txt", "cat a.txt"}, &out)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "a.txt"))
	assert.Equal(t, "$ echo hi > a.txt\n$ cat a.txt\nhi\n", ansi.Strip(out.String()))
}

func TestRunEchoesEachCommandBeforeRunningIt(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	err := hooks.Run(t.Context(), t.TempDir(), []string{"echo $((1+1))", "echo $((2+2))"}, &out)

	require.NoError(t, err)
	assert.Equal(t, "$ echo $((1+1))\n2\n$ echo $((2+2))\n4\n", ansi.Strip(out.String()))
}

func TestRunStopsAtFirstFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := hooks.Run(t.Context(), dir, []string{"exit 3", "touch b.txt"}, &bytes.Buffer{})

	require.ErrorContains(t, err, `"exit 3"`)
	assert.NoFileExists(t, filepath.Join(dir, "b.txt"))
}

func TestRunRejectsInvalidSyntax(t *testing.T) {
	t.Parallel()

	err := hooks.Run(t.Context(), t.TempDir(), []string{"echo ("}, &bytes.Buffer{})

	require.ErrorContains(t, err, `"echo ("`)
}

func TestRunWithoutCommands(t *testing.T) {
	t.Parallel()
	require.NoError(t, hooks.Run(t.Context(), t.TempDir(), nil, &bytes.Buffer{}))
}
