package request_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/npikall/gotpm/internal/request"
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
		got, err := request.FetchDataFromGitHub(testServer.URL, ctx)
		want := []*request.ResponseModel{
			{Name: "foobar"},
			{Name: "baz"},
		}
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestValidateVersions(t *testing.T) {
	tests := []struct {
		name string
		resp request.ResponseModel
		want bool
		err  error
	}{
		{name: "valid response", resp: request.ResponseModel{Name: "0.0.1"}, want: true, err: nil},
		{name: "invalid response", resp: request.ResponseModel{Name: "a.b.c"}, want: false, err: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := request.ValidateVersion(tt.resp)
			assert.Equal(t, tt.want, got)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []*request.ResponseModel
		want     string
		wantErr  bool
	}{
		{"trivial", []*request.ResponseModel{{"0.1.0"}, {"0.2.0"}}, "0.2.0", false},
		{"some invalid version", []*request.ResponseModel{{"invalid"}, {"0.2.0"}}, "0.2.0", true},
		{"more complex", []*request.ResponseModel{{"1.2.1"}, {"1.2.3"}, {"1.3.2"}}, "1.3.2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := request.GetLatestVersion(tt.versions)
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
