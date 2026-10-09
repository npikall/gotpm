package manifest_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setFonts(t *testing.T, content string, fonts []string) string {
	t.Helper()
	path := write(t, content)
	require.NoError(t, manifest.SetFonts(path, fonts))
	return read(t, path)
}

func TestSetFonts_SitsBesideTheDependencies(t *testing.T) {
	t.Parallel()
	withDeps := setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"})

	got := setFonts(t, withDeps, []string{"Open Sans"})

	assert.Equal(t, commented+`
[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
fonts = [
  "Open Sans",
]
`, got)
	m, err := manifest.LoadFile(write(t, got))
	require.NoError(t, err)
	assert.Equal(t, []string{"Open Sans"}, m.Fonts())
	assert.Equal(t, []string{"@gotpm/cetz:0.3.1"}, m.Dependencies())
}

func TestSetFonts_EmptyLeavesTheDependencies(t *testing.T) {
	t.Parallel()
	both := setFonts(t, setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"}), []string{"Lato"})

	got := setFonts(t, both, nil)

	assert.Equal(t, setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"}), got)
}

func TestSetDependencies_EmptyKeepsTheFonts(t *testing.T) {
	t.Parallel()
	both := setDeps(t, setFonts(t, commented, []string{"Lato"}), []string{"@gotpm/cetz:0.3.1"})

	got := setDeps(t, both, nil)

	assert.Equal(t, setFonts(t, commented, []string{"Lato"}), got)
}
