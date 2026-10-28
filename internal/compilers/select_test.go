package compilers_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/compilers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stable lists releases the way compilers.Stable does, newest first.
func stable(versions ...string) []compilers.Release {
	releases := make([]compilers.Release, 0, len(versions))
	for _, version := range versions {
		releases = append(releases, compilers.Release{Version: version})
	}
	return releases
}

func TestSelect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		floor     string
		pin       string
		installed []string
		releases  []compilers.Release // nil: the release source must not be asked
		want      compilers.Selection
	}{
		{
			name:      "a pin satisfying the floor wins over newer compilers",
			floor:     "0.12.0",
			pin:       "0.12.0",
			installed: []string{"0.12.0", "0.14.0"},
			want:      compilers.Selection{Version: "0.12.0"},
		},
		{
			name:  "a pin is kept even when it is not installed",
			floor: "0.12.0",
			pin:   "0.13.1",
			want:  compilers.Selection{Version: "0.13.1"},
		},
		{
			name:      "a pinned pre-release is kept",
			floor:     "0.12.0",
			pin:       "0.14.0-rc.1",
			installed: []string{"0.13.1"},
			want:      compilers.Selection{Version: "0.14.0-rc.1"},
		},
		{
			name:      "no pin takes the newest installed stable compiler satisfying the floor",
			floor:     "0.12.0",
			installed: []string{"0.11.0", "0.12.0", "0.13.1", "0.14.0-rc.1"},
			want:      compilers.Selection{Version: "0.13.1", NewPin: true},
		},
		{
			name:      "no pin and nothing installed takes the latest stable release satisfying the floor",
			floor:     "0.12.0",
			installed: nil,
			releases:  stable("0.14.0", "0.13.1", "0.11.0"),
			want:      compilers.Selection{Version: "0.14.0", NewPin: true},
		},
		{
			name:      "nothing installed satisfies the floor",
			floor:     "0.13.0",
			installed: []string{"0.11.0", "0.12.0"},
			releases:  stable("0.13.1", "0.12.0"),
			want:      compilers.Selection{Version: "0.13.1", NewPin: true},
		},
		{
			name:      "only pre-releases installed",
			floor:     "0.12.0",
			installed: []string{"0.14.0-rc.1"},
			releases:  stable("0.13.1"),
			want:      compilers.Selection{Version: "0.13.1", NewPin: true},
		},
		{
			name:      "a pin below the floor is raised",
			floor:     "0.13.0",
			pin:       "0.12.0",
			installed: []string{"0.12.0", "0.13.1"},
			want:      compilers.Selection{Version: "0.13.1", NewPin: true},
		},
		{
			name:      "a pin below the floor is raised to a release when none installed satisfies it",
			floor:     "0.13.0",
			pin:       "0.12.0",
			installed: []string{"0.12.0"},
			releases:  stable("0.14.0", "0.13.1"),
			want:      compilers.Selection{Version: "0.14.0", NewPin: true},
		},
		{
			name:      "a pre-release of the floor's version does not satisfy it",
			floor:     "0.14.0",
			pin:       "0.14.0-rc.1",
			installed: []string{"0.14.0-rc.1"},
			releases:  stable("0.14.0"),
			want:      compilers.Selection{Version: "0.14.0", NewPin: true},
		},
		{
			name:      "no floor takes the newest installed stable compiler",
			installed: []string{"0.11.0", "0.13.1"},
			want:      compilers.Selection{Version: "0.13.1", NewPin: true},
		},
		{
			name:     "no floor and nothing installed takes the latest stable release",
			releases: stable("0.14.0", "0.13.1"),
			want:     compilers.Selection{Version: "0.14.0", NewPin: true},
		},
		{
			name: "no floor keeps any pin",
			pin:  "0.11.0",
			want: compilers.Selection{Version: "0.11.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			releases := func() ([]compilers.Release, error) {
				require.NotNil(t, tt.releases, "the release source was asked without need")
				return tt.releases, nil
			}

			got, err := compilers.Select(tt.floor, tt.pin, tt.installed, releases)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSelect_FailsWhenNoStableReleaseSatisfiesTheFloor(t *testing.T) {
	t.Parallel()
	releases := func() ([]compilers.Release, error) { return stable("0.13.1"), nil }

	_, err := compilers.Select("0.14.0", "", []string{"0.15.0-rc.1"}, releases)

	require.ErrorIs(t, err, compilers.ErrNoRelease)
	assert.ErrorContains(t, err, "0.14.0")
}

func TestSelect_PassesOnTheReleaseSourceFailing(t *testing.T) {
	t.Parallel()
	releases := func() ([]compilers.Release, error) { return nil, assert.AnError }

	_, err := compilers.Select("", "", nil, releases)

	require.ErrorIs(t, err, assert.AnError)
}

func TestSelect_RejectsAnInvalidFloorOrPin(t *testing.T) {
	t.Parallel()
	none := func() ([]compilers.Release, error) { return nil, nil }

	_, err := compilers.Select("0.12", "", nil, none)
	require.ErrorIs(t, err, compilers.ErrInvalidVersion)

	_, err = compilers.Select("0.12.0", "latest", nil, none)
	require.ErrorIs(t, err, compilers.ErrInvalidVersion)
}
