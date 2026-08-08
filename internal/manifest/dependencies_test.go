package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// write puts content in a temporary typst.toml and returns its path.
func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "typst.toml")
	require.NoError(t, paths.WriteFile(path, []byte(content)))
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

// setDeps runs the edit and returns the resulting document.
func setDeps(t *testing.T, content string, deps []string) string {
	t.Helper()
	path := write(t, content)
	require.NoError(t, manifest.SetDependencies(path, deps))
	return read(t, path)
}

const commented = `# my favourite package
[package]
name = "foo"          # the import name
version = "0.1.0"
entrypoint = "lib.typ"
exclude = ["docs", "tests"]

# only relevant for templates
[template]
path = "template"
entrypoint = "main.typ"
`

func TestSetDependencies_AppendsASectionWithoutDisturbingTheDocument(t *testing.T) {
	t.Parallel()

	got := setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, commented+`
[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
`, got, "everything the user wrote must survive byte for byte")
}

func TestSetDependencies_KeepsCommentsAndOrderOnRewrite(t *testing.T) {
	t.Parallel()

	once := setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"})
	twice := setDeps(t, once, []string{"@gotpm/cetz:0.3.1", "@gotpm/oxifmt:0.2.1"})

	assert.Contains(t, twice, `name = "foo"          # the import name`,
		"trailing comments and their alignment must be preserved")
	assert.Contains(t, twice, "# only relevant for templates")
	assert.Contains(t, twice, `dependencies = [
  "@gotpm/cetz:0.3.1",
  "@gotpm/oxifmt:0.2.1",
]`)
	assert.Less(t, strings.Index(twice, "[package]"), strings.Index(twice, "[template]"),
		"key order must not be reshuffled the way the TOML encoder would")
}

func TestSetDependencies_ReplacesAnExistingArrayInPlace(t *testing.T) {
	t.Parallel()

	before := `[package]
name = "foo"

[tool.gotpm]
# pinned by hand
dependencies = [
  "@gotpm/cetz:0.3.1",
  "@gotpm/old:0.1.0",
]

[template]
path = "template"
`

	got := setDeps(t, before, []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, `[package]
name = "foo"

[tool.gotpm]
# pinned by hand
dependencies = [
  "@gotpm/cetz:0.3.1",
]

[template]
path = "template"
`, got)
}

func TestSetDependencies_ReplacesASingleLineArray(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[tool.gotpm]
dependencies = ["@gotpm/old:0.1.0"]
`, []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, `[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
`, got)
}

func TestSetDependencies_InsertsIntoASectionThatHasOtherKeys(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[tool.gotpm]
something = "else"
`, []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, `[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
something = "else"
`, got)
}

func TestSetDependencies_LeavesOtherToolSectionsAlone(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[package]
name = "foo"

[tool.othertool]
setting = true
`, []string{"@gotpm/cetz:0.3.1"})

	assert.Contains(t, got, "[tool.othertool]\nsetting = true")
	assert.Contains(t, got, "[tool.gotpm]")
}

func TestSetDependencies_EmptyRemovesTheSection(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[package]
name = "foo"

[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
`, nil)

	assert.Equal(t, `[package]
name = "foo"
`, got, "a section left holding nothing is removed with its blank line")
}

func TestSetDependencies_EmptyKeepsASectionThatStillHasContent(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[tool.gotpm]
# keep me
dependencies = ["@gotpm/cetz:0.3.1"]
other = 1
`, nil)

	assert.Equal(t, `[tool.gotpm]
# keep me
other = 1
`, got)
}

func TestSetDependencies_EmptyOnAManifestWithoutTheSectionIsANoOp(t *testing.T) {
	t.Parallel()

	got := setDeps(t, commented, nil)

	assert.Equal(t, commented, got)
}

func TestSetDependencies_HandlesAFileWithoutATrailingNewline(t *testing.T) {
	t.Parallel()

	got := setDeps(t, `[package]
name = "foo"`, []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, `[package]
name = "foo"

[tool.gotpm]
dependencies = [
  "@gotpm/cetz:0.3.1",
]
`, got)
}

func TestSetDependencies_KeepsWindowsLineEndings(t *testing.T) {
	t.Parallel()

	got := setDeps(t, "[package]\r\nname = \"foo\"\r\n", []string{"@gotpm/cetz:0.3.1"})

	assert.Equal(t, "[package]\r\nname = \"foo\"\r\n\r\n[tool.gotpm]\r\ndependencies = [\r\n  \"@gotpm/cetz:0.3.1\",\r\n]\r\n", got)
}

func TestSetDependencies_RefusesAnInlineToolSection(t *testing.T) {
	t.Parallel()

	path := write(t, `[tool]
gotpm = { dependencies = ["@gotpm/cetz:0.3.1"] }
`)

	err := manifest.SetDependencies(path, []string{"@gotpm/oxifmt:0.2.1"})
	require.ErrorIs(t, err, manifest.ErrInlineToolSection,
		"appending a second [tool.gotpm] would make the document invalid TOML")
}

func TestSetDependencies_ProducesTOMLThatParsesBack(t *testing.T) {
	t.Parallel()

	deps := []string{"@gotpm/cetz:0.3.1", "@gotpm/oxifmt:0.2.1"}
	got := setDeps(t, commented, deps)

	var m manifest.Manifest
	require.NoError(t, toml.Unmarshal([]byte(got), &m))
	assert.Equal(t, deps, m.Dependencies())
	assert.Equal(t, "foo", m.Package.Name)
	assert.Equal(t, []string{"docs", "tests"}, m.Package.Exclude)
	assert.Equal(t, "template", m.Template.Path)
}

func TestParseDependencies_AcceptsTheGotpmNamespace(t *testing.T) {
	t.Parallel()

	refs, err := manifest.ParseDependencies([]string{"@gotpm/cetz:0.3.1"})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "cetz", refs[0].Name)
	assert.Equal(t, "0.3.1", refs[0].Version.String())
}

func TestParseDependencies_RejectsOtherNamespaces(t *testing.T) {
	t.Parallel()

	for _, dep := range []string{"@preview/cetz:0.3.1", "@local/cetz:0.3.1"} {
		_, err := manifest.ParseDependencies([]string{dep})
		require.ErrorIs(t, err, manifest.ErrInvalidDependency, dep)
	}
}

func TestParseDependencies_ReportsEveryBadEntry(t *testing.T) {
	t.Parallel()

	_, err := manifest.ParseDependencies([]string{"nonsense", "@gotpm/cetz:0.3.1", "@preview/x:1.0.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonsense")
	assert.Contains(t, err.Error(), "@preview/x:1.0.0")
}
