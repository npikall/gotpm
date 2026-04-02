package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadManifest_validManifest_returns_correct(t *testing.T) {
	t.Run("valid manifest returns correct", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
entrypoint = "lib.typ"
`)
		got, err := loadManifest(dir)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", got.Package.Name)
		assert.Equal(t, "1.0.0", got.Package.Version)
		assert.Equal(t, "lib.typ", got.Package.Entrypoint)
	})
	t.Run("no manifest returns not found error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrManifestNotFound)
	})
	t.Run("malformed toml returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `this is not valid [ toml`)
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing name returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
version = "1.0.0"
entrypoint = "lib.typ"
`)
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing version returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
entrypoint = "lib.typ"
`)
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("missing entrypoint returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
`)
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})
	t.Run("all fields missing reports all errors", func(t *testing.T) {
		dir := writeManifest(t, `[package]`)
		_, err := loadManifest(dir)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.ErrorContains(t, err, "package.name")
		assert.ErrorContains(t, err, "package.version")
		assert.ErrorContains(t, err, "package.entrypoint")
	})
}

func Test_validateIsDirectory(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	notExistingDir := filepath.Join(dir, "subdir")
	file := filepath.Join(dir, "empty")
	check(os.WriteFile(file, []byte(""), 0644))

	t.Run("file returns error", func(t *testing.T) {
		err := validateIsDirectory(file)
		assert.ErrorContains(t, err, "path is not a directory:")
	})
	t.Run("directory does not return error", func(t *testing.T) {
		err := validateIsDirectory(dir)
		assert.NoError(t, err)
	})
	t.Run("non existing directory does return error", func(t *testing.T) {
		err := validateIsDirectory(notExistingDir)
		assert.ErrorContains(t, err, "directory does not exist:")
	})
}

func Test_resolveProvidedPath(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	subdir := filepath.Join(dir, "subdir")
	check(os.Mkdir(subdir, 0755))
	check(os.Chdir(dir))

	t.Run("absolute path to existing dir returns correct", func(t *testing.T) {
		got, gotErr := resolveProvidedPath(dir)
		assert.Equal(t, dir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("absolute path to non-existing dir returns error", func(t *testing.T) {
		_, gotErr := resolveProvidedPath("/foo/bar/baz")
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
	t.Run("relative path to existing dir returns correct", func(t *testing.T) {
		got, gotErr := resolveProvidedPath("subdir")
		assert.Equal(t, subdir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("relative path to non-existing dir returns parent", func(t *testing.T) {
		_, gotErr := resolveProvidedPath("nonSubDir")
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
}

func Test_resolveSourceDir(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	check(os.Chdir(dir))
	cwd, _ := os.Getwd()
	subdir := filepath.Join(dir, "subdir")
	check(os.Mkdir(subdir, 0755))
	file := filepath.Join(dir, "file.txt")
	check(os.WriteFile(file, []byte(""), 0644))

	t.Run("no args returns cwd", func(t *testing.T) {
		got, gotErr := resolveSourceDir([]string{})
		assert.Equal(t, cwd, got)
		assert.NoError(t, gotErr)
	})
	t.Run("too many args returns error", func(t *testing.T) {
		_, gotErr := resolveSourceDir([]string{"a", "b"})
		assert.ErrorIs(t, gotErr, ErrTooManyArguments)
	})
	t.Run("valid dir returns absPath", func(t *testing.T) {
		got, gotErr := resolveSourceDir([]string{dir})
		assert.Equal(t, cwd, got)
		assert.NoError(t, gotErr)
	})
	t.Run("relative path resolves to absolute", func(t *testing.T) {
		got, gotErr := resolveSourceDir([]string{"subdir"})
		assert.Equal(t, subdir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("non-existing path returns error", func(t *testing.T) {
		_, gotErr := resolveSourceDir([]string{"path/does/not/exist"})
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
	t.Run("filepath returns error", func(t *testing.T) {
		_, gotErr := resolveSourceDir([]string{"file.txt"})
		assert.ErrorContains(t, gotErr, "path is not a directory")
	})
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("writing test manifest: %v", err)
	}
	return dir
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
