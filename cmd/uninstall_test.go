package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_resolveUninstallTarget(t *testing.T) {
	pkgDir := "/tmp/typst/packages"

	t.Run("specific version targets version dir", func(t *testing.T) {
		got := resolveUninstallTarget(pkgDir, "local", "foo", "0.1.0", false)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo", "0.1.0"), got)
	})
	t.Run("deleteAll without version targets package dir", func(t *testing.T) {
		got := resolveUninstallTarget(pkgDir, "local", "foo", "", true)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo"), got)
	})
	t.Run("deleteAll with version targets version dir", func(t *testing.T) {
		got := resolveUninstallTarget(pkgDir, "local", "foo", "0.1.0", true)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo", "0.1.0"), got)
	})
	t.Run("custom namespace uses namespace in path", func(t *testing.T) {
		got := resolveUninstallTarget(pkgDir, "preview", "foo", "0.1.0", false)
		assert.Equal(t, filepath.Join(pkgDir, "preview", "foo", "0.1.0"), got)
	})
}

func Test_resolvePackageIdentity(t *testing.T) {
	t.Run("name and version from args", func(t *testing.T) {
		name, ver, err := resolvePackageIdentity([]string{"foo"}, "0.1.0", false, "")
		assert.NoError(t, err)
		assert.Equal(t, "foo", name)
		assert.Equal(t, "0.1.0", ver)
	})
	t.Run("name without version and without deleteAll returns error", func(t *testing.T) {
		_, _, err := resolvePackageIdentity([]string{"foo"}, "", false, "")
		assert.ErrorIs(t, err, ErrInsufficientPackage)
	})
	t.Run("name without version but with deleteAll succeeds", func(t *testing.T) {
		name, ver, err := resolvePackageIdentity([]string{"foo"}, "", true, "")
		assert.NoError(t, err)
		assert.Equal(t, "foo", name)
		assert.Equal(t, "", ver)
	})
	t.Run("no args reads name and version from manifest", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
entrypoint = "lib.typ"
`)
		name, ver, err := resolvePackageIdentity([]string{}, "", false, dir)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", name)
		assert.Equal(t, "1.0.0", ver)
	})
	t.Run("no args and missing manifest returns error", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := resolvePackageIdentity([]string{}, "", false, dir)
		assert.ErrorIs(t, err, ErrManifestNotFound)
	})
}

func Test_validateTargetExists(t *testing.T) {
	dir := t.TempDir()

	t.Run("existing directory returns no error", func(t *testing.T) {
		err := validateTargetExists(dir)
		assert.NoError(t, err)
	})
	t.Run("existing file returns no error", func(t *testing.T) {
		file := filepath.Join(dir, "file.txt")
		check(os.WriteFile(file, []byte(""), 0644))
		err := validateTargetExists(file)
		assert.NoError(t, err)
	})
	t.Run("non-existing path returns error", func(t *testing.T) {
		err := validateTargetExists(filepath.Join(dir, "nonexistent"))
		assert.ErrorContains(t, err, "path does not exist")
	})
}
