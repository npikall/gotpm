package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/add"
	"github.com/npikall/gotpm/internal/cmds/font"
	"github.com/npikall/gotpm/internal/cmds/remove"
	"github.com/npikall/gotpm/internal/cmds/sync"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fontDir(t *testing.T) fonts.Dir {
	t.Helper()
	dir, err := fonts.OpenDir()
	require.NoError(t, err)
	return dir
}

func fontContent(t *testing.T, family, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fontDir(t).Path(family), file))
	require.NoError(t, err)
	return string(data)
}

func addLato(t *testing.T) {
	t.Helper()
	require.NoError(t, font.Add(context.Background(), "Lato", &font.Options{}, discardLogger()))
}

// pinLato pins the newest commit of Lato the way a dependency's author would
// have, digests included, without touching the font directory.
func pinLato(t *testing.T, srv *fonttest.Server) lockfile.Font {
	t.Helper()
	pin, err := srv.Source().Resolve(context.Background(), "Lato")
	require.NoError(t, err)
	result, err := fonts.Dir{Root: t.TempDir()}.Ensure(context.Background(), srv.Source(), pin, false)
	require.NoError(t, err)
	return result.Pin
}

func emptyFontDir(t *testing.T) {
	t.Helper()
	require.NoError(t, os.RemoveAll(fontDir(t).Root))
}

func TestRun_RestoresTheLockedCommitOfADeclaredFont(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "pinned"})
	addLato(t)
	srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "newer"})
	emptyFontDir(t)

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	assert.Equal(t, "pinned", fontContent(t, "Lato", "Lato.ttf"))
}

func TestRun_RefusesADeclaredFontTheLockDoesNotPin(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	require.NoError(t, manifest.SetFonts(filepath.Join(project, manifest.FileName), []string{"Lato"}))

	require.ErrorIs(t, sync.Run(&sync.Options{}, discardLogger()), sync.ErrUnpinnedFont)
}

func TestRun_DropsThePinOfAnUndeclaredFont(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	addLato(t)
	require.NoError(t, manifest.SetFonts(filepath.Join(project, manifest.FileName), nil))

	require.ErrorIs(t, sync.Run(&sync.Options{Frozen: true}, discardLogger()), sync.ErrLockOutOfDate)
	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	assert.Empty(t, lockOf(t, project).Fonts)
}

func TestRun_SkipsAHandInstalledFamilyUnlessForced(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	addLato(t)
	emptyFontDir(t)
	mine := filepath.Join(fontDir(t).Path("Lato"), "Mine.ttf")
	require.NoError(t, paths.EnsureDir(filepath.Dir(mine)))
	require.NoError(t, paths.WriteFile(mine, []byte("mine")))

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()), "one font must not fail a sync")
	assert.FileExists(t, mine)

	require.NoError(t, sync.Run(&sync.Options{Force: true}, discardLogger()))
	assert.NoFileExists(t, mine)
	assert.FileExists(t, filepath.Join(fontDir(t).Path("Lato"), "Lato.ttf"))
}

func TestAdd_PinsAndInstallsTheFontsADependencyDeclares(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "theirs"})
	thesis := testrepo.New(t, "thesis", "1.0.0").ReleaseFonts(pinLato(t, srv))
	srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "newer"})

	require.NoError(t, add.Run(thesis.URL(), &add.Options{}, discardLogger()))

	pin, ok := lockOf(t, project).GetFont("Lato")
	require.True(t, ok)
	assert.False(t, pin.Direct)
	assert.Equal(t, []string{thesis.Import()}, pin.RequiredBy)
	assert.Equal(t, "theirs", fontContent(t, "Lato", "Lato.ttf"), "the dependency's pin, not upstream's newest")
	m, err := manifest.LoadFrom(project)
	require.NoError(t, err)
	assert.Empty(t, m.Fonts(), "a transitive font is pinned, not declared")
}

func TestRun_RestoresTheFontsOfADependency(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	thesis := testrepo.New(t, "thesis", "1.0.0").ReleaseFonts(pinLato(t, srv))
	require.NoError(t, add.Run(thesis.URL(), &add.Options{}, discardLogger()))
	emptyFontDir(t)

	require.NoError(t, sync.Run(&sync.Options{}, discardLogger()))

	assert.FileExists(t, filepath.Join(fontDir(t).Path("Lato"), "Lato.ttf"))
}

func TestAdd_TheProjectsOwnFontPinWins(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	thesis := testrepo.New(t, "thesis", "1.0.0").ReleaseFonts(pinLato(t, srv))
	srv.Publish("ofl", "lato", "Lato.ttf")
	addLato(t)
	mine, _ := lockOf(t, project).GetFont("Lato")

	require.NoError(t, add.Run(thesis.URL(), &add.Options{}, discardLogger()))

	pin, _ := lockOf(t, project).GetFont("Lato")
	assert.Equal(t, mine.Hash, pin.Hash)
	assert.True(t, pin.Direct)
	assert.Equal(t, []string{thesis.Import()}, pin.RequiredBy)
}

func TestRemove_DropsTheFontsOnlyTheDependencyRequired(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "my-doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	thesis := testrepo.New(t, "thesis", "1.0.0").ReleaseFonts(pinLato(t, srv))
	require.NoError(t, add.Run(thesis.URL(), &add.Options{}, discardLogger()))

	require.NoError(t, remove.Run(thesis.Import(), &remove.Options{}, discardLogger()))

	assert.Empty(t, lockOf(t, project).Fonts)
}
