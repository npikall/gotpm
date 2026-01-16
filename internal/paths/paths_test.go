package paths_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
)

func TestGetTypstPathRoot(t *testing.T) {
	cases := []struct {
		goos string
		path string
	}{
		{"darwin", "~/Library/Application Support"},
		{"windows", "~/AppData/Roaming"},
		{"linux", "~/.local/share"},
	}

	for _, tt := range cases {
		t.Run(tt.goos, func(t *testing.T) {
			got, _ := i.GetPlatformDataDirectory(tt.goos, "~")
			assert.Equal(t, tt.path, got, "got %s want %s", got, tt.path)
		})
	}

	t.Run("unsupported os", func(t *testing.T) {
		_, err := i.GetPlatformDataDirectory("", "~")
		assert.Error(t, err, "expected an error, but got none")
		assert.Equal(t, err, i.ErrOperatingSystem, "got %q want %q", err, i.ErrOperatingSystem)
	})

	t.Run("custom windows path", func(t *testing.T) {
		t.Setenv("APPDATA", "~/path/foo")
		got, _ := i.GetPlatformDataDirectory(i.WINDOWS, "~")
		want := "~/path/foo"

		assert.Equal(t, want, got, "got %s want %s", got, want)
	})
	t.Run("custom linux path", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "~/path/foo")
		got, _ := i.GetPlatformDataDirectory(i.LINUX, "~")
		want := "~/path/foo"

		assert.Equal(t, want, got, "got %q want %q", got, want)
	})
}

func TestResolveTargetPath(t *testing.T) {
	cwd := "src/dir/"
	src := "src/dir/main.go"
	dst := "dst/dir/"
	want := "dst/dir/main.go"

	got, err := i.ResolveTargetPath(cwd, src, dst)
	assert.NoError(t, err, "expected no error, but got one")
	assert.Equal(t, want, got, "got %s want %s", got, want)
}

func TestResolveUninstallTarget(t *testing.T) {
	dataDir := "typst"
	defaultNamespace := "local"

	t.Run("exact", func(t *testing.T) {
		got, err := i.ResolveUninstallTarget(dataDir, false, defaultNamespace, "foo", "v0")
		assert.Equal(t, "typst/local/foo/v0", got)
		assert.NoError(t, err)
	})
	t.Run("missing version", func(t *testing.T) {
		_, err := i.ResolveUninstallTarget(dataDir, false, defaultNamespace, "foo", "")
		assert.Error(t, err)
	})
	t.Run("missing package name", func(t *testing.T) {
		_, err := i.ResolveUninstallTarget(dataDir, false, defaultNamespace, "", "v0")
		assert.Error(t, err)
	})

	t.Run("--all --namespace local ; no typst.toml", func(t *testing.T) {
		got, err := i.ResolveUninstallTarget(dataDir, true, defaultNamespace, "", "")
		want := "typst/local"

		assert.Equal(t, want, got)
		assert.NoError(t, err)
	})
	t.Run("--all --namespace local ; typst.toml", func(t *testing.T) {
		got, err := i.ResolveUninstallTarget(dataDir, true, defaultNamespace, "foo", "v0")
		assert.Equal(t, "typst/local/foo/v0", got)
		assert.NoError(t, err)
	})
	t.Run("foo --all --namespace local", func(t *testing.T) {
		got, err := i.ResolveUninstallTarget(dataDir, true, defaultNamespace, "foo", "")
		assert.Equal(t, "typst/local/foo", got)
		assert.NoError(t, err)
	})
}
