package index_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/npikall/gotpm/internal/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const indexPayload = `[{"name":"cetz","version":"0.3.0"},{"name":"cetz","version":"0.4.0"}]`

// serveIndex points Load at a server answering with payload, or failing when
// payload is empty.
func serveIndex(t *testing.T, payload string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if payload == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)
	index.UseIndexURL(t, srv.URL)
}

func TestLoad_FetchesAndCaches(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	serveIndex(t, indexPayload)

	idx, err := index.Load(context.Background(), index.Opts{})
	require.NoError(t, err)
	assert.Equal(t, index.Index{"cetz": "0.4.0"}, idx)

	cache, err := index.LoadCache()
	require.NoError(t, err)
	assert.Equal(t, idx, cache.Index)
}

func TestLoad_PrefersValidCache(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	serveIndex(t, "")
	require.NoError(t, index.SaveCache(index.Index{"cached": "1.0.0"}))

	idx, err := index.Load(context.Background(), index.Opts{})
	require.NoError(t, err)
	assert.Equal(t, index.Index{"cached": "1.0.0"}, idx)
}

func TestLoad_NoCacheBypassesCache(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	serveIndex(t, indexPayload)
	require.NoError(t, index.SaveCache(index.Index{"cached": "1.0.0"}))

	idx, err := index.Load(context.Background(), index.Opts{NoCache: true})
	require.NoError(t, err)
	assert.Equal(t, index.Index{"cetz": "0.4.0"}, idx)

	cache, err := index.LoadCache()
	require.NoError(t, err)
	assert.Equal(t, index.Index{"cached": "1.0.0"}, cache.Index, "--no-cache leaves the cache alone")
}

func TestLoad_FetchFails(t *testing.T) { //nolint: paralleltest
	redirectCacheToTempDir(t)
	serveIndex(t, "")

	for _, opts := range []index.Opts{{}, {NoCache: true}} {
		_, err := index.Load(context.Background(), opts)
		require.ErrorIs(t, err, index.ErrHTTPFailedRequest)
	}
}
