package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_resolveDestination(t *testing.T) {
	dataDir := t.TempDir()
	manifest := newManifest("my-package", "0.1.0", "lib.typ")
	t.Run("default namespace builds correct path", func(t *testing.T) {
		got, err := resolveDestination(dataDir, manifest, defaultNamespace)
		assert.NoError(t, err)
		assert.Equal(t, "my-package", got.Name)
		assert.Equal(t, "0.1.0", got.Version)
		assert.Equal(t, defaultNamespace, got.Namespace)
		wantPath := filepath.Join(dataDir, "typst", "packages", "local", "my-package", "0.1.0")
		assert.Equal(t, wantPath, got.Path)
	})
	t.Run("custom namespace builds correct path", func(t *testing.T) {
		got, err := resolveDestination(dataDir, manifest, "preview")
		assert.NoError(t, err)
		assert.Equal(t, "preview", got.Namespace)
		wantPath := filepath.Join(dataDir, "typst", "packages", "preview", "my-package", "0.1.0")
		assert.Equal(t, wantPath, got.Path)
	})
	t.Run("empty namespace returns error", func(t *testing.T) {
		_, err := resolveDestination(dataDir, manifest, "")
		assert.ErrorIs(t, err, ErrEmptyNamespace)
	})
	t.Run("already installed returns error", func(t *testing.T) {
		existing := filepath.Join(dataDir, "typst", "packages", "local", "my-package", "0.1.0")
		err := os.MkdirAll(existing, 0755)
		assert.NoError(t, err)

		_, err = resolveDestination(dataDir, manifest, defaultNamespace)
		assert.ErrorIs(t, err, ErrPackageAlreadyInstalled)
	})
}

func Test_resolveLocalPackageDir(t *testing.T) {
	t.Run("creates dir", func(t *testing.T) {
		got, err := resolveLocalPackageDir()
		assert.NoError(t, err)
		info, statErr := os.Stat(got)
		assert.NoError(t, statErr)
		if !info.IsDir() {
			t.Fatalf("expected a directory at %q", got)
		}
	})
	t.Run("contains typst-packages", func(t *testing.T) {
		got, err := resolveLocalPackageDir()
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
		got, err := resolveLocalPackageDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, xdgDir)
	})
	t.Run("fallsback to home/.local", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		home, _ := os.UserHomeDir()
		got, err := resolveLocalPackageDir()
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
		got, err := resolveLocalPackageDir()
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
		got, err := resolveLocalPackageDir()
		assert.NoError(t, err)
		assertHasPrefix(t, got, appData)
	})
	t.Run("missing AppData returns error", func(t *testing.T) {
		t.Setenv("APPDATA", "")
		_, err := resolveLocalPackageDir()
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

func newManifest(name, version, entrypoint string) Manifest {
	return Manifest{
		Package: PackageMeta{
			Name:       name,
			Version:    version,
			Entrypoint: entrypoint,
		},
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
