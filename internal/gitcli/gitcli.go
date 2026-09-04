// Package gitcli shells out to the git binary for operations go-git cannot
// perform: a blobless partial clone combined with a sparse checkout. go-git
// has no mechanism to lazily fetch objects it excluded via a server-side
// filter, so any Checkout/Reset that needs a filtered-out blob fails with
// "object not found" - real git's promisor-remote support fetches such blobs
// on demand instead. This is also what the typst team recommends for cloning
// github.com/typst/packages: a --filter=blob:none clone kept sparse to only
// the package(s) actually being worked on, out of 1000+ in the repo.
package gitcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// run executes git with args inside dir, returning an error wrapping git's
// own combined stdout+stderr on failure.
func run(dir string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...) //nolint: gosec
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out.String()))
	}
	return nil
}

// Clone performs a blobless partial, history-shallow, checkout-less clone of
// remote into dest. Callers must follow up with SparseCheckoutSet and a
// checkout to materialize the subset of the tree they need.
func Clone(remote, dest string) error {
	cmd := exec.CommandContext( //nolint: gosec
		context.Background(), "git", "clone", "--depth", "1", "--filter=blob:none", "--no-checkout", remote, dest,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w: %s", err, strings.TrimSpace(out.String()))
	}
	return nil
}

// HasMain reports whether dir's clone has a resolvable origin/main
// remote-tracking ref, i.e. whether its initial clone actually completed.
func HasMain(dir string) bool {
	return run(dir, "rev-parse", "--verify", "--quiet", "origin/main") == nil
}

// Fetch updates origin/main, and only origin/main: Clone's --depth implies
// --single-branch, leaving the clone's refspec as
// +refs/heads/main:refs/remotes/origin/main. Package branches are not covered
// by it and need FetchBranch. Fetching keeps the clone shallow.
func Fetch(dir string) error {
	return run(dir, "fetch", "origin")
}

// FetchBranch updates origin/<branch> via an explicit refspec
func FetchBranch(dir, branch string) error {
	return run(dir, "fetch", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch)
}

// MergeFFOnly fast-forwards the current branch to origin/<branch>, failing
// rather than merging when the two have diverged
func MergeFFOnly(dir, branch string) error {
	return run(dir, "merge", "--ff-only", "origin/"+branch)
}

// Diverged reports whether branch and origin/<branch> have both moved on
// independently, i.e. neither ref contains the other. A branch merely ahead of
// the fork - the ordinary result of a publish that has not been pushed - is
// not diverged.
func Diverged(dir, branch string) bool {
	remote := "origin/" + branch
	behind := run(dir, "merge-base", "--is-ancestor", branch, remote) == nil
	ahead := run(dir, "merge-base", "--is-ancestor", remote, branch) == nil
	return !behind && !ahead
}

// ResetHard moves the current branch to rev and matches the worktree to it,
// discarding commits and changes that are not on rev.
func ResetHard(dir, rev string) error {
	return run(dir, "reset", "--hard", rev)
}

// SetUpstream makes branch track origin/<branch>.
func SetUpstream(dir, branch string) error {
	if err := run(dir, "config", "branch."+branch+".remote", "origin"); err != nil {
		return err
	}
	return run(dir, "config", "branch."+branch+".merge", "refs/heads/"+branch)
}

// TracksOwnBranch reports whether branch tracks origin/<branch> rather than
// origin/main or nothing at all
func TracksOwnBranch(dir, branch string) bool {
	pattern := "^refs/heads/" + regexp.QuoteMeta(branch) + "$"
	return run(dir, "config", "--get", "branch."+branch+".merge", pattern) == nil
}

// BranchExists reports whether branch already exists locally in dir.
func BranchExists(dir, branch string) bool {
	return run(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch) == nil
}

// SparseCheckoutSet scopes dir's worktree to path in cone mode. It does not
// itself materialize any files - a following checkout does that.
func SparseCheckoutSet(dir, path string) error {
	return run(dir, "sparse-checkout", "set", "--cone", path)
}

// CheckoutNewBranch creates branch from base and checks it out, force
// discarding any conflicting worktree state.
func CheckoutNewBranch(dir, branch, base string) error {
	return run(dir, "checkout", "--force", "-B", branch, base)
}

// CheckoutBranch checks out an already-existing branch, force discarding any
// conflicting worktree state.
func CheckoutBranch(dir, branch string) error {
	return run(dir, "checkout", "--force", branch)
}

// Add stages path.
func Add(dir, path string) error {
	return run(dir, "add", path)
}

// Commit commits the staged changes with msg, using the caller's own git
// config as-is (signing included) rather than overriding it.
func Commit(dir, msg string) error {
	return run(dir, "commit", "-m", msg)
}

// Push pushes branch to origin, updating the remote ref of the same name.
func Push(dir, branch string) error {
	return run(dir, "push", "origin", branch+":"+branch)
}
