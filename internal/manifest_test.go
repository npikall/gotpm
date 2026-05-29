package internal_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(content), FilePerm))
	return dir
}

const validManifestTOML = `
[package]
name = "my-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
`

func TestLoadManifest(t *testing.T) {
	t.Parallel()
	t.Run("valid manifest", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		m, err := LoadManifest(dir)
		require.NoError(t, err)
		assert.Equal(t, "my-pkg", m.Package.Name)
		assert.Equal(t, "0.1.0", m.Package.Version)
		assert.Equal(t, "lib.typ", m.Package.Entrypoint)
	})
	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		_, err := LoadManifest(t.TempDir())
		assert.ErrorIs(t, err, ErrManifestNotFound)
	})
	t.Run("invalid TOML", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "not { valid ] toml ::::")
		_, err := LoadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing name", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nversion=\"0.1.0\"\nentrypoint=\"lib.typ\"\n")
		_, err := LoadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing version", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nname=\"pkg\"\nentrypoint=\"lib.typ\"\n")
		_, err := LoadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing entrypoint", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nname=\"pkg\"\nversion=\"0.1.0\"\n")
		_, err := LoadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("all fields missing", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\n")
		_, err := LoadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
}

func TestPackageMeta_ValidateVersion(t *testing.T) {
	t.Parallel()
	assert.True(t, (&PackageMeta{Version: "1.2.3"}).ValidateVersion())
	assert.True(t, (&PackageMeta{Version: "0.0.0"}).ValidateVersion())
	assert.False(t, (&PackageMeta{Version: ""}).ValidateVersion())
	assert.False(t, (&PackageMeta{Version: "v1.0.0"}).ValidateVersion())
	assert.False(t, (&PackageMeta{Version: "1.2"}).ValidateVersion())
}

func TestPackageMeta_Bump(t *testing.T) {
	t.Parallel()
	t.Run("exact semver sets version directly", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "0.1.0"}
		require.NoError(t, p.Bump("2.3.4"))
		assert.Equal(t, "2.3.4", p.Version)
	})
	t.Run("patch increment", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "1.2.3"}
		require.NoError(t, p.Bump(PATCH))
		assert.Equal(t, "1.2.4", p.Version)
	})
	t.Run("minor increment resets patch", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "1.2.3"}
		require.NoError(t, p.Bump(MINOR))
		assert.Equal(t, "1.3.0", p.Version)
	})
	t.Run("major increment resets minor and patch", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "1.2.3"}
		require.NoError(t, p.Bump(MAJOR))
		assert.Equal(t, "2.0.0", p.Version)
	})
	t.Run("invalid current version returns error", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "not-a-version"}
		assert.Error(t, p.Bump(PATCH))
	})
	t.Run("invalid increment returns error", func(t *testing.T) {
		t.Parallel()
		p := &PackageMeta{Version: "1.0.0"}
		assert.ErrorIs(t, p.Bump("bogus"), ErrInvalidIncrement)
	})
}
