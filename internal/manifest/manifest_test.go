package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validManifestTOML = `
[package]
name = "my-pkg"
version = "0.1.0"
entrypoint = "lib.typ"
`

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(content), paths.FilePerm))
	return dir
}

func TestLoadFrom(t *testing.T) {
	t.Parallel()
	t.Run("valid manifest", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		m, err := manifest.LoadFrom(dir)
		require.NoError(t, err)
		assert.Equal(t, "my-pkg", m.Package.Name)
		assert.Equal(t, "0.1.0", m.Package.Version)
		assert.Equal(t, "lib.typ", m.Package.Entrypoint)
	})
	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.LoadFrom(t.TempDir())
		assert.ErrorIs(t, err, manifest.ErrManifestNotFound)
	})
	t.Run("invalid TOML", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "not { valid ] toml ::::")
		_, err := manifest.LoadFrom(dir)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("missing name", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nversion=\"0.1.0\"\nentrypoint=\"lib.typ\"\n")
		_, err := manifest.LoadFrom(dir)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("missing version", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nname=\"pkg\"\nentrypoint=\"lib.typ\"\n")
		_, err := manifest.LoadFrom(dir)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("missing entrypoint", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\nname=\"pkg\"\nversion=\"0.1.0\"\n")
		_, err := manifest.LoadFrom(dir)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("all fields missing", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, "[package]\n")
		_, err := manifest.LoadFrom(dir)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("found in a parent directory", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		nested := filepath.Join(dir, "src", "chapters")
		require.NoError(t, paths.EnsureDir(nested))

		m, err := manifest.LoadFrom(nested)
		require.NoError(t, err)
		assert.Equal(t, "my-pkg", m.Package.Name)
	})
}

func TestFindFile(t *testing.T) {
	t.Parallel()
	t.Run("returns the path of the manifest", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		got, err := manifest.FindFile(dir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "typst.toml"), got)
	})
	t.Run("walks up to the closest manifest", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		nested := filepath.Join(dir, "a", "b")
		require.NoError(t, paths.EnsureDir(nested))

		got, err := manifest.FindFile(nested)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "typst.toml"), got)
	})
	t.Run("reports the search origin when nothing is found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := manifest.FindFile(dir)
		require.ErrorIs(t, err, manifest.ErrManifestNotFound)
		assert.Contains(t, err.Error(), dir)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	t.Run("writes back the package fields", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML)
		file := filepath.Join(dir, "typst.toml")

		m, err := manifest.LoadFile(file)
		require.NoError(t, err)
		m.Package.Version = "2.0.0"
		m.Package.Name = "renamed"
		require.NoError(t, manifest.Update(file, m, false))

		got, err := manifest.LoadFile(file)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", got.Package.Version)
		assert.Equal(t, "renamed", got.Package.Name)
		assert.Equal(t, "lib.typ", got.Package.Entrypoint)
	})
	t.Run("keeps unrelated sections untouched", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, validManifestTOML+"\n[tool.gotpm]\nkey = \"value\"\n")
		file := filepath.Join(dir, "typst.toml")

		m, err := manifest.LoadFile(file)
		require.NoError(t, err)
		m.Package.Version = "9.9.9"
		require.NoError(t, manifest.Update(file, m, false))

		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, toml.Unmarshal(data, &doc))
		tool, ok := doc["tool"].(map[string]any)
		require.True(t, ok, "the [tool] section must survive an update")
		gotpm, ok := tool["gotpm"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "value", gotpm["key"])
	})
	t.Run("missing package section is an invalid manifest", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		file := filepath.Join(dir, "typst.toml")
		require.NoError(t, paths.WriteFile(file, []byte("[other]\nkey = \"val\"\n")))

		err := manifest.Update(file, &manifest.Manifest{}, false)
		assert.ErrorIs(t, err, manifest.ErrInvalidManifest)
	})
	t.Run("unreadable file is reported", func(t *testing.T) {
		t.Parallel()
		err := manifest.Update(filepath.Join(t.TempDir(), "missing.toml"), &manifest.Manifest{}, false)
		require.Error(t, err)
	})
}
