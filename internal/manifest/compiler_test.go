package manifest_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setPin(t *testing.T, content, version string) string {
	t.Helper()
	path := write(t, content)
	require.NoError(t, manifest.SetCompilerPin(path, version))
	return read(t, path)
}

func TestLoad_ExposesTheCompilerPinBesideTheRequirement(t *testing.T) {
	t.Parallel()

	m, err := manifest.LoadFile(write(t, `[package]
name = "foo"
version = "0.1.0"
entrypoint = "lib.typ"
compiler = "0.12.0"

[tool.gotpm]
compiler-pin = "0.13.1"
`))

	require.NoError(t, err)
	assert.Equal(t, "0.12.0", m.Package.Compiler)
	assert.Equal(t, "0.13.1", m.CompilerPin())
}

func TestLoad_CompilerPinIsEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	m, err := manifest.LoadFile(write(t, commented))

	require.NoError(t, err)
	assert.Empty(t, m.CompilerPin())
}

func TestSetCompilerPin_AddsTheSectionWithoutDisturbingTheDocument(t *testing.T) {
	t.Parallel()

	got := setPin(t, commented, "0.13.1")

	assert.Equal(t, commented+`
[tool.gotpm]
compiler-pin = "0.13.1"
`, got, "everything the user wrote must survive byte for byte")
}

func TestSetCompilerPin_AddsTheKeyToAnExistingSection(t *testing.T) {
	t.Parallel()
	withDeps := setDeps(t, commented, []string{"@gotpm/cetz:0.3.1"})

	got := setPin(t, withDeps, "0.13.1")

	assert.Equal(t, withDeps+`compiler-pin = "0.13.1"
`, got)
}

func TestSetCompilerPin_ReplacesOnlyThePin(t *testing.T) {
	t.Parallel()

	before := `[package]
name = "foo"          # the import name
compiler = "0.12.0"

[tool.gotpm]
# chosen by run
compiler-pin = "0.12.0"   # keep in step with CI
fonts = ["Lato"]

[template]
path = "template"
`

	got := setPin(t, before, "0.14.0")

	assert.Equal(t, `[package]
name = "foo"          # the import name
compiler = "0.12.0"

[tool.gotpm]
# chosen by run
compiler-pin = "0.14.0"   # keep in step with CI
fonts = ["Lato"]

[template]
path = "template"
`, got)
}

func TestSetCompilerPin_SameValueLeavesTheFileUntouched(t *testing.T) {
	t.Parallel()
	before := `[tool.gotpm]
compiler-pin = '0.13.1'   # literal string, odd spacing
`

	got := setPin(t, before, "0.13.1")

	assert.Equal(t, before, got)
}

func TestSetCompilerPin_WritesIntoAnInlineTable(t *testing.T) {
	t.Parallel()

	got := setPin(t, `[tool]
gotpm = { fonts = ["Lato"] }
`, "0.13.1")

	assert.Equal(t, `[tool]
gotpm = { fonts = ["Lato"], compiler-pin = "0.13.1" }
`, got)
}

func TestSetCompilerPin_RejectsAPinThatIsNotAString(t *testing.T) {
	t.Parallel()
	before := `[tool.gotpm]
compiler-pin = 13
`
	path := write(t, before)

	err := manifest.SetCompilerPin(path, "0.13.1")

	require.ErrorIs(t, err, manifest.ErrInvalidManifest)
	assert.Equal(t, before, read(t, path), "a rejected edit must not touch the file")
}
