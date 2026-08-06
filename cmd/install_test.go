package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/npikall/gotpm/cmd"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateIsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	notExistingDir := filepath.Join(dir, "subdir")
	file := filepath.Join(dir, "empty")
	check(os.WriteFile(file, []byte(""), paths.FilePerm))

	t.Run("file returns error", func(t *testing.T) {
		t.Parallel()
		err := ValidateIsDirectory(file)
		assert.ErrorContains(t, err, "path is not a directory")
	})
	t.Run("directory does not return error", func(t *testing.T) {
		t.Parallel()
		err := ValidateIsDirectory(dir)
		assert.NoError(t, err)
	})
	t.Run("non existing directory does return error", func(t *testing.T) {
		t.Parallel()
		err := ValidateIsDirectory(notExistingDir)
		assert.ErrorContains(t, err, "directory does not exist")
	})
}

func Test_resolveProvidedPath(t *testing.T) { //nolint: paralleltest
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	subdir := filepath.Join(dir, "subdir")
	check(os.Mkdir(subdir, paths.DirPerm))
	t.Chdir(dir)

	t.Run("absolute path to existing dir returns correct", func(t *testing.T) { //nolint: paralleltest
		got, gotErr := ResolveProvidedPath(dir)
		assert.Equal(t, dir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("absolute path to non-existing dir returns error", func(t *testing.T) { //nolint: paralleltest
		_, gotErr := ResolveProvidedPath("/foo/bar/baz")
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
	t.Run("relative path to existing dir returns correct", func(t *testing.T) { //nolint: paralleltest
		got, gotErr := ResolveProvidedPath("subdir")
		assert.Equal(t, subdir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("relative path to non-existing dir returns parent", func(t *testing.T) { //nolint: paralleltest
		_, gotErr := ResolveProvidedPath("nonSubDir")
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
}

func Test_resolveSourceDir(t *testing.T) { //nolint: paralleltest
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	t.Chdir(dir)
	cwd, _ := os.Getwd()
	subdir := filepath.Join(dir, "subdir")
	check(os.Mkdir(subdir, paths.DirPerm))
	file := filepath.Join(dir, "file.txt")
	check(os.WriteFile(file, []byte(""), paths.FilePerm))

	t.Run("no args returns cwd", func(t *testing.T) { //nolint: paralleltest
		got, gotErr := ResolveLocalSourceDir([]string{})
		assert.Equal(t, cwd, got)
		assert.NoError(t, gotErr)
	})
	t.Run("too many args returns error", func(t *testing.T) { //nolint: paralleltest
		_, gotErr := ResolveLocalSourceDir([]string{"a", "b"})
		assert.ErrorIs(t, gotErr, ErrTooManyArguments)
	})
	t.Run("valid dir returns absPath", func(t *testing.T) { //nolint: paralleltest
		got, gotErr := ResolveLocalSourceDir([]string{dir})
		assert.Equal(t, cwd, got)
		assert.NoError(t, gotErr)
	})
	t.Run("relative path resolves to absolute", func(t *testing.T) { //nolint: paralleltest
		got, gotErr := ResolveLocalSourceDir([]string{"subdir"})
		assert.Equal(t, subdir, got)
		assert.NoError(t, gotErr)
	})
	t.Run("non-existing path returns error", func(t *testing.T) { //nolint: paralleltest
		_, gotErr := ResolveLocalSourceDir([]string{"path/does/not/exist"})
		assert.ErrorContains(t, gotErr, "directory does not exist")
	})
	t.Run("filepath returns error", func(t *testing.T) { //nolint: paralleltest
		_, gotErr := ResolveLocalSourceDir([]string{"file.txt"})
		assert.ErrorContains(t, gotErr, "path is not a directory")
	})
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(content), paths.FilePerm)
	if err != nil {
		t.Fatalf("writing test manifest: %v", err)
	}
	return dir
}

func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	dir, _ = filepath.EvalSymlinks(dir)
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), paths.FilePerm)
	if err != nil {
		t.Fatalf("writing test file: %v", err)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// buildIgnoreMatcher_fromLines builds a matcher from inline pattern strings,
// used in unit tests that do not need real files on disk.
// jobSrcBasenames extracts the base filename of each job's source path.
func Test_installRunner_force(t *testing.T) {
	t.Parallel()
	const manifest = "[package]\nname = \"my-pkg\"\nversion = \"0.1.0\"\nentrypoint = \"lib.typ\"\n"
	newForceCmd := func(installDir string) *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().CountP("verbose", "v", "")
		cmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "")
		cmd.Flags().BoolP("editable", "e", false, "")
		cmd.Flags().BoolP("force", "f", false, "")
		cmd.Flags().String(paths.InstallDirFlag, installDir, "")
		cmd.Flags().StringP("remote", "r", "", "")
		cmd.Flags().StringP("rev", "t", "HEAD", "")
		return cmd
	}

	t.Run("force overwrites existing copy install", func(t *testing.T) {
		t.Parallel()
		src := writeManifest(t, manifest)
		writeFile(t, src, "lib.typ", "v1")

		dest := filepath.Join(t.TempDir(), "pkg-dest")

		// first install
		cmd := newForceCmd(dest)
		err := InstallRunner(cmd, []string{src})
		require.NoError(t, err)

		// write updated file
		writeFile(t, src, "lib.typ", "v2")
		writeFile(t, src, "extra.typ", "new")

		// second install without --force should fail
		cmd = newForceCmd(dest)
		err = InstallRunner(cmd, []string{src})
		require.ErrorIs(t, err, store.ErrAlreadyInstalled)

		// second install with --force should succeed
		cmd = newForceCmd(dest)
		check(cmd.Flags().Set("force", "true"))
		err = InstallRunner(cmd, []string{src})
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(dest, "lib.typ"))
		require.NoError(t, err)
		assert.Equal(t, "v2", string(content))
		assert.FileExists(t, filepath.Join(dest, "extra.typ"))
	})

	t.Run("force replaces existing editable install", func(t *testing.T) {
		t.Parallel()
		src := writeManifest(t, manifest)
		writeFile(t, src, "lib.typ", "content")
		src, _ = filepath.EvalSymlinks(src)

		dest := filepath.Join(t.TempDir(), "pkg-dest")

		// first install as editable
		cmd := newForceCmd(dest)
		check(cmd.Flags().Set("editable", "true"))
		check(InstallRunner(cmd, []string{src}))

		info, err := os.Lstat(dest)
		require.NoError(t, err)
		assert.NotEqual(t, 0, info.Mode()&os.ModeSymlink)

		// force re-install as editable
		cmd = newForceCmd(dest)
		check(cmd.Flags().Set("editable", "true"))
		check(cmd.Flags().Set("force", "true"))
		err = InstallRunner(cmd, []string{src})
		require.NoError(t, err)

		info, err = os.Lstat(dest)
		require.NoError(t, err)
		assert.NotEqual(t, 0, info.Mode()&os.ModeSymlink)
		target, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, src, target)
	})
}

func Test_readInstallOptions(t *testing.T) {
	t.Parallel()
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().BoolP("force", "f", false, "")
		cmd.Flags().BoolP("editable", "e", false, "")
		cmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "")
		cmd.Flags().String(paths.InstallDirFlag, "", "")
		cmd.Flags().StringP("remote", "r", "", "")
		cmd.Flags().StringP("rev", "t", "HEAD", "")
		return cmd
	}

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		cmd := newCmd()
		opts := ReadInstallOptions(cmd)
		assert.False(t, opts.Force)
		assert.False(t, opts.Editable)
		assert.Equal(t, paths.DefaultNamespace, opts.Namespace)
	})
	t.Run("force flag", func(t *testing.T) {
		t.Parallel()
		cmd := newCmd()
		check(cmd.Flags().Set("force", "true"))
		opts := ReadInstallOptions(cmd)
		assert.True(t, opts.Force)
	})
	t.Run("editable flag", func(t *testing.T) {
		t.Parallel()
		cmd := newCmd()
		check(cmd.Flags().Set("editable", "true"))
		opts := ReadInstallOptions(cmd)
		assert.True(t, opts.Editable)
	})
	t.Run("custom namespace", func(t *testing.T) {
		t.Parallel()
		cmd := newCmd()
		check(cmd.Flags().Set("namespace", "preview"))
		opts := ReadInstallOptions(cmd)
		assert.Equal(t, "preview", opts.Namespace)
	})
}
