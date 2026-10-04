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

func TestNormalize_SeparatesThePackagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		canonical string
		cloneURL  string
		path      string
	}{
		{"bare path", "github.com/a/mono//pkg", "github.com/a/mono", "https://github.com/a/mono", "pkg"},
		{"nested path", "github.com/a/mono//packages/pkg", "github.com/a/mono", "https://github.com/a/mono", "packages/pkg"},
		{"https url", "https://github.com/a/mono//pkg", "github.com/a/mono", "https://github.com/a/mono", "pkg"},
		{"git suffix", "https://github.com/a/mono.git//pkg", "github.com/a/mono", "https://github.com/a/mono.git", "pkg"},
		{"scp style ssh", "git@github.com:a/mono.git//pkg", "github.com/a/mono", "git@github.com:a/mono.git", "pkg"},
		{"gitlab subgroup", "gitlab.com/g/sub/mono//pkg", "gitlab.com/g/sub/mono", "https://gitlab.com/g/sub/mono", "pkg"},
		{"surrounding slashes", "github.com/a/mono///pkg/", "github.com/a/mono", "https://github.com/a/mono", "pkg"},
		{"local repository", "file:///tmp/mono//pkg", "file:///tmp/mono", "file:///tmp/mono", "pkg"},
		{"no package path", "github.com/a/mono", "github.com/a/mono", "https://github.com/a/mono", ""},
		{"empty package path", "github.com/a/mono//", "github.com/a/mono", "https://github.com/a/mono", ""},
		{"package path of the root", "github.com/a/mono//.", "github.com/a/mono", "https://github.com/a/mono", ""},
		{"package path back to the root", "github.com/a/mono//pkg/..", "github.com/a/mono", "https://github.com/a/mono", ""},
		{"local without package path", "file:///tmp/mono", "file:///tmp/mono", "file:///tmp/mono", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolve.Normalize(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.canonical, got.Canonical, "canonical form")
			assert.Equal(t, tt.cloneURL, got.CloneURL, "clone url")
			assert.Equal(t, tt.path, got.Path, "package path")
		})
	}
}

func TestNormalize_RejectsAPackagePathLeavingTheRepository(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"github.com/a/mono//..",
		"github.com/a/mono//pkg/../../etc",
		"file:///tmp/mono//../other",
	} {
		_, err := resolve.Normalize(raw)
		require.ErrorIs(t, err, resolve.ErrInvalidRepoURL, raw)
	}
}

func TestSource_StringJoinsThePackagePath(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"github.com/a/mono//pkg/sub", "github.com/a/mono"} {
		got, err := resolve.Normalize(raw)
		require.NoError(t, err)
		assert.Equal(t, raw, got.String())
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
		{"url without host", "https:///a/cetz"},
		{"unparsable url", "https://git hub.com/a/cetz"},
		{"scp without host", "git@:a/cetz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolve.Normalize(tt.raw)
			require.ErrorIs(t, err, resolve.ErrInvalidRepoURL)
		})
	}
}

func TestPackagePathHint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"names a subdirectory without the separator", "github.com/a/mono/pkg", "github.com/a/mono//path/to/package"},
		{"names only the repository", "github.com/a/mono", ""},
		{"already has a package path", "github.com/a/mono//pkg", ""},
		{"is a local repository", "file:///tmp/mono/pkg", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src, err := resolve.Normalize(tt.raw)
			require.NoError(t, err)

			hint := resolve.PackagePathHint(src)

			if tt.want == "" {
				assert.Empty(t, hint)
				return
			}
			assert.Contains(t, hint, tt.want)
		})
	}
}
