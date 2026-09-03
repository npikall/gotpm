package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func provenance() store.Provenance {
	return store.Provenance{
		URL:      "github.com/example/my-pkg",
		Revision: "v0.1.0",
		Hash:     "abc123def456",
	}
}

func TestProvenance_RoundTrips(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))

	require.NoError(t, s.WriteProvenance(r, provenance()))

	got, found, err := s.ReadProvenance(r)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, provenance(), got)
}

func TestProvenance_LivesInsideTheInstalledPackage(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))
	require.NoError(t, s.WriteProvenance(r, provenance()))

	assert.Equal(t, filepath.Join(s.Dir(r), store.ProvenanceFile), s.ProvenancePath(r),
		"provenance must travel with the package so it disappears when the package does")
	assert.FileExists(t, s.ProvenancePath(r))
}

func TestReadProvenance_AbsentIsNotAnError(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))

	_, found, err := s.ReadProvenance(r)
	require.NoError(t, err, "a package placed by hand is an unknown origin, not a failure")
	assert.False(t, found)
}

func TestReadProvenance_NotInstalledIsNotAnError(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())

	_, found, err := s.ReadProvenance(ref(t))
	require.NoError(t, err)
	assert.False(t, found)
}

func TestReadProvenance_RejectsAMalformedFile(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))
	require.NoError(t, paths.WriteFile(s.ProvenancePath(r), []byte("not json")))

	_, _, err := s.ReadProvenance(r)
	require.ErrorIs(t, err, store.ErrInvalidProvenance)
}

func TestInstall_DropsAProvenanceFileFoundInTheSource(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	src := sourcePackage(t, "v1")
	// The source is itself a former gotpm-managed install (e.g. a checked-out
	// fork), carrying a record of a repository and commit its files no longer
	// match.
	require.NoError(t, paths.WriteFile(filepath.Join(src, store.ProvenanceFile), []byte(`{"url":"github.com/other/pkg"}`)))

	require.NoError(t, s.Install(r, src))

	_, found, err := s.ReadProvenance(r)
	require.NoError(t, err)
	assert.False(t, found, "a provenance file copied in from the source must not survive as the destination's own record")
}

func TestReadProvenance_IgnoresAnEditableInstall(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	src := sourcePackage(t, "v1")
	// The linked working tree happens to contain a file with the right name,
	// but it names the source's own history, not how this ref got here.
	require.NoError(t, paths.WriteFile(filepath.Join(src, store.ProvenanceFile), []byte(`{"url":"github.com/other/pkg"}`)))
	require.NoError(t, s.Link(r, src))

	_, found, err := s.ReadProvenance(r)
	require.NoError(t, err)
	assert.False(t, found, "an editable install has no provenance by construction, whatever the linked working tree contains")
}

func TestProvenance_IsNotListedAsAVersion(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))
	require.NoError(t, s.WriteProvenance(r, provenance()))

	namespaces, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	require.Len(t, namespaces[0].Packages, 1)
	assert.Equal(t, []store.Version{{Name: "0.1.0"}}, namespaces[0].Packages[0].Versions)
}

func TestInstall_IsAtomic(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)

	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))

	entries, err := os.ReadDir(filepath.Dir(s.Dir(r)))
	require.NoError(t, err)
	require.Len(t, entries, 1, "the staging directory must not survive a successful install")
	assert.Equal(t, "0.1.0", entries[0].Name())

	info, err := os.Stat(s.Dir(r))
	require.NoError(t, err)
	assert.Equal(t, paths.DirPerm, info.Mode().Perm(),
		"an installed package must be readable, not left with the staging directory's permissions")
}

func TestScan_IgnoresLeftoverStagingDirectories(t *testing.T) {
	t.Parallel()
	s := store.At(t.TempDir())
	r := ref(t)
	require.NoError(t, s.Install(r, sourcePackage(t, "v1")))

	// What an install interrupted between staging and rename leaves behind.
	leftover := filepath.Join(s.PackageDir("local", "my-pkg"), ".gotpm-staging-123")
	require.NoError(t, paths.EnsureDir(leftover))
	require.NoError(t, paths.WriteFile(filepath.Join(leftover, "lib.typ"), []byte("half")))

	namespaces, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	require.Len(t, namespaces[0].Packages, 1)
	assert.Equal(t, []store.Version{{Name: "0.1.0"}}, namespaces[0].Packages[0].Versions,
		"a half-written install must never be reported as a usable version")
}
