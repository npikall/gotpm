package git

import (
	"errors"
	"fmt"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

var (
	// ErrNoIdentity reports a git installation with no user.name or
	// user.email, which git itself also refuses to commit without.
	ErrNoIdentity = errors.New("no git identity configured")

	// ErrNoFiles reports a package with no publishable files.
	ErrNoFiles = errors.New("package has no files to publish")

	// ErrNothingToCommit reports a publish whose files are byte for byte what
	// the fork already holds, which git would also refuse as an empty commit.
	ErrNothingToCommit = errors.New("package is unchanged from the fork")
)

// Publication is one package directory to commit into the fork.
type Publication struct {
	// Branch is the package branch to commit on.
	Branch string
	// Dir is the slash-separated directory in the fork the files go into,
	// replacing whatever it held before.
	Dir string
	// Files are the package's files, relative to Dir.
	Files []File
	// Message is the commit message.
	Message string
}

// Commit writes pub into the fork on top of base and moves the branch to the
// new commit. It returns the new commit's hash.
//
// Nothing is checked out: the files are stored as blobs, gathered into a tree
// for Dir, and spliced into a copy of base's tree. Only the trees on the path
// down to Dir are read, so the fork's other packages are never fetched.
func (f *Fork) Commit(pub Publication, base Base) (plumbing.Hash, error) {
	if len(pub.Files) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("%w: %s", ErrNoFiles, pub.Dir)
	}

	author, err := f.identity()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	parent, err := f.repo.CommitObject(base.Commit)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("reading base commit %s: %w", base.Commit, err)
	}

	dirTree, err := writeDir(f.repo.Storer, pub.Files)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	root, err := splice(f.repo.Storer, parent.TreeHash, splitPath(pub.Dir), dirTree)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if root == parent.TreeHash {
		return plumbing.ZeroHash, fmt.Errorf("%w: %s", ErrNothingToCommit, pub.Dir)
	}

	commit := &object.Commit{
		Author:       author,
		Committer:    author,
		Message:      ensureTrailingNewline(pub.Message),
		TreeHash:     root,
		ParentHashes: []plumbing.Hash{parent.Hash},
	}
	obj := f.repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding commit: %w", err)
	}
	hash, err := f.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing commit: %w", err)
	}

	if err := f.SetBranch(pub.Branch, hash); err != nil {
		return plumbing.ZeroHash, err
	}
	return hash, nil
}

// identity returns the committer to record, read from the user's git
// configuration the same way `git commit` reads it.
//
// Signing is deliberately not attempted. go-git ignores commit.gpgsign and
// user.signingkey and can only sign from a key handed to it in process, so a
// signed commit here would need gotpm to reimplement git's key discovery. An
// unsigned commit can be amended with `git commit --amend -S` in the fork.
func (f *Fork) identity() (object.Signature, error) {
	cfg, err := f.repo.ConfigScoped(config.SystemScope)
	if err != nil {
		return object.Signature{}, fmt.Errorf("reading git configuration: %w", err)
	}
	if cfg.User.Name == "" || cfg.User.Email == "" {
		return object.Signature{}, fmt.Errorf(
			"%w\nRun: git config --global user.name <name> && git config --global user.email <email>",
			ErrNoIdentity,
		)
	}
	return object.Signature{
		Name:  cfg.User.Name,
		Email: cfg.User.Email,
		When:  time.Now(),
	}, nil
}

// Push sends branch to the fork, updating the remote branch of the same name.
func (f *Fork) Push(branch string) error {
	err := f.repo.Push(&gogit.PushOptions{
		RemoteName: RemoteName,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", localRef(branch), localRef(branch)))},
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pushing %q to %s: %w", branch, RemoteName, err)
	}
	return nil
}

func ensureTrailingNewline(msg string) string {
	if msg == "" || msg[len(msg)-1] == '\n' {
		return msg
	}
	return msg + "\n"
}
