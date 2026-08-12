// This test lives inside the package rather than in git_test because it
// inspects the object database directly - the shallow boundary and the exact
// set of objects a push would pack - which is the whole point of it.

package git //nolint: testpackage

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/format/packfile"
	"github.com/go-git/go-git/v6/plumbing/revlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveTypstPackages runs the real thing against github.com/typst/packages:
// a blobless shallow clone, a package committed onto it, and the exact object
// selection a push would perform.
//
// The file:// tests cannot cover any of this. go-git's own server implements
// neither partial clone nor shallow fetch, so those clones always degrade to a
// full one - and a full clone hides the property that matters here, which is
// that a commit can be built and packed while the fork's other 1500 packages
// are absent from the object database.
func TestLiveTypstPackages(t *testing.T) {
	t.Parallel()
	if os.Getenv("GOTPM_LIVE") == "" {
		t.Skip("set GOTPM_LIVE=1 to run against github.com")
	}

	start := time.Now()
	fork, err := Clone("https://github.com/typst/packages.git", filepath.Join(t.TempDir(), "fork"))
	require.NoError(t, err)
	t.Logf("blobless shallow clone took %s", time.Since(start))
	require.True(t, fork.HasMain())

	shallow, err := fork.repo.Storer.Shallow()
	require.NoError(t, err)
	require.Len(t, shallow, 1, "the clone kept its depth-1 boundary")

	base, err := fork.ResolveBase("gotpm-probe-0.1.0")
	require.NoError(t, err)
	require.False(t, base.Existed)

	source := filepath.Join(t.TempDir(), "lib.typ")
	require.NoError(t, os.WriteFile(source, []byte("#let hello() = []\n"), 0o644))

	commit, err := fork.Commit(Publication{
		Branch:  "gotpm-probe-0.1.0",
		Dir:     "packages/preview/gotpm-probe/0.1.0",
		Files:   []File{{Path: "lib.typ", Source: source}},
		Message: "release: gotpm-probe 0.1.0",
	}, base)
	require.NoError(t, err)

	// What Remote.push would pack: wants is the new commit, haves are the
	// fork's refs plus the shallow boundary. Walking into any of the 1500
	// packages the clone never fetched would fail here.
	haves := append([]plumbing.Hash{base.Commit}, shallow...)
	hashes, err := revlist.Objects(fork.repo.Storer, []plumbing.Hash{commit}, haves)
	require.NoError(t, err)
	assert.Len(t, hashes, 7,
		"the commit, the five trees down to the version directory, and one blob")

	_, err = packfile.NewEncoder(io.Discard, fork.repo.Storer, false).Encode(hashes, 10)
	require.NoError(t, err)
}
