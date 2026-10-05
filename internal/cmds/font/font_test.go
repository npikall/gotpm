package font_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/cmds/font"
	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const regular = "OpenSans-Regular.ttf"

func discardLogger() *log.Logger { return log.New(io.Discard) }

func fontDir(t *testing.T) fonts.Dir {
	t.Helper()
	dir, err := fonts.OpenDir()
	require.NoError(t, err)
	return dir
}

func declaredFonts(t *testing.T, project string) []string {
	t.Helper()
	m, err := manifest.LoadFrom(project)
	require.NoError(t, err)
	return m.Fonts()
}

func lockOf(t *testing.T, project string) *lockfile.Lock {
	t.Helper()
	lock, err := lockfile.Load(project)
	require.NoError(t, err)
	return lock
}

func add(t *testing.T, name string) error {
	t.Helper()
	return font.Add(context.Background(), name, &font.Options{}, discardLogger())
}

func TestInstall_PutsTheFamilyInTheFontDirectoryOnly(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)

	require.NoError(t, font.Install(context.Background(), "Open Sans", &font.Options{}, discardLogger()))

	assert.FileExists(t, filepath.Join(fontDir(t).Path("Open Sans"), regular))
	assert.Empty(t, declaredFonts(t, project), "install is not a project command")
	assert.NoFileExists(t, lockfile.Path(project))
}

func TestUninstall_DeletesTheFamily(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)
	require.NoError(t, font.Install(context.Background(), "Open Sans", &font.Options{}, discardLogger()))

	require.NoError(t, font.Uninstall("opensans", &font.Options{}, discardLogger()))

	assert.NoDirExists(t, fontDir(t).Path("Open Sans"))
}

func TestAdd_DeclaresPinsAndInstallsTheFamily(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	hash := srv.Publish("ofl", "opensans", regular)

	require.NoError(t, add(t, "Open Sans"))

	assert.Equal(t, []string{"Open Sans"}, declaredFonts(t, project))
	pin, ok := lockOf(t, project).GetFont("Open Sans")
	require.True(t, ok)
	assert.Equal(t, hash, pin.Hash)
	assert.True(t, pin.Direct)
	assert.NotEmpty(t, pin.Files[0].SHA256, "the digest is what makes a later sync verifiable")
	assert.FileExists(t, filepath.Join(fontDir(t).Path("Open Sans"), regular))
}

func TestAdd_AgainUnderAnotherSpellingDeclaresItOnce(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)
	require.NoError(t, add(t, "Open Sans"))
	newer := srv.Publish("ofl", "opensans", regular)

	require.NoError(t, add(t, "opensans"))

	assert.Equal(t, []string{"Open Sans"}, declaredFonts(t, project))
	pin, _ := lockOf(t, project).GetFont("Open Sans")
	assert.Equal(t, newer, pin.Hash, "adding again pins the newest commit")
	assert.Len(t, lockOf(t, project).Fonts, 1)
}

func TestAdd_RefusesAFamilyGotpmDidNotInstall(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)
	mine := filepath.Join(fontDir(t).Path("opensans"), "Mine.ttf")
	require.NoError(t, paths.EnsureDir(filepath.Dir(mine)))
	require.NoError(t, paths.WriteFile(mine, []byte("mine")))

	require.ErrorIs(t, add(t, "Open Sans"), fonts.ErrNotOwned)

	assert.FileExists(t, mine)
	assert.Empty(t, declaredFonts(t, project), "nothing is recorded for a font that was not installed")
	require.NoError(t, font.Add(context.Background(), "Open Sans", &font.Options{Force: true}, discardLogger()))
	assert.NoFileExists(t, mine)
}

func TestRemove_DropsTheDeclarationAndPinButKeepsTheFiles(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)
	srv.Publish("ofl", "lato", "Lato.ttf")
	require.NoError(t, add(t, "Open Sans"))
	require.NoError(t, add(t, "Lato"))

	require.NoError(t, font.Remove("open sans", discardLogger()))

	assert.Equal(t, []string{"Lato"}, declaredFonts(t, project))
	_, pinned := lockOf(t, project).GetFont("Open Sans")
	assert.False(t, pinned)
	assert.FileExists(t, filepath.Join(fontDir(t).Path("Open Sans"), regular),
		"the font directory is shared, so the files stay")
}

func TestRemove_LastFontWritesALockOlderGotpmReads(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	require.NoError(t, add(t, "Lato"))

	require.NoError(t, font.Remove("Lato", discardLogger()))

	data, err := os.ReadFile(lockfile.Path(project))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"version": 1`)
}

func TestRemove_RefusesAFontTheProjectDoesNotDeclare(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	testrepo.Project(t, "doc")

	require.ErrorIs(t, font.Remove("Lato", discardLogger()), font.ErrNotDeclared)
}

func TestSearch_ListsMatchingFamiliesAndToleratesNone(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "opensans", regular)

	require.NoError(t, font.Search(context.Background(), "open", false, discardLogger()))
	require.NoError(t, font.Search(context.Background(), "nothing like it", true, discardLogger()))
}

func TestRemove_KeepsAPinADependencyStillRequires(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	require.NoError(t, add(t, "Lato"))
	lock := lockOf(t, project)
	lock.Upsert(lockfile.Entry{Import: "@gotpm/thesis:1.0.0", Name: "thesis", Version: "1.0.0", Namespace: "gotpm"})
	pin, _ := lock.GetFont("Lato")
	pin.Direct, pin.RequiredBy = false, []string{"@gotpm/thesis:1.0.0"}
	lock.UpsertFont(pin)
	require.NoError(t, lockfile.Save(project, lock))

	require.NoError(t, font.Remove("Lato", discardLogger()))

	_, pinned := lockOf(t, project).GetFont("Lato")
	assert.True(t, pinned)
	assert.Empty(t, declaredFonts(t, project))
}

func TestCommandsOutsideAProjectFail(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	t.Chdir(t.TempDir())

	require.Error(t, add(t, "Lato"))
	require.Error(t, font.Remove("Lato", discardLogger()))
}

func TestInstall_ReportsAnUnknownFont(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	fonttest.Serve(t)

	require.ErrorIs(t, font.Install(context.Background(), "Nope", &font.Options{}, discardLogger()), fonts.ErrFontNotFound)
	require.ErrorIs(t, font.Uninstall("Nope", &font.Options{}, discardLogger()), fonts.ErrNotInstalled)
}

func TestAdd_RecordsDigestsForAFamilyAlreadyInstalled(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	project := testrepo.Project(t, "doc")
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	require.NoError(t, font.Install(context.Background(), "Lato", &font.Options{}, discardLogger()))

	require.NoError(t, add(t, "Lato"))

	pin, _ := lockOf(t, project).GetFont("Lato")
	assert.NotEmpty(t, pin.Files[0].SHA256, "an up-to-date family must not leave the pin unverifiable")
}
