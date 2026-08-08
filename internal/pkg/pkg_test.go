package pkg_test

import (
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		want    pkg.Ref
		wantErr bool
	}{
		{
			"@preview/cetz:0.5.2",
			pkg.Ref{Namespace: "preview", Name: "cetz", Version: semver.Version{Minor: 5, Patch: 2}},
			false,
		},
		{
			"preview/cetz:0.5.2",
			pkg.Ref{Namespace: "preview", Name: "cetz", Version: semver.Version{Minor: 5, Patch: 2}},
			false,
		},
		{
			"@local/my-pkg:1.2.3",
			pkg.Ref{Namespace: "local", Name: "my-pkg", Version: semver.Version{Major: 1, Minor: 2, Patch: 3}},
			false,
		},
		{"@preview/cetz", pkg.Ref{}, true},
		{"cetz:0.5.2", pkg.Ref{}, true},
		{"@preview/cetz:not-a-version", pkg.Ref{}, true},
		{"@/cetz:0.5.2", pkg.Ref{}, true},
		{"@preview/:0.5.2", pkg.Ref{}, true},
		{"", pkg.Ref{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := pkg.ParseImport(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, pkg.ErrInvalidRef)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRef_String(t *testing.T) {
	t.Parallel()
	ref, err := pkg.New("preview", "cetz", "0.5.2")
	require.NoError(t, err)
	assert.Equal(t, "@preview/cetz:0.5.2", ref.String())
}

func TestParseImport_RoundTrip(t *testing.T) {
	t.Parallel()
	const raw = "@preview/cetz:0.5.2"
	ref, err := pkg.ParseImport(raw)
	require.NoError(t, err)
	assert.Equal(t, raw, ref.String())
}

func TestRef_Segments(t *testing.T) {
	t.Parallel()
	ref, err := pkg.New("local", "my-pkg", "1.2.3")
	require.NoError(t, err)
	assert.Equal(t, []string{"local", "my-pkg", "1.2.3"}, ref.Segments())
}

func TestRef_Dir(t *testing.T) {
	t.Parallel()
	ref, err := pkg.New("local", "my-pkg", "1.2.3")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/data", "local", "my-pkg", "1.2.3"), ref.Dir("/data"))
}

func TestNew_RejectsInvalidParts(t *testing.T) {
	t.Parallel()
	_, err := pkg.New("", "cetz", "0.5.2")
	require.ErrorIs(t, err, pkg.ErrInvalidRef)

	_, err = pkg.New("preview", "", "0.5.2")
	require.ErrorIs(t, err, pkg.ErrInvalidRef)

	_, err = pkg.New("preview", "cetz", "0.5")
	require.ErrorIs(t, err, pkg.ErrInvalidRef)
}
