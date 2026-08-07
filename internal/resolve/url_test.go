package resolve_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		raw       string
		canonical string
		cloneURL  string
	}{
		{"bare path", "github.com/a/cetz", "github.com/a/cetz", "https://github.com/a/cetz"},
		{"git suffix", "github.com/a/cetz.git", "github.com/a/cetz", "https://github.com/a/cetz"},
		{"trailing slash", "github.com/a/cetz/", "github.com/a/cetz", "https://github.com/a/cetz"},
		{"https url", "https://github.com/a/cetz", "github.com/a/cetz", "https://github.com/a/cetz"},
		{"https url with suffix", "https://github.com/a/cetz.git", "github.com/a/cetz", "https://github.com/a/cetz.git"},
		{"http url", "http://gitea.example.com/a/cetz", "gitea.example.com/a/cetz", "http://gitea.example.com/a/cetz"},
		{"scp style ssh", "git@github.com:a/cetz.git", "github.com/a/cetz", "git@github.com:a/cetz.git"},
		{"ssh url", "ssh://git@github.com/a/cetz", "github.com/a/cetz", "ssh://git@github.com/a/cetz"},
		{"gitlab subgroup", "gitlab.com/g/sub/cetz", "gitlab.com/g/sub/cetz", "https://gitlab.com/g/sub/cetz"},
		{"surrounding space", "  github.com/a/cetz  ", "github.com/a/cetz", "https://github.com/a/cetz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolve.Normalize(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.canonical, got.Canonical, "canonical form")
			assert.Equal(t, tt.cloneURL, got.CloneURL, "clone url")
		})
	}
}

func TestNormalize_SpellingsOfTheSameRepositoryAgree(t *testing.T) {
	t.Parallel()

	// The canonical form is what gotpm.lock and a package's provenance are
	// keyed on, so two spellings of one repository must not look like two.
	spellings := []string{
		"github.com/a/cetz",
		"github.com/a/cetz.git",
		"https://github.com/a/cetz",
		"git@github.com:a/cetz.git",
	}
	for _, spelling := range spellings {
		got, err := resolve.Normalize(spelling)
		require.NoError(t, err, spelling)
		assert.Equal(t, "github.com/a/cetz", got.Canonical, spelling)
	}
}

func TestNormalize_KeepsSSHSoCredentialsKeepWorking(t *testing.T) {
	t.Parallel()

	got, err := resolve.Normalize("git@github.com:a/cetz.git")
	require.NoError(t, err)
	assert.Equal(t, "git@github.com:a/cetz.git", got.CloneURL,
		"a user who asked for ssh must not be silently switched to https")
}

func TestNormalize_Rejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"blank", "   "},
		{"no owner", "github.com/cetz"},
		{"host only", "github.com"},
		{"not a host", "some/owner/repo"},
		{"empty segment", "github.com//cetz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolve.Normalize(tt.raw)
			require.ErrorIs(t, err, resolve.ErrInvalidRepoURL)
		})
	}
}
