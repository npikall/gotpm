package gitcli_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/npikall/gotpm/internal/gitcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain detaches every git invocation in this package from the developer's
// own git configuration, which could otherwise sign commits, rename the
// initial branch or fail for want of an identity.
func TestMain(m *testing.M) {
	os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	os.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	os.Exit(m.Run())
}

// TestDiverged covers the predicate that decides whether publish may reset a
// package branch onto the fork. Only the last case may reset: a branch behind
// the fork fast-forwards, and one merely ahead holds an unpushed commit.
func TestDiverged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		localExtraCommits int
		forkExtraCommits  int
		want              bool
	}{
		{name: "identical", want: false},
		{name: "behind the fork", forkExtraCommits: 1, want: false},
		{name: "ahead of the fork", localExtraCommits: 1, want: false},
		{name: "diverged", localExtraCommits: 1, forkExtraCommits: 1, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := repoWithBranchAndRemoteRef(t, "foo-0.1.0", tt.localExtraCommits, tt.forkExtraCommits)
			assert.Equal(t, tt.want, gitcli.Diverged(dir, "foo-0.1.0"))
		})
	}
}

// repoWithBranchAndRemoteRef builds a repo holding branch and a stand-in
// refs/remotes/origin/<branch>, forked from a common commit and then advanced
// by the given number of commits on each side.
func repoWithBranchAndRemoteRef(t *testing.T, branch string, local, fork int) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "initial commit")

	runGit(t, dir, "checkout", "-q", "-b", "fork-side")
	for i := range fork {
		runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "fork commit "+string(rune('a'+i)))
	}
	runGit(t, dir, "update-ref", "refs/remotes/origin/"+branch, "fork-side")

	runGit(t, dir, "checkout", "-q", "-B", branch, "main")
	for i := range local {
		runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "local commit "+string(rune('a'+i)))
	}
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
}
