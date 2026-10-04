package deps_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/deps"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectManifest = "[package]\nname = \"doc\"\nversion = \"0.1.0\"\nentrypoint = \"main.typ\"\n"

func TestOpenProjectAt_FindsTheManifestInAParent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, manifest.FileName)
	require.NoError(t, os.WriteFile(file, []byte(projectManifest), 0o600))
	sub := filepath.Join(root, "chapters")
	require.NoError(t, os.Mkdir(sub, 0o750))

	project, err := deps.OpenProjectAt(sub)

	require.NoError(t, err)
	assert.Equal(t, root, project.Dir)
	assert.Equal(t, file, project.File)
	assert.Equal(t, "doc", project.Manifest.Package.Name)
}

func TestOpenProjectAt_SuggestsInitWithoutAManifest(t *testing.T) {
	t.Parallel()

	_, err := deps.OpenProjectAt(t.TempDir())

	require.ErrorIs(t, err, manifest.ErrManifestNotFound)
	assert.Contains(t, err.Error(), "gotpm init")
}

func TestOpenProjectAt_ReportsAnUnreadableManifest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, manifest.FileName), []byte("not = [toml"), 0o600))

	_, err := deps.OpenProjectAt(root)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "gotpm init")
}

func TestOpenProjectReadsTheWorkingDirectory(t *testing.T) { //nolint: paralleltest // t.Chdir
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, manifest.FileName), []byte(projectManifest), 0o600))
	t.Chdir(root)

	project, err := deps.OpenProject()

	require.NoError(t, err)
	assert.Equal(t, "doc", project.Manifest.Package.Name)
}

func TestProjectSetDependenciesUpdatesFileAndMemory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, manifest.FileName), []byte(projectManifest), 0o600))
	project, err := deps.OpenProjectAt(root)
	require.NoError(t, err)
	want := []string{"github.com/a/cetz"}

	require.NoError(t, project.SetDependencies(want))

	assert.Equal(t, want, project.Dependencies())
	reopened, err := deps.OpenProjectAt(root)
	require.NoError(t, err)
	assert.Equal(t, want, reopened.Dependencies())
}

func TestProjectSetDependenciesReportsAMissingFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, manifest.FileName), []byte(projectManifest), 0o600))
	project, err := deps.OpenProjectAt(root)
	require.NoError(t, err)
	require.NoError(t, os.Remove(project.File))

	require.Error(t, project.SetDependencies([]string{"github.com/a/cetz"}))
}
