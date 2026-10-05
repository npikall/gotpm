package fonts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateDataDir points gotpm's data directory into a temporary directory.
func isolateDataDir(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("HOME", data)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("APPDATA", data)
	dataDir, err := paths.GotpmDataDir()
	require.NoError(t, err)
	return dataDir
}

func TestOpenDirAndIndexLiveInTheDataDirectory(t *testing.T) { //nolint: paralleltest // t.Setenv
	dataDir := isolateDataDir(t)

	dir, err := fonts.OpenDir()
	require.NoError(t, err)
	index, err := fonts.OpenIndex()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dataDir, "fonts"), dir.Root)
	assert.Equal(t, filepath.Join(dataDir, "font-index.json"), index.Path)
	assert.Equal(t, fonts.GoogleSource, index.Source.Name())
}

func TestClearIndex_DeletesTheCachedListAndToleratesNone(t *testing.T) { //nolint: paralleltest // t.Setenv
	isolateDataDir(t)
	path, err := fonts.IndexPath()
	require.NoError(t, err)
	require.NoError(t, paths.EnsureDir(filepath.Dir(path)))
	require.NoError(t, paths.WriteFile(path, []byte("{}")))

	require.NoError(t, fonts.ClearIndex())
	assert.NoFileExists(t, path)
	require.NoError(t, fonts.ClearIndex(), "a missing index is already cleared")
}

func TestSourceFor_RefusesAnUnknownSource(t *testing.T) {
	t.Parallel()

	_, err := fonts.SourceFor("somewhere-else")

	require.ErrorIs(t, err, fonts.ErrUnknownSource)
}

func TestEnsureAll_InstallsEveryPinAndSkipsForeignFamilies(t *testing.T) { //nolint: paralleltest // Serve replaces the registered source
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	srv.Publish("ofl", "inter", "Inter[wght].ttf")
	dir := fonts.Dir{Root: t.TempDir()}
	handInstalled(t, dir, "lato")
	pins := []lockfile.Font{locked(t, srv, "Lato"), locked(t, srv, "Inter")}

	results, err := dir.EnsureAll(context.Background(), pins, false)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, fonts.Skipped, results[0].Outcome)
	info, warning := results[0].Notes(dir)
	assert.Empty(t, info)
	assert.Contains(t, warning, "pass --force")
	assert.Equal(t, fonts.Installed, results[1].Outcome)
	info, warning = results[1].Notes(dir)
	assert.Empty(t, info)
	assert.Contains(t, warning, "only variable fonts")
}

// locked pins a family with its digests, the way a lock records it.
func locked(t *testing.T, srv *fonttest.Server, family string) lockfile.Font {
	t.Helper()
	result, err := ensure(t, srv, fonts.Dir{Root: t.TempDir()}, resolve(t, srv, family), false)
	require.NoError(t, err)
	return result.Pin
}

func TestEnsureAll_RefusesAPinFromAnUnknownSource(t *testing.T) {
	t.Parallel()
	pin := lockfile.Font{Family: "Lato", Source: "somewhere-else"}

	_, err := fonts.Dir{Root: t.TempDir()}.EnsureAll(context.Background(), []lockfile.Font{pin}, false)

	require.ErrorIs(t, err, fonts.ErrUnknownSource)
}

func TestNotes_NameTheReplacedCommit(t *testing.T) {
	t.Parallel()
	result := fonts.Result{
		Pin:            lockfile.Font{Family: "Lato", Hash: "0123456789abcdef"},
		Outcome:        fonts.Replaced,
		ReplacedSource: fonts.Provenance{Hash: "fedcba9876543210"},
	}

	info, warning := result.Notes(fonts.Dir{})

	assert.Equal(t, `replaced font "Lato" fedcba98 -> 01234567`, info)
	assert.Empty(t, warning)
	result.ReplacedSource = fonts.Provenance{}
	info, _ = result.Notes(fonts.Dir{})
	assert.Equal(t, `replaced font "Lato" an unknown source -> 01234567`, info)
}

func TestDownload_RefusesAPinOutsideTheRepository(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	for _, url := range []string{
		"github.com/evil/fonts//ofl/lato",
		"github.com/google/fonts//secret/lato",
		"github.com/google/fonts//ofl/../lato",
		"github.com/google/fonts//ofl/",
	} {
		pin := lockfile.Font{Family: "Lato", URL: url, Hash: "abc"}
		_, err := srv.Source().Download(context.Background(), pin, "Lato.ttf")
		require.ErrorIs(t, err, fonts.ErrInvalidFileRef, url)
	}
}

func TestEnsure_RefusesAPinnedFileNameEscapingTheFamily(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	pin := resolve(t, srv, "Lato")
	pin.Files = []lockfile.FontFile{{Name: ".."}}

	_, err := ensure(t, srv, fonts.Dir{Root: t.TempDir()}, pin, false)

	require.ErrorIs(t, err, fonts.ErrInvalidFileRef)
}

func TestResolve_ReportsAFailingServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	google := fonts.Google{RawURL: srv.URL, APIURL: srv.URL}

	_, err := google.Resolve(context.Background(), "Lato")
	require.ErrorIs(t, err, fonts.ErrFailedRequest)
	_, err = google.Families(context.Background())
	require.ErrorIs(t, err, fonts.ErrFailedRequest)
}

func TestResolve_ReportsAnUnreachableServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	_, err := fonts.Google{RawURL: url, APIURL: url}.Resolve(context.Background(), "Lato")

	require.Error(t, err)
}

func TestIndex_IgnoresAnUnreadableCache(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	index := newIndex(t, srv)
	require.NoError(t, os.WriteFile(index.Path, []byte("not json"), paths.FilePerm))

	got, err := index.Search(context.Background(), "lato", false)

	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestEnsureAll_RefusesALockedPinWithoutDigests(t *testing.T) { //nolint: paralleltest // Serve replaces the registered source
	srv := fonttest.Serve(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	pin := resolve(t, srv, "Lato")

	_, err := fonts.Dir{Root: t.TempDir()}.EnsureAll(context.Background(), []lockfile.Font{pin}, false)

	require.ErrorIs(t, err, fonts.ErrChecksum, "a pin read from a lock is only reproducible with its digests")
}
