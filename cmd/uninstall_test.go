package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/cmd"
	"github.com/npikall/gotpm/internal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resolveUninstallTarget(t *testing.T) {
	t.Parallel()
	pkgDir := "/tmp/typst/packages"

	t.Run("specific version targets version dir", func(t *testing.T) {
		t.Parallel()
		got := ResolveUninstallTarget(pkgDir, "local", "foo", "0.1.0", false)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo", "0.1.0"), got)
	})
	t.Run("deleteAll without version targets package dir", func(t *testing.T) {
		t.Parallel()
		got := ResolveUninstallTarget(pkgDir, "local", "foo", "", true)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo"), got)
	})
	t.Run("deleteAll with version targets version dir", func(t *testing.T) {
		t.Parallel()
		got := ResolveUninstallTarget(pkgDir, "local", "foo", "0.1.0", true)
		assert.Equal(t, filepath.Join(pkgDir, "local", "foo", "0.1.0"), got)
	})
	t.Run("custom namespace uses namespace in path", func(t *testing.T) {
		t.Parallel()
		got := ResolveUninstallTarget(pkgDir, "preview", "foo", "0.1.0", false)
		assert.Equal(t, filepath.Join(pkgDir, "preview", "foo", "0.1.0"), got)
	})
}

func Test_resolvePackageIdentity(t *testing.T) {
	t.Parallel()
	t.Run("name and version from args", func(t *testing.T) {
		t.Parallel()
		name, ver, err := ResolvePackageIdentityFromWorkingDir([]string{"foo"}, "0.1.0", false, "")
		require.NoError(t, err)
		assert.Equal(t, "foo", name)
		assert.Equal(t, "0.1.0", ver)
	})
	t.Run("name without version and without deleteAll returns error", func(t *testing.T) {
		t.Parallel()
		_, _, err := ResolvePackageIdentityFromWorkingDir([]string{"foo"}, "", false, "")
		assert.ErrorIs(t, err, ErrInsufficientPackage)
	})
	t.Run("name without version but with deleteAll succeeds", func(t *testing.T) {
		t.Parallel()
		name, ver, err := ResolvePackageIdentityFromWorkingDir([]string{"foo"}, "", true, "")
		require.NoError(t, err)
		assert.Equal(t, "foo", name)
		assert.Empty(t, ver)
	})
	t.Run("no args reads name and version from manifest", func(t *testing.T) {
		t.Parallel()
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
entrypoint = "lib.typ"
`)
		name, ver, err := ResolvePackageIdentityFromWorkingDir([]string{}, "", false, dir)
		require.NoError(t, err)
		assert.Equal(t, "my-package", name)
		assert.Equal(t, "1.0.0", ver)
	})
	t.Run("no args and missing manifest returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, _, err := ResolvePackageIdentityFromWorkingDir([]string{}, "", false, dir)
		assert.ErrorIs(t, err, internal.ErrManifestNotFound)
	})
}

func Test_validateTargetExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	t.Run("existing directory returns no error", func(t *testing.T) {
		t.Parallel()
		err := ValidateTargetExists(dir)
		assert.NoError(t, err)
	})
	t.Run("existing file returns no error", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(dir, "file.txt")
		check(os.WriteFile(file, []byte(""), internal.FilePerm))
		err := ValidateTargetExists(file)
		assert.NoError(t, err)
	})
	t.Run("non-existing path returns error containing path", func(t *testing.T) {
		t.Parallel()
		nonexistent := filepath.Join(dir, "nonexistent")
		err := ValidateTargetExists(nonexistent)
		require.ErrorContains(t, err, "path does not exist")
		assert.ErrorContains(t, err, nonexistent)
	})
	t.Run("dangling symlink counts as present", func(t *testing.T) {
		t.Parallel()
		link := filepath.Join(dir, "dangling")
		check(os.Symlink(filepath.Join(dir, "nowhere"), link))
		err := ValidateTargetExists(link)
		assert.NoError(t, err)
	})
}

func Test_removeTarget(t *testing.T) {
	t.Parallel()
	t.Run("removes regular directory and its contents", func(t *testing.T) {
		t.Parallel()
		parent := t.TempDir()
		target := filepath.Join(parent, "pkg")
		check(os.MkdirAll(filepath.Join(target, "sub"), internal.DirPerm))
		check(os.WriteFile(filepath.Join(target, "lib.typ"), []byte(""), internal.FilePerm))

		err := RemoveTarget(target)
		require.NoError(t, err)
		assert.NoDirExists(t, target)
	})
	t.Run("removes symlink without deleting the pointed-to directory", func(t *testing.T) {
		t.Parallel()
		actual := t.TempDir()
		check(os.WriteFile(filepath.Join(actual, "lib.typ"), []byte(""), internal.FilePerm))
		parent := t.TempDir()
		link := filepath.Join(parent, "link")
		check(os.Symlink(actual, link))

		err := RemoveTarget(link)
		require.NoError(t, err)
		assert.NoFileExists(t, link)
		assert.DirExists(t, actual) // target directory must be untouched
	})
	t.Run("removes dangling symlink", func(t *testing.T) {
		t.Parallel()
		parent := t.TempDir()
		link := filepath.Join(parent, "dangling")
		check(os.Symlink(filepath.Join(parent, "nowhere"), link))

		err := RemoveTarget(link)
		require.NoError(t, err)
		assert.NoFileExists(t, link)
	})
}

func Test_readUninstallFlags(t *testing.T) {
	t.Parallel()
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().StringP("namespace", "n", "local", "")
		cmd.Flags().StringP("version", "v", "", "")
		cmd.Flags().Bool("all", false, "")
		cmd.Flags().Bool("dry-run", false, "")
		return cmd
	}

	t.Run("defaults are applied", func(t *testing.T) {
		t.Parallel()
		flags, err := ReadUninstallFlags(newCmd())
		require.NoError(t, err)
		assert.Equal(t, "local", flags.Namespace)
		assert.Empty(t, flags.Version)
		assert.False(t, flags.DeleteAll)
		assert.False(t, flags.IsDryRun)
	})
	t.Run("explicit values are read", func(t *testing.T) {
		t.Parallel()
		cmd := newCmd()
		check(cmd.Flags().Set("namespace", "preview"))
		check(cmd.Flags().Set("version", "0.2.0"))
		check(cmd.Flags().Set("all", "true"))
		check(cmd.Flags().Set("dry-run", "true"))
		flags, err := ReadUninstallFlags(cmd)
		require.NoError(t, err)
		assert.Equal(t, "preview", flags.Namespace)
		assert.Equal(t, "0.2.0", flags.Version)
		assert.True(t, flags.DeleteAll)
		assert.True(t, flags.IsDryRun)
	})
}
