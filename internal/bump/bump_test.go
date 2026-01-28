package bump_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/bump"
	"github.com/stretchr/testify/assert"
)

func TestVersionBump(t *testing.T) {
	bumpTests := []struct {
		got       i.Version
		want      i.Version
		increment string
		name      string
	}{
		{got: i.Version{Major: 0, Minor: 0, Patch: 0}, want: i.Version{Major: 1, Minor: 0, Patch: 0}, increment: i.MAJOR, name: "major simple"},
		{got: i.Version{Major: 0, Minor: 1, Patch: 0}, want: i.Version{Major: 1, Minor: 0, Patch: 0}, increment: i.MAJOR, name: "major with minor"},
		{got: i.Version{Major: 0, Minor: 0, Patch: 1}, want: i.Version{Major: 1, Minor: 0, Patch: 0}, increment: i.MAJOR, name: "major with patch"},
		{got: i.Version{Major: 0, Minor: 1, Patch: 1}, want: i.Version{Major: 1, Minor: 0, Patch: 0}, increment: i.MAJOR, name: "major with minor and patch"},
		{got: i.Version{Major: 1, Minor: 1, Patch: 1}, want: i.Version{Major: 2, Minor: 0, Patch: 0}, increment: i.MAJOR, name: "major with all"},

		{got: i.Version{Major: 0, Minor: 0, Patch: 0}, want: i.Version{Major: 0, Minor: 1, Patch: 0}, increment: i.MINOR, name: "minor simple"},
		{got: i.Version{Major: 0, Minor: 0, Patch: 1}, want: i.Version{Major: 0, Minor: 1, Patch: 0}, increment: i.MINOR, name: "minor with patch"},
		{got: i.Version{Major: 1, Minor: 0, Patch: 0}, want: i.Version{Major: 1, Minor: 1, Patch: 0}, increment: i.MINOR, name: "minor with major"},
		{got: i.Version{Major: 1, Minor: 0, Patch: 1}, want: i.Version{Major: 1, Minor: 1, Patch: 0}, increment: i.MINOR, name: "minor with major and patch"},
		{got: i.Version{Major: 1, Minor: 1, Patch: 1}, want: i.Version{Major: 1, Minor: 2, Patch: 0}, increment: i.MINOR, name: "minor with all"},

		{got: i.Version{Major: 0, Minor: 0, Patch: 0}, want: i.Version{Major: 0, Minor: 0, Patch: 1}, increment: i.PATCH, name: "patch simple"},
		{got: i.Version{Major: 1, Minor: 0, Patch: 0}, want: i.Version{Major: 1, Minor: 0, Patch: 1}, increment: i.PATCH, name: "patch with major"},
		{got: i.Version{Major: 0, Minor: 1, Patch: 0}, want: i.Version{Major: 0, Minor: 1, Patch: 1}, increment: i.PATCH, name: "patch with mino"},
		{got: i.Version{Major: 1, Minor: 1, Patch: 0}, want: i.Version{Major: 1, Minor: 1, Patch: 1}, increment: i.PATCH, name: "patch with major and minor"},
		{got: i.Version{Major: 1, Minor: 1, Patch: 1}, want: i.Version{Major: 1, Minor: 1, Patch: 2}, increment: i.PATCH, name: "patch with all"},
	}
	for _, tt := range bumpTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.got.Bump(tt.increment)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.got, "got %v, want %v", tt.got, tt.want)
		})
	}

	t.Run("wrong increment", func(t *testing.T) {
		startVersion := i.Version{Major: 0, Minor: 0, Patch: 0}
		err := startVersion.Bump("123")
		assertErr(t, err, i.ErrInvalidIncrement.Error())
	})
}

func TestParseVersion(t *testing.T) {
	t.Run("valid version", func(t *testing.T) {
		got, err := i.ParseVersion("0.0.0")
		want := i.NewVersion()
		assert.NoError(t, err)
		assert.Equal(t, want, got, "got %v, want %v", got, want)
	})
	invalidStringTests := []struct {
		name    string
		version string
	}{
		{name: "empty string", version: ""},
		{name: "too little components", version: "0.0"},
		{name: "too many components", version: "0.0.0.0"},
		{name: "too many components", version: "0.0.0."},

		{name: "has letter", version: "a.0.0"},
		{name: "has letter", version: "0.a.0"},
		{name: "has letter", version: "0.0.a"},
		{name: "has letter", version: "a.a.a"},

		{name: "has negative", version: "-1.0.0"},
		{name: "has negative", version: "0.-1.0"},
		{name: "has negative", version: "0.0.-1"},
	}
	for _, tt := range invalidStringTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := i.ParseVersion(tt.version)
			assertErr(t, err, i.ErrInvalidVersion.Error())
		})
	}
}

func TestIsValidSemver(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want bool
	}{
		{"correct semver", "0.1.2", true},
		{"short semver", "0.1", false},
		{"short semver with letters", "a.1", false},
		{"long semver", "0.1.2.3", false},
		{"long semver with letters", "0.1.2.a", false},
		{"prerelease", "0.1.2-a", false},
		{"extra", "0.1.2+a", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := i.IsSemVer(tt.got)
			assert.Equal(t, tt.want, got, "given '%s'", tt.got)
		})
	}
}

func assertErr(t testing.TB, got error, want string) {
	t.Helper()
	if got == nil {
		t.Fatal("wanted an error but did not get one")
	}
	if got.Error() != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a    i.Version
		b    i.Version
		want int
	}{
		{"simple bigger patch", i.Version{0, 0, 1}, i.Version{0, 0, 0}, 1},
		{"simple bigger minor", i.Version{0, 1, 1}, i.Version{0, 0, 0}, 1},
		{"simple bigger major", i.Version{1, 1, 1}, i.Version{0, 0, 0}, 1},
		{"simple smaller patch", i.Version{0, 0, 1}, i.Version{0, 0, 2}, -1},
		{"simple smaller minor", i.Version{0, 1, 1}, i.Version{0, 2, 2}, -1},
		{"simple smaller major", i.Version{1, 1, 1}, i.Version{2, 2, 2}, -1},
		{"equal versions", i.Version{2, 2, 2}, i.Version{2, 2, 2}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := i.CompareVersions(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
