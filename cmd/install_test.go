package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/npikall/gotpm/cmd/internal"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/assert"
)

func Test_copyPackageFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "README.md", "readme")
	srcSubDir := filepath.Join(src, "utils")
	os.MkdirAll(srcSubDir, 0755)
	writeFile(t, srcSubDir, "helper.typ", "helper")

	t.Run("copies all files concurrently", func(t *testing.T) {
		err := copyPackageFiles(src, dst)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(dst, "lib.typ"))
		assert.FileExists(t, filepath.Join(dst, "README.md"))
		assert.FileExists(t, filepath.Join(dst, "utils", "helper.typ"))
	})
}

func Test_resolveDestination(t *testing.T) {
	dataDir := t.TempDir()
	manifest := newManifest("my-package", "0.1.0", "lib.typ")
	t.Run("default namespace builds correct path", func(t *testing.T) {
		got, err := resolveDestinationWithNamespace(dataDir, manifest, internal.DefaultNamespace)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", got.Name)
		assert.Equal(t, "0.1.0", got.Version)
		assert.Equal(t, internal.DefaultNamespace, got.Namespace)
		wantPath := filepath.Join(dataDir, "local", "my-package", "0.1.0")
		assert.Equal(t, wantPath, got.Path)
	})
	t.Run("custom namespace builds correct path", func(t *testing.T) {
		got, err := resolveDestinationWithNamespace(dataDir, manifest, "preview")
		assert.NoError(t, err)
		assert.Equal(t, "preview", got.Namespace)
		wantPath := filepath.Join(dataDir, "preview", "my-package", "0.1.0")
		assert.Equal(t, wantPath, got.Path)
	})
	t.Run("empty namespace returns error", func(t *testing.T) {
		_, err := resolveDestinationWithNamespace(dataDir, manifest, "")
		assert.ErrorIs(t, err, ErrEmptyNamespace)
	})
	t.Run("already installed returns error", func(t *testing.T) {
		existing := filepath.Join(dataDir, "local", "my-package", "0.1.0")
		err := os.MkdirAll(existing, 0755)
		assert.NoError(t, err)

		_, err = resolveDestinationWithNamespace(dataDir, manifest, internal.DefaultNamespace)
		assert.ErrorIs(t, err, ErrPackageAlreadyInstalled)
	})
}

func Test_resolveLocalPackageDir(t *testing.T) {
	t.Run("creates dir", func(t *testing.T) {
		got, err := internal.ResolveLocalPackageDir()
		assert.NoError(t, err)
		info, statErr := os.Stat(got)
		assert.NoError(t, statErr)
		if !info.IsDir() {
			t.Fatalf("expected a directory at %q", got)
		}
	})
	t.Run("contains typst/packages", func(t *testing.T) {
		got, err := internal.ResolveLocalPackageDir()
		assert.NoError(t, err)
		suffix := filepath.Join("typst", "packages")
		assertHasSuffix(t, got, suffix)
	})
}

func Test_resolveLinuxDataDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	t.Run("uses xdg if set", func(t *testing.T) {
		xdgDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", xdgDir)
		got, err := internal.ResolveLinuxDataDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, xdgDir)
	})
	t.Run("fallsback to home/.local", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		home, _ := os.UserHomeDir()
		got, err := internal.ResolveLinuxDataDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, filepath.Join(home, ".local", "share"))
	})
}

func Test_resolveDarwinDataDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	t.Run("uses Library Application Support", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		got, err := internal.ResolveDarwinDataDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, filepath.Join(home, "Library", "Application Support"))
	})
}

func Test_resolveWindowsDataDir(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	t.Run("uses AppData", func(t *testing.T) {
		appData := t.TempDir()
		t.Setenv("APPDATA", appData)
		got, err := internal.ResolveWindowsDataDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, appData)
	})
	t.Run("missing AppData returns error", func(t *testing.T) {
		t.Setenv("APPDATA", "")
		_, err := internal.ResolveWindowsDataDir()
		assert.Error(t, err)
	})
}

func TestLoadManifest_validManifest_returns_correct(t *testing.T) {
	t.Run("valid manifest returns correct", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
entrypoint = "lib.typ"
`)
		got, err := internal.LoadManifest(dir)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", got.Package.Name)
		assert.Equal(t, "1.0.0", got.Package.Version)
		assert.Equal(t, "lib.typ", got.Package.Entrypoint)
	})
	t.Run("no manifest returns not found error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrManifestNotFound)
	})
	t.Run("malformed toml returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `this is not valid [ toml`)
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrInvalidManifest)
	})
	t.Run("missing name returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
version = "1.0.0"
entrypoint = "lib.typ"
`)
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrInvalidManifest)
	})
	t.Run("missing version returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
entrypoint = "lib.typ"
`)
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrInvalidManifest)
	})
	t.Run("missing entrypoint returns invalid error", func(t *testing.T) {
		dir := writeManifest(t, `
[package]
name = "my-package"
version = "1.0.0"
`)
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrInvalidManifest)
	})
	t.Run("all fields missing reports all errors", func(t *testing.T) {
		dir := writeManifest(t, `[package]`)
		_, err := internal.LoadManifest(dir)
		assert.ErrorIs(t, err, internal.ErrInvalidManifest)
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

func assertHasSuffix(t *testing.T, path, suffix string) {
	t.Helper()
	if !strings.HasSuffix(path, suffix) {
		t.Fatalf("expected path %q to end with %q", path, suffix)
	}
}

func assertHasPrefix(t *testing.T, path, prefix string) {
	t.Helper()
	if !strings.HasPrefix(path, prefix) {
		t.Fatalf("expected path %q to start with %q", path, prefix)
	}
}

func newManifest(name, version, entrypoint string) internal.Manifest {
	return internal.Manifest{
		Package: internal.PackageMeta{
			Name:       name,
			Version:    version,
			Entrypoint: entrypoint,
		},
	}
}

func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	dir, _ = filepath.EvalSymlinks(dir)
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		t.Fatalf("writing test file: %v", err)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Test_symlinkPackage(t *testing.T) {
	src := t.TempDir()
	src, _ = filepath.EvalSymlinks(src) // normalise macOS /var -> /private/var
	writeFile(t, src, "lib.typ", "#let x = 1")

	t.Run("symlink is created at dest", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "local", "my-pkg", "0.1.0")
		err := symlinkPackage(src, dest)
		assert.NoError(t, err)
		info, err := os.Lstat(dest)
		assert.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "dest should be a symlink")
	})
	t.Run("symlink points to absolute sourceDir", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "local", "my-pkg", "0.1.0")
		err := symlinkPackage(src, dest)
		assert.NoError(t, err)
		target, err := os.Readlink(dest)
		assert.NoError(t, err)
		assert.True(t, filepath.IsAbs(target), "symlink target must be absolute")
		assert.Equal(t, src, target)
	})
	t.Run("source contents are accessible through the symlink", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "local", "my-pkg", "0.1.0")
		err := symlinkPackage(src, dest)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(dest, "lib.typ"))
		content, err := os.ReadFile(filepath.Join(dest, "lib.typ"))
		assert.NoError(t, err)
		assert.Equal(t, "#let x = 1", string(content))
	})
	t.Run("parent directory is created if missing", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "preview", "other-pkg", "1.2.3")
		err := symlinkPackage(src, dest)
		assert.NoError(t, err)
		assert.DirExists(t, filepath.Dir(dest))
	})
}

func Test_readIgnoreLines(t *testing.T) {
	t.Run("returns nil for non-existent file", func(t *testing.T) {
		got := readIgnoreLines("/does/not/exist/.typstignore")
		assert.Nil(t, got)
	})
	t.Run("returns trimmed non-empty lines", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".typstignore")
		os.WriteFile(path, []byte("*.typ\nREADME.md\n"), 0644)
		got := readIgnoreLines(path)
		assert.Equal(t, []string{"*.typ", "README.md"}, got)
	})
	t.Run("skips blank lines", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".typstignore")
		os.WriteFile(path, []byte("*.typ\n\nREADME.md\n"), 0644)
		got := readIgnoreLines(path)
		assert.Equal(t, []string{"*.typ", "README.md"}, got)
	})
	t.Run("strips windows-style CRLF line endings", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".typstignore")
		os.WriteFile(path, []byte("*.typ\r\nREADME.md\r\n"), 0644)
		got := readIgnoreLines(path)
		assert.Equal(t, []string{"*.typ", "README.md"}, got)
	})
}

func Test_shouldIgnore(t *testing.T) {
	t.Run("root . is never ignored", func(t *testing.T) {
		assert.False(t, shouldIgnore(".", nil))
	})
	t.Run("hardcoded filenames are ignored", func(t *testing.T) {
		for name := range ignoredFileNames {
			assert.True(t, shouldIgnore(name, nil), "expected %q to be ignored", name)
			assert.True(t, shouldIgnore(filepath.Join("subdir", name), nil))
		}
	})
	t.Run("nil matcher does not ignore unknown files", func(t *testing.T) {
		assert.False(t, shouldIgnore("lib.typ", nil))
	})
	t.Run("matcher-matched path is ignored", func(t *testing.T) {
		matcher := buildIgnoreMatcher_fromLines(t, "*.md")
		assert.True(t, shouldIgnore("README.md", matcher))
		assert.False(t, shouldIgnore("lib.typ", matcher))
	})
}

func Test_collectJobs_respectsTypstIgnore(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "README.md", "readme")
	writeFile(t, src, "secret.txt", "secret")
	writeFile(t, src, ".typstignore", "*.md\nsecret.txt\n")

	jobs, err := collectJobs(src, dst, buildIgnoreMatcher(src))
	assert.NoError(t, err)

	paths := jobSrcBasenames(jobs)
	assert.Contains(t, paths, "lib.typ")
	assert.NotContains(t, paths, "README.md", ".typstignore pattern *.md should exclude README.md")
	assert.NotContains(t, paths, "secret.txt", ".typstignore pattern secret.txt should exclude secret.txt")
	assert.NotContains(t, paths, ".typstignore", "hardcoded rule should exclude .typstignore itself")
}

func Test_collectJobs_respectsGitIgnore(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "README.md", "readme")
	writeFile(t, src, ".gitignore", "*.md\n")

	jobs, err := collectJobs(src, dst, buildIgnoreMatcher(src))
	assert.NoError(t, err)

	paths := jobSrcBasenames(jobs)
	assert.Contains(t, paths, "lib.typ")
	assert.NotContains(t, paths, "README.md", ".gitignore pattern *.md should exclude README.md")
	assert.NotContains(t, paths, ".gitignore", "hardcoded rule should exclude .gitignore itself")
}

func Test_collectJobs_gitIgnoreAndTypstIgnoreCombined(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	writeFile(t, src, "README.md", "readme")
	writeFile(t, src, "secret.txt", "secret")
	writeFile(t, src, ".gitignore", "*.md\n")
	writeFile(t, src, ".typstignore", "secret.txt\n")

	jobs, err := collectJobs(src, dst, buildIgnoreMatcher(src))
	assert.NoError(t, err)

	paths := jobSrcBasenames(jobs)
	assert.Contains(t, paths, "lib.typ")
	assert.NotContains(t, paths, "README.md")
	assert.NotContains(t, paths, "secret.txt")
}

func Test_collectJobs_ignoredDirectorySkipsContents(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "lib.typ", "content")
	subdir := filepath.Join(src, "dist")
	os.MkdirAll(subdir, 0755)
	writeFile(t, subdir, "output.typ", "generated")
	writeFile(t, src, ".typstignore", "dist/\n")

	jobs, err := collectJobs(src, dst, buildIgnoreMatcher(src))
	assert.NoError(t, err)

	paths := jobSrcBasenames(jobs)
	assert.Contains(t, paths, "lib.typ")
	assert.NotContains(t, paths, "dist/output.typ", "files inside ignored directory should be excluded")
}

// buildIgnoreMatcher_fromLines builds a matcher from inline pattern strings,
// used in unit tests that do not need real files on disk.
func buildIgnoreMatcher_fromLines(t *testing.T, patterns ...string) *ignore.GitIgnore {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, ".typstignore", strings.Join(patterns, "\n")+"\n")
	return buildIgnoreMatcher(dir)
}

// jobSrcBasenames extracts the base filename of each job's source path.
func jobSrcBasenames(jobs []transferJob) []string {
	names := make([]string, len(jobs))
	for i, j := range jobs {
		names[i] = filepath.Base(j.src)
	}
	return names
}

func Test_validateDestinationConflict_rejectsEditableReinstall(t *testing.T) {
	src := t.TempDir()
	src, _ = filepath.EvalSymlinks(src)
	writeFile(t, src, "lib.typ", "")
	dest := filepath.Join(t.TempDir(), "local", "my-pkg", "0.1.0")

	check(symlinkPackage(src, dest))

	err := validateDestinationConflict(dest)
	assert.ErrorIs(t, err, ErrPackageAlreadyInstalled)
}
