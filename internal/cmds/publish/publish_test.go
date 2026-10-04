package publish_test

import (
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/publish"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolatePublish gives a publish its own data and config directories, a git
// identity to commit with, and a fork to publish to. It returns the fork's
// origin repository.
func isolatePublish(t *testing.T) string {
	t.Helper()
	testrepo.Isolate(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, key := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(key+"_NAME", "test")
		t.Setenv(key+"_EMAIL", "test@test.com")
	}
	origin := setupOriginRepo(t, []string{"packages/preview/other"})
	require.NoError(t, config.Save(&config.Config{Fork: config.ForkConfig{URL: cloneURL(origin)}}))
	return origin
}

// forkClone returns where publish keeps the clone of the fork at origin.
func forkClone(t *testing.T, origin string) string {
	t.Helper()
	forkPath, err := publish.DefaultForkPath(cloneURL(origin))
	require.NoError(t, err)
	return forkPath
}

func writePackageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, paths.WriteFile(filepath.Join(dir, name), []byte(content)))
}

// TestRunLocalCommitsPackageToFork covers a first publish with --local: the
// package lands in its own version directory on a fresh branch of the fork
// clone, and nothing is pushed.
func TestRunLocalCommitsPackageToFork(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	origin := isolatePublish(t)
	src := testrepo.Project(t, "foo")
	writePackageFile(t, src, "main.typ", "= foo")

	require.NoError(t, publish.Run(&publish.Options{Local: true}, testLogger()))

	forkPath := forkClone(t, origin)
	assert.FileExists(t, filepath.Join(forkPath, "packages", "preview", "foo", "0.1.0", "main.typ"))
	assert.Equal(t, "foo-0.1.0", gitOut(t, forkPath, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, "release: foo 0.1.0", gitOut(t, forkPath, "log", "-1", "--format=%s"))
	assert.Empty(t, gitOut(t, origin, "branch", "--list", "foo-0.1.0"), "--local does not push")
}

// TestRunPushesBranchToFork covers a full publish: the package branch ends up
// on the fork, ready for a pull request.
func TestRunPushesBranchToFork(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	origin := isolatePublish(t)
	testrepo.Project(t, "foo")

	require.NoError(t, publish.Run(&publish.Options{Message: "custom"}, testLogger()))

	assert.Equal(t, "custom", gitOut(t, origin, "log", "-1", "--format=%s", "foo-0.1.0"))
}

// TestRunRepublishReusesSourceCommitMessage covers publishing a version again
// after a fix: the fork gets the source repository's latest commit message
// rather than a second, indistinguishable release message.
func TestRunRepublishReusesSourceCommitMessage(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	origin := isolatePublish(t)
	src := testrepo.Project(t, "foo")
	require.NoError(t, publish.Run(&publish.Options{Local: true}, testLogger()))

	writePackageFile(t, src, "main.typ", "= fixed")
	runGit(t, src, "init", "-q")
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-q", "-m", "fix: heading")
	require.NoError(t, publish.Run(&publish.Options{Local: true}, testLogger()))

	assert.Equal(t, "fix: heading", gitOut(t, forkClone(t, origin), "log", "-1", "--format=%s"))
}

// TestRunRepublishWithoutSourceRepoFallsBack covers a republish from a
// package that is not under git: there is no commit message to reuse.
func TestRunRepublishWithoutSourceRepoFallsBack(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	origin := isolatePublish(t)
	src := testrepo.Project(t, "foo")
	require.NoError(t, publish.Run(&publish.Options{Local: true}, testLogger()))

	writePackageFile(t, src, "main.typ", "= changed")
	require.NoError(t, publish.Run(&publish.Options{Local: true}, testLogger()))

	assert.Equal(t, "release: foo 0.1.0", gitOut(t, forkClone(t, origin), "log", "-1", "--format=%s"))
}

func TestRunWithoutForkURL(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	isolatePublish(t)
	require.NoError(t, config.Save(&config.Config{}))
	testrepo.Project(t, "foo")

	err := publish.Run(&publish.Options{Local: true}, testLogger())

	require.ErrorIs(t, err, publish.ErrMissingForkURL)
}

func TestRunOutsideAPackage(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	isolatePublish(t)
	t.Chdir(t.TempDir())

	err := publish.Run(&publish.Options{Local: true}, testLogger())

	require.ErrorContains(t, err, "could not load manifest")
}

func TestRunWithCustomMessage(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	origin := isolatePublish(t)
	src := testrepo.Project(t, "foo")
	writePackageFile(t, src, "lib.typ", "#let foo = 1")

	require.NoError(t, publish.Run(&publish.Options{Local: true, Message: "custom"}, testLogger()))

	assert.Equal(t, "custom", gitOut(t, forkClone(t, origin), "log", "-1", "--format=%s"))
}

func TestRunWithBrokenConfig(t *testing.T) { //nolint: paralleltest // Isolate uses t.Setenv
	isolatePublish(t)
	configPath, err := config.Path()
	require.NoError(t, err)
	require.NoError(t, paths.WriteFile(configPath, []byte("not = [toml")))
	testrepo.Project(t, "foo")

	require.Error(t, publish.Run(&publish.Options{Local: true}, testLogger()))
}
