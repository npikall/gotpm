package version_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/version"
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

func assertErr(t testing.TB, got error, want string) {
	t.Helper()
	if got == nil {
		t.Fatal("wanted an error but did not get one")
	}
	if got.Error() != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
