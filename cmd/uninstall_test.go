package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
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
		name, ver, err := resolvePackageIdentityFromWorkingDir([]string{"foo"}, "0.1.0", false, "")
		assert.NoError(t, err)
		assert.Equal(t, "foo", name)
		assert.Equal(t, "0.1.0", ver)
	})
	t.Run("name without version and without deleteAll returns error", func(t *testing.T) {
		_, _, err := resolvePackageIdentityFromWorkingDir([]string{"foo"}, "", false, "")
		assert.ErrorIs(t, err, ErrInsufficientPackage)
	})
	t.Run("name without version but with deleteAll succeeds", func(t *testing.T) {
		name, ver, err := resolvePackageIdentityFromWorkingDir([]string{"foo"}, "", true, "")
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
		name, ver, err := resolvePackageIdentityFromWorkingDir([]string{}, "", false, dir)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", name)
		assert.Equal(t, "1.0.0", ver)
	})
	t.Run("no args and missing manifest returns error", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := resolvePackageIdentityFromWorkingDir([]string{}, "", false, dir)
		assert.ErrorIs(t, err, internal.ErrManifestNotFound)
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
	t.Run("non-existing path returns error containing path", func(t *testing.T) {
		nonexistent := filepath.Join(dir, "nonexistent")
		err := validateTargetExists(nonexistent)
		assert.ErrorContains(t, err, "path does not exist")
		assert.ErrorContains(t, err, nonexistent)
	})
	t.Run("dangling symlink counts as present", func(t *testing.T) {
		link := filepath.Join(dir, "dangling")
		check(os.Symlink(filepath.Join(dir, "nowhere"), link))
		err := validateTargetExists(link)
		assert.NoError(t, err)
	})
}

func Test_removeTarget(t *testing.T) {
	t.Run("removes regular directory and its contents", func(t *testing.T) {
		parent := t.TempDir()
		target := filepath.Join(parent, "pkg")
		check(os.MkdirAll(filepath.Join(target, "sub"), 0755))
		check(os.WriteFile(filepath.Join(target, "lib.typ"), []byte(""), 0644))

		err := removeTarget(target)
		assert.NoError(t, err)
		assert.NoDirExists(t, target)
	})
	t.Run("removes symlink without deleting the pointed-to directory", func(t *testing.T) {
		real := t.TempDir()
		check(os.WriteFile(filepath.Join(real, "lib.typ"), []byte(""), 0644))
		parent := t.TempDir()
		link := filepath.Join(parent, "link")
		check(os.Symlink(real, link))

		err := removeTarget(link)
		assert.NoError(t, err)
		assert.NoFileExists(t, link)
		assert.DirExists(t, real) // target directory must be untouched
	})
	t.Run("removes dangling symlink", func(t *testing.T) {
		parent := t.TempDir()
		link := filepath.Join(parent, "dangling")
		check(os.Symlink(filepath.Join(parent, "nowhere"), link))

		err := removeTarget(link)
		assert.NoError(t, err)
		assert.NoFileExists(t, link)
	})
}

func Test_readUninstallFlags(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().StringP("namespace", "n", "local", "")
		cmd.Flags().StringP("version", "v", "", "")
		cmd.Flags().Bool("all", false, "")
		cmd.Flags().Bool("dry-run", false, "")
		return cmd
	}

	t.Run("defaults are applied", func(t *testing.T) {
		flags, err := readUninstallFlags(newCmd())
		assert.NoError(t, err)
		assert.Equal(t, "local", flags.namespace)
		assert.Equal(t, "", flags.version)
		assert.False(t, flags.deleteAll)
		assert.False(t, flags.isDryRun)
	})
	t.Run("explicit values are read", func(t *testing.T) {
		cmd := newCmd()
		check(cmd.Flags().Set("namespace", "preview"))
		check(cmd.Flags().Set("version", "0.2.0"))
		check(cmd.Flags().Set("all", "true"))
		check(cmd.Flags().Set("dry-run", "true"))
		flags, err := readUninstallFlags(cmd)
		assert.NoError(t, err)
		assert.Equal(t, "preview", flags.namespace)
		assert.Equal(t, "0.2.0", flags.version)
		assert.True(t, flags.deleteAll)
		assert.True(t, flags.isDryRun)
	})
}
