package fonttest_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func get(t *testing.T, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

func TestServerServesEveryCommitOfAFamily(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	first := srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "one"})
	srv.PublishContents("ofl", "lato", map[string]string{"Lato.ttf": "two"})
	google := srv.Source()

	pin, err := google.Resolve(context.Background(), "Lato")
	require.NoError(t, err)
	newest, err := google.Download(context.Background(), pin, "Lato.ttf")
	require.NoError(t, err)
	pin.Hash = first
	older, err := google.Download(context.Background(), pin, "Lato.ttf")
	require.NoError(t, err)

	assert.Equal(t, "two", string(newest))
	assert.Equal(t, "one", string(older))
	assert.Equal(t, 2, srv.Downloads())
	assert.NotEmpty(t, srv.Requests())
}

func TestServerListsFamiliesByLicense(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	srv.Publish("apache", "roboto", "Roboto.ttf")

	got, err := srv.Source().Families(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []fonts.Listing{{Family: "lato", License: "ofl"}, {Family: "roboto", License: "apache"}}, got)
}

func TestServerAnswersNotFoundForAnythingElse(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "lato", "Lato.ttf")
	base := srv.Source()

	for _, url := range []string{
		base.RawURL + "/main/ofl/lato/Missing.ttf",
		base.RawURL + "/unknown-commit/ofl/lato/Lato.ttf",
		base.RawURL + "/main/ofl",
		base.APIURL + "/git/trees/somewhere",
		base.APIURL + "/unknown",
	} {
		assert.Equal(t, http.StatusNotFound, get(t, url), url)
	}
}

func TestServeBecomesTheDefaultSource(t *testing.T) { //nolint: paralleltest // Serve replaces the registered source
	srv := fonttest.Serve(t)

	assert.Equal(t, srv.Source(), fonts.Default())
}
