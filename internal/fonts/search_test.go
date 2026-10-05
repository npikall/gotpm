package fonts_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstall_DeletesAFamilyGotpmInstalled(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	dir := fonts.Dir{Root: t.TempDir()}
	_, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)
	require.NoError(t, err)

	require.NoError(t, dir.Uninstall("open sans", false))

	assert.NoDirExists(t, dir.Path("Open Sans"))
}

func TestUninstall_RefusesAFamilyGotpmDidNotInstall(t *testing.T) {
	t.Parallel()
	dir := fonts.Dir{Root: t.TempDir()}
	mine := handInstalled(t, dir, "lato")

	require.ErrorIs(t, dir.Uninstall("Lato", false), fonts.ErrNotOwned)
	assert.FileExists(t, mine)

	require.NoError(t, dir.Uninstall("Lato", true))
	assert.NoFileExists(t, mine)
}

func TestUninstall_ReportsAMissingFamily(t *testing.T) {
	t.Parallel()
	dir := fonts.Dir{Root: t.TempDir()}

	require.ErrorIs(t, dir.Uninstall("Lato", false), fonts.ErrNotInstalled)
}

func newIndex(t *testing.T, srv *fonttest.Server) fonts.Index {
	t.Helper()
	return fonts.Index{Source: srv.Source(), Path: filepath.Join(t.TempDir(), "font-index.json")}
}

func TestSearch_MatchesTheQueryLooselyAcrossLicenses(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	srv.Publish("ofl", "opensanscondensed", regular)
	srv.Publish("apache", "roboto", "Roboto.ttf")
	srv.Publish("ofl", "lato", "Lato.ttf")

	got, err := newIndex(t, srv).Search(context.Background(), "Open Sans", false)

	require.NoError(t, err)
	assert.Equal(t, []fonts.Listing{
		{Family: "opensans", License: "ofl"},
		{Family: "opensanscondensed", License: "ofl"},
	}, got)
}

func TestSearch_ReusesTheCachedListUntilRefreshed(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	index := newIndex(t, srv)
	_, err := index.Search(context.Background(), "", false)
	require.NoError(t, err)
	srv.Publish("ofl", "inter", "Inter.ttf")
	requests := len(srv.Requests())

	cached, err := index.Search(context.Background(), "inter", false)
	require.NoError(t, err)
	assert.Empty(t, cached, "the cached list does not know the new family yet")
	assert.Len(t, srv.Requests(), requests, "a fresh cache is not refetched")

	refreshed, err := index.Search(context.Background(), "inter", true)
	require.NoError(t, err)
	assert.Equal(t, []fonts.Listing{{Family: "inter", License: "ofl"}}, refreshed)
}

func TestSearch_RefetchesAStaleCache(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	index := newIndex(t, srv)
	index.Now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	_, err := index.Search(context.Background(), "", false)
	require.NoError(t, err)
	srv.Publish("ofl", "inter", "Inter.ttf")

	index.Now = func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) }
	got, err := index.Search(context.Background(), "inter", false)

	require.NoError(t, err)
	assert.Len(t, got, 1)
}

// Not parallel: t.Setenv cannot be used from a parallel test.
func TestGitHubTokenIsSentToTheAPIOnly(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret")
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")

	_, err := ensure(t, srv, fonts.Dir{Root: t.TempDir()}, resolve(t, srv, "Lato"), false)
	require.NoError(t, err)

	for _, r := range srv.Requests() {
		if strings.HasPrefix(r.Path, "/api/") {
			assert.Equal(t, "Bearer secret", r.Authorization, r.Path)
		} else {
			assert.Empty(t, r.Authorization, r.Path)
		}
	}
}
