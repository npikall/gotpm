package internal_test

import (
	"testing"

	. "github.com/npikall/gotpm/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSemVer(t *testing.T) {
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
		assert.True(t, IsSemVer(s), "expected %q to be valid semver", s)
	}
	for _, s := range invalid {
		assert.False(t, IsSemVer(s), "expected %q to be invalid semver", s)
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"0.0.0", Version{0, 0, 0}, false},
		{"1.2.3", Version{1, 2, 3}, false},
		{"10.20.30", Version{10, 20, 30}, false},
		{"999.0.0", Version{999, 0, 0}, false},
		{"invalid", Version{}, true},
		{"1.2", Version{}, true},
		{"v1.0.0", Version{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseVersion(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewVersion(t *testing.T) {
	t.Parallel()
	v := NewVersion()
	assert.Equal(t, Version{0, 0, 0}, v)
}

func TestVersion_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.2.3", Version{1, 2, 3}.String())
	assert.Equal(t, "0.0.0", Version{0, 0, 0}.String())
	assert.Equal(t, "10.0.99", Version{10, 0, 99}.String())
}

func TestVersion_Bump(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		start     Version
		increment string
		want      Version
		wantErr   bool
	}{
		{"patch increments patch", Version{1, 2, 3}, PATCH, Version{1, 2, 4}, false},
		{"minor increments minor, resets patch", Version{1, 2, 3}, MINOR, Version{1, 3, 0}, false},
		{"major increments major, resets minor and patch", Version{1, 2, 3}, MAJOR, Version{2, 0, 0}, false},
		{"patch from zero", Version{0, 0, 0}, PATCH, Version{0, 0, 1}, false},
		{"minor from zero", Version{0, 0, 0}, MINOR, Version{0, 1, 0}, false},
		{"major from zero", Version{0, 0, 0}, MAJOR, Version{1, 0, 0}, false},
		{"invalid increment", Version{1, 0, 0}, "invalid", Version{1, 0, 0}, true},
		{"empty increment", Version{1, 0, 0}, "", Version{1, 0, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := tt.start
			err := v.Bump(tt.increment)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidIncrement)
				assert.Equal(t, tt.start, v, "version must not change on error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b Version
		want int
	}{
		{"equal", Version{1, 2, 3}, Version{1, 2, 3}, 0},
		{"major a>b", Version{2, 0, 0}, Version{1, 9, 9}, 1},
		{"major a<b", Version{1, 9, 9}, Version{2, 0, 0}, -1},
		{"minor a>b", Version{1, 2, 0}, Version{1, 1, 9}, 1},
		{"minor a<b", Version{1, 1, 9}, Version{1, 2, 0}, -1},
		{"patch a>b", Version{1, 2, 4}, Version{1, 2, 3}, 1},
		{"patch a<b", Version{1, 2, 3}, Version{1, 2, 4}, -1},
		{"zeros equal", Version{0, 0, 0}, Version{0, 0, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CompareVersions(tt.a, tt.b))
		})
	}
}
