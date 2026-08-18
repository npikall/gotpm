package publish_test

import (
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/publish"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultForkPathSeparatesForks covers the bug this layout exists to fix:
// a user publishing on behalf of two owners must not have the second fork's
// submissions committed into the first fork's clone.
func TestDefaultForkPathSeparatesForks(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	dataDir, err := paths.GotpmDataDir()
	require.NoError(t, err)

	mine, err := publish.DefaultForkPath("https://github.com/me/packages")
	require.NoError(t, err)
	theirs, err := publish.DefaultForkPath("https://github.com/acme/packages")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dataDir, "forks", "github.com", "me", "packages"), mine)
	assert.NotEqual(t, mine, theirs)
}

// TestDefaultForkPathIgnoresSpelling covers the same fork written two ways.
// Deriving two locations from them would clone a repository gotpm already has,
// which for a fork of typst/packages is the cost the sparse cloning is there
// to avoid.
func TestDefaultForkPathIgnoresSpelling(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)

	https, err := publish.DefaultForkPath("https://github.com/me/packages")
	require.NoError(t, err)
	ssh, err := publish.DefaultForkPath("git@github.com:me/packages.git")
	require.NoError(t, err)

	assert.Equal(t, https, ssh)
}

// TestResolveForkPathKeepsConfiguredPath covers the user who chose where their
// fork clone lives: deriving a location would move it out from under them.
func TestResolveForkPathKeepsConfiguredPath(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	cfg := &config.Config{Fork: config.ForkConfig{Path: "/home/user/typst-packages"}}

	forkPath, err := publish.ResolveForkPath(testLogger(), cfg, "https://github.com/me/packages")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/typst-packages", forkPath)
}

// TestResolveForkPathMovesLegacyClone covers the upgrade: the clone gotpm kept
// at $DATA_DIR/fork is of the fork the user is publishing to, so it is moved
// there rather than cloned again.
func TestResolveForkPathMovesLegacyClone(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	origin := setupOriginRepo(t, []string{pkgDir})
	legacy := legacyClone(t, origin)

	forkPath, err := publish.ResolveForkPath(testLogger(), &config.Config{}, cloneURL(origin))
	require.NoError(t, err)

	assert.NoDirExists(t, legacy, "the legacy clone is moved, not copied")
	assert.DirExists(t, filepath.Join(forkPath, ".git"))
}

// TestResolveForkPathLeavesForeignLegacyClone covers the user who already hit
// the bug: their legacy clone is of the fork they published to before, and
// moving it would put that fork's clone where this fork's clone belongs.
func TestResolveForkPathLeavesForeignLegacyClone(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	legacy := legacyClone(t, setupOriginRepo(t, []string{pkgDir}))

	forkPath, err := publish.ResolveForkPath(testLogger(), &config.Config{}, "https://github.com/acme/packages")
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(legacy, ".git"), "another fork's clone stays where it is")
	assert.NoDirExists(t, forkPath)
}

// TestEnsureForkRepoRejectsForeignClone covers what deriving the path cannot
// see: a clone at an explicitly configured fork.path taken from a different
// fork. Publishing into it would push the submission to the wrong owner
// without a word.
func TestEnsureForkRepoRejectsForeignClone(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	forkPath := cloneFork(t, setupOriginRepo(t, []string{pkgDir}))

	err := publish.EnsureForkRepo(testLogger(), "https://github.com/acme/packages", forkPath)

	require.ErrorIs(t, err, publish.ErrForkOriginMismatch)
}

// TestEnsureForkRepoAcceptsOwnClone guards the check against firing on a
// correct setup that spells the fork url differently in the config than the
// clone's origin does.
func TestEnsureForkRepoAcceptsOwnClone(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	testrepo.Isolate(t)
	origin := setupOriginRepo(t, []string{pkgDir})
	forkPath := cloneFork(t, origin)

	err := publish.EnsureForkRepo(testLogger(), cloneURL(origin)+".git", forkPath)

	require.NoError(t, err)
}

// legacyClone puts a clone of origin where gotpm kept the single fork clone,
// back before the location was derived from the fork url.
func legacyClone(t *testing.T, origin string) string {
	t.Helper()
	dataDir, err := paths.GotpmDataDir()
	require.NoError(t, err)
	legacy := filepath.Join(dataDir, "fork")
	require.NoError(t, paths.EnsureDir(filepath.Dir(legacy)))
	require.NoError(t, gitcli.Clone(cloneURL(origin), legacy))
	return legacy
}
