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

func serve(t *testing.T, payload string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchReleases(t *testing.T) {
	t.Parallel()
	srv := serve(t, `[{"name":"foobar"},{"name":"baz"}]`)

	got, err := index.FetchReleases(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, []index.Release{{Name: "foobar"}, {Name: "baz"}}, got)
}

func TestFetchReleases_ErrorStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	_, err := index.FetchReleases(context.Background(), srv.URL)
	assert.ErrorIs(t, err, index.ErrHTTPFailedRequest)
}

func TestFetchFrom(t *testing.T) {
	t.Parallel()
	srv := serve(t, `[{"name":"foo","version":"0.1.0"},{"name":"bar","version":"2.3.4"}]`)

	entries, err := index.FetchFrom(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, []index.Entry{{Name: "foo", Version: "0.1.0"}, {Name: "bar", Version: "2.3.4"}}, entries)

	idx := index.Build(entries)
	assert.Equal(t, index.Index{"foo": "0.1.0", "bar": "2.3.4"}, idx)
}

func TestLatestRelease(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		releases []index.Release
		want     string
		wantErr  bool
	}{
		{"trivial", []index.Release{{Name: "0.1.0"}, {Name: "0.2.0"}}, "0.2.0", false},
		{"some invalid version", []index.Release{{Name: "invalid"}, {Name: "0.2.0"}}, "", true},
		{"more complex", []index.Release{{Name: "1.2.1"}, {Name: "1.2.3"}, {Name: "1.3.2"}}, "1.3.2", false},
		{"no releases", nil, "0.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := index.LatestRelease(tt.releases)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()
	t.Run("picks latest version per package", func(t *testing.T) {
		t.Parallel()
		got := index.Build([]index.Entry{
			{Name: "pkg-a", Version: "0.1.0"},
			{Name: "pkg-a", Version: "0.3.0"},
			{Name: "pkg-a", Version: "0.2.0"},
			{Name: "pkg-b", Version: "1.0.0"},
		})
		assert.Equal(t, "0.3.0", got["pkg-a"])
		assert.Equal(t, "1.0.0", got["pkg-b"])
	})
	t.Run("single entry per package", func(t *testing.T) {
		t.Parallel()
		got := index.Build([]index.Entry{{Name: "only", Version: "2.0.0"}})
		assert.Equal(t, "2.0.0", got["only"])
	})
	t.Run("empty input returns empty map", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, index.Build(nil))
	})
	t.Run("invalid version strings skipped gracefully", func(t *testing.T) {
		t.Parallel()
		got := index.Build([]index.Entry{
			{Name: "pkg", Version: "1.0.0"},
			{Name: "pkg", Version: "not-a-version"},
		})
		assert.Equal(t, "1.0.0", got["pkg"])
	})
}

func TestIndex_Latest(t *testing.T) {
	t.Parallel()
	idx := index.Index{"pkg": "1.2.3"}

	version, ok := idx.Latest("pkg")
	assert.True(t, ok)
	assert.Equal(t, "1.2.3", version)

	_, ok = idx.Latest("missing")
	assert.False(t, ok)
}
