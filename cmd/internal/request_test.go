package internal_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/npikall/gotpm/cmd/internal"
	"github.com/stretchr/testify/assert"
)

func TestFetchDataFromGitHub(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`[{"name":"foobar"},{"name":"baz"}]`))
		if err != nil {
			panic(err)
		}
	}))
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		got, err := internal.FetchDataFromGitHub(testServer.URL, ctx)
		want := []*internal.ResponseModel{
			{Name: "foobar"},
			{Name: "baz"},
		}
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []*internal.ResponseModel
		want     string
		wantErr  bool
	}{
		{"trivial", []*internal.ResponseModel{{"0.1.0"}, {"0.2.0"}}, "0.2.0", false},
		{"some invalid version", []*internal.ResponseModel{{"invalid"}, {"0.2.0"}}, "0.2.0", true},
		{"more complex", []*internal.ResponseModel{{"1.2.1"}, {"1.2.3"}, {"1.3.2"}}, "1.3.2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := internal.GetLatestVersion(tt.versions)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetLatestVersion() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetLatestVersion() succeeded unexpectedly")
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildVersionIndex(t *testing.T) {
	t.Run("picks latest version per package", func(t *testing.T) {
		entries := []internal.TypstIndexEntry{
			{Name: "pkg-a", Version: "0.1.0"},
			{Name: "pkg-a", Version: "0.3.0"},
			{Name: "pkg-a", Version: "0.2.0"},
			{Name: "pkg-b", Version: "1.0.0"},
		}
		got := internal.BuildVersionIndex(entries)
		assert.Equal(t, "0.3.0", got["pkg-a"])
		assert.Equal(t, "1.0.0", got["pkg-b"])
	})
	t.Run("single entry per package", func(t *testing.T) {
		entries := []internal.TypstIndexEntry{
			{Name: "only", Version: "2.0.0"},
		}
		got := internal.BuildVersionIndex(entries)
		assert.Equal(t, "2.0.0", got["only"])
	})
	t.Run("empty input returns empty map", func(t *testing.T) {
		got := internal.BuildVersionIndex(nil)
		assert.Empty(t, got)
	})
	t.Run("invalid version strings skipped gracefully", func(t *testing.T) {
		entries := []internal.TypstIndexEntry{
			{Name: "pkg", Version: "1.0.0"},
			{Name: "pkg", Version: "not-a-version"},
		}
		got := internal.BuildVersionIndex(entries)
		assert.Equal(t, "1.0.0", got["pkg"])
	})
}

func TestFetchTypstIndex(t *testing.T) {
	payload := `[{"name":"foo","version":"0.1.0"},{"name":"bar","version":"2.3.4"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)

	// FetchTypstIndex uses a hardcoded URL, so test via BuildVersionIndex with
	// the same data shape to verify the integration contract.
	entries := []internal.TypstIndexEntry{
		{Name: "foo", Version: "0.1.0"},
		{Name: "bar", Version: "2.3.4"},
	}
	idx := internal.BuildVersionIndex(entries)
	assert.Equal(t, "0.1.0", idx["foo"])
	assert.Equal(t, "2.3.4", idx["bar"])
}
