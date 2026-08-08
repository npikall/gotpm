package semver_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v is a terse constructor so the table rows below stay readable.
func v(major, minor, patch int) semver.Version {
	return semver.Version{Major: major, Minor: minor, Patch: patch}
}

func TestIsValidVersion(t *testing.T) {
	t.Parallel()
	valid := []string{
		"0.0.0", "1.0.0", "0.1.0", "0.0.1",
		"1.2.3", "10.20.30", "999.0.0",
	}
	invalid := []string{
		"", "1", "1.2", "1.2.3.4",
		"v1.2.3", "1.2.3-alpha", "1.2.3+build",
		"01.2.3", "1.02.3", "1.2.03",
		"-1.0.0", "1.-1.0", "a.b.c",
	}
	for _, s := range valid {
		assert.True(t, semver.IsValidVersion(s), "expected %q to be valid semver", s)
	}
	for _, s := range invalid {
		assert.False(t, semver.IsValidVersion(s), "expected %q to be invalid semver", s)
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		want    semver.Version
		wantErr bool
	}{
		{"0.0.0", v(0, 0, 0), false},
		{"1.2.3", v(1, 2, 3), false},
		{"10.20.30", v(10, 20, 30), false},
		{"999.0.0", v(999, 0, 0), false},
		{"invalid", semver.Version{}, true},
		{"1.2", semver.Version{}, true},
		{"v1.0.0", semver.Version{}, true},
		{"01.2.3", semver.Version{}, true},
		{"-1.0.0", semver.Version{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := semver.Parse(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, semver.ErrInvalidVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestVersion_String(t *testing.T) {
	t.Parallel()
	one := v(1, 2, 3)
	zero := v(0, 0, 0)
	big := v(10, 0, 99)
	assert.Equal(t, "1.2.3", one.String())
	assert.Equal(t, "0.0.0", zero.String())
	assert.Equal(t, "10.0.99", big.String())
}

func TestVersion_Bump(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		start     semver.Version
		increment string
		want      semver.Version
		wantErr   bool
	}{
		{"patch increments patch", v(1, 2, 3), semver.PATCH, v(1, 2, 4), false},
		{"minor resets patch", v(1, 2, 3), semver.MINOR, v(1, 3, 0), false},
		{"major resets minor and patch", v(1, 2, 3), semver.MAJOR, v(2, 0, 0), false},
		{"patch from zero", v(0, 0, 0), semver.PATCH, v(0, 0, 1), false},
		{"minor from zero", v(0, 0, 0), semver.MINOR, v(0, 1, 0), false},
		{"major from zero", v(0, 0, 0), semver.MAJOR, v(1, 0, 0), false},
		{"invalid increment", v(1, 0, 0), "invalid", v(1, 0, 0), true},
		{"empty increment", v(1, 0, 0), "", v(1, 0, 0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.start
			err := got.Bump(tt.increment)
			if tt.wantErr {
				require.ErrorIs(t, err, semver.ErrInvalidIncrement)
				assert.Equal(t, tt.start, got, "version must not change on error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b semver.Version
		want int
	}{
		{"equal", v(1, 2, 3), v(1, 2, 3), 0},
		{"major a>b", v(2, 0, 0), v(1, 9, 9), 1},
		{"major a<b", v(1, 9, 9), v(2, 0, 0), -1},
		{"minor a>b", v(1, 2, 0), v(1, 1, 9), 1},
		{"minor a<b", v(1, 1, 9), v(1, 2, 0), -1},
		{"patch a>b", v(1, 2, 4), v(1, 2, 3), 1},
		{"patch a<b", v(1, 2, 3), v(1, 2, 4), -1},
		{"zeros equal", v(0, 0, 0), v(0, 0, 0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.a.Compare(&tt.b))
		})
	}
}
