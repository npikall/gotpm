package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateIsDirectory(t *testing.T) {
	dir := t.TempDir()
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}
