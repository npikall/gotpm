package compilers_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/compilers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		requirements []compilers.Requirement
		want         compilers.Requirement
	}{
		{
			name: "no requirement is no floor",
			want: compilers.Requirement{},
		},
		{
			name:         "the project's own requirement",
			requirements: []compilers.Requirement{{Version: "0.12.0"}},
			want:         compilers.Requirement{Version: "0.12.0"},
		},
		{
			name: "a dependency requiring a newer compiler raises the floor",
			requirements: []compilers.Requirement{
				{Version: "0.12.0"},
				{Version: "0.13.0", Of: "@gotpm/a:0.1.0"},
				{Version: "0.12.1", Of: "@gotpm/b:0.1.0"},
			},
			want: compilers.Requirement{Version: "0.13.0", Of: "@gotpm/a:0.1.0"},
		},
		{
			name: "a dependency raises a project without a requirement",
			requirements: []compilers.Requirement{
				{},
				{Version: "0.13.0", Of: "@gotpm/a:0.1.0"},
			},
			want: compilers.Requirement{Version: "0.13.0", Of: "@gotpm/a:0.1.0"},
		},
		{
			name: "requirements are compared as versions, not strings",
			requirements: []compilers.Requirement{
				{Version: "0.9.0"},
				{Version: "0.10.0", Of: "@gotpm/a:0.1.0"},
			},
			want: compilers.Requirement{Version: "0.10.0", Of: "@gotpm/a:0.1.0"},
		},
		{
			name: "a tie is the first stating it",
			requirements: []compilers.Requirement{
				{Version: "0.13.0"},
				{Version: "0.13.0", Of: "@gotpm/a:0.1.0"},
			},
			want: compilers.Requirement{Version: "0.13.0"},
		},
		{
			name: "a dependency without a requirement contributes nothing",
			requirements: []compilers.Requirement{
				{Version: "0.12.0"},
				{Of: "@gotpm/a:0.1.0"},
			},
			want: compilers.Requirement{Version: "0.12.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := compilers.Floor(tt.requirements)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFloor_RejectsAnInvalidRequirementNamingWhoStatesIt(t *testing.T) {
	t.Parallel()
	_, err := compilers.Floor([]compilers.Requirement{
		{Version: "0.12.0"},
		{Version: "latest", Of: "@gotpm/a:0.1.0"},
	})

	require.ErrorIs(t, err, compilers.ErrInvalidVersion)
	assert.Contains(t, err.Error(), "@gotpm/a:0.1.0")
}
