package uninstall_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/uninstall"
	"github.com/stretchr/testify/assert"
)

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
