package lockfile_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// font builds a font pin with plausible provenance.
func font(family, hash string, direct bool, requiredBy ...string) lockfile.Font {
	return lockfile.Font{
		Family:     family,
		Source:     "google-fonts",
		URL:        "github.com/google/fonts//ofl/" + family,
		Hash:       hash,
		Files:      []lockfile.FontFile{{Name: family + ".ttf", SHA256: "sum-" + hash}},
		Direct:     direct,
		RequiredBy: requiredBy,
	}
}

func families(fonts []lockfile.Font) []string {
	out := make([]string, 0, len(fonts))
	for _, f := range fonts {
		out = append(out, f.Family)
	}
	return out
}

func savedVersion(t *testing.T, dir string) int {
	t.Helper()
	data, err := os.ReadFile(lockfile.Path(dir))
	require.NoError(t, err)
	var raw struct {
		Version int `json:"version"`
	}
	require.NoError(t, json.Unmarshal(data, &raw))
	return raw.Version
}

func TestSave_WithoutFontsStaysVersionOne(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/cetz:0.3.1", true))

	require.NoError(t, lockfile.Save(dir, lock))

	assert.Equal(t, 1, savedVersion(t, dir), "older gotpm must still read a lock without font pins")
}

func TestSave_WithFontsIsVersionTwoAndRoundTrips(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lock := lockfile.New()
	lock.UpsertFont(font("roboto", "abc", true))

	require.NoError(t, lockfile.Save(dir, lock))
	loaded, err := lockfile.Load(dir)

	require.NoError(t, err)
	assert.Equal(t, 2, savedVersion(t, dir))
	assert.Equal(t, []lockfile.Font{font("roboto", "abc", true)}, loaded.Fonts)
}

func TestGetFont_MatchesTheFamilyLooselyWritten(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.UpsertFont(font("Open Sans", "abc", true))

	got, ok := lock.GetFont("opensans")

	require.True(t, ok)
	assert.Equal(t, "Open Sans", got.Family)
}

func TestUpsertFont_DirectPinReplacesATransitiveOne(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.UpsertFont(font("roboto", "old", false, "@gotpm/thesis:1.0.0"))

	_, conflict := lock.UpsertFont(font("roboto", "new", true))

	require.False(t, conflict, "the project's own pin settles it")
	got, _ := lock.GetFont("roboto")
	assert.Equal(t, "new", got.Hash)
	assert.True(t, got.Direct)
	assert.Equal(t, []string{"@gotpm/thesis:1.0.0"}, got.RequiredBy)
}

func TestUpsertFont_TransitivePinNeverOverridesTheExistingOne(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.UpsertFont(font("roboto", "mine", true))

	kept, conflict := lock.UpsertFont(font("roboto", "theirs", false, "@gotpm/thesis:1.0.0"))

	assert.True(t, conflict)
	assert.Equal(t, "mine", kept.Hash)
	got, _ := lock.GetFont("roboto")
	assert.Equal(t, "mine", got.Hash)
	assert.Equal(t, []string{"@gotpm/thesis:1.0.0"}, got.RequiredBy, "the dependant is still recorded")
}

func TestUpsertFont_SameCommitIsNoConflict(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.UpsertFont(font("roboto", "abc", false, "@gotpm/a:1.0.0"))

	_, conflict := lock.UpsertFont(font("roboto", "abc", false, "@gotpm/b:1.0.0"))

	assert.False(t, conflict)
	got, _ := lock.GetFont("roboto")
	assert.Equal(t, []string{"@gotpm/a:1.0.0", "@gotpm/b:1.0.0"}, got.RequiredBy)
}

func TestPruneFonts_DropsFontsNothingReaches(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/thesis:1.0.0", true))
	lock.UpsertFont(font("roboto", "a", true))
	lock.UpsertFont(font("lato", "b", false, "@gotpm/thesis:1.0.0"))
	lock.UpsertFont(font("inter", "c", false, "@gotpm/gone:1.0.0"))

	removed := lock.PruneFonts([]string{"Roboto"})

	assert.Equal(t, []string{"inter"}, families(removed))
	assert.Equal(t, []string{"lato", "roboto"}, families(lock.Fonts))
}

func TestPruneFonts_RemarksDirectAndDropsGoneDependants(t *testing.T) {
	t.Parallel()
	lock := lockfile.New()
	lock.Upsert(entry("@gotpm/thesis:1.0.0", true))
	lock.UpsertFont(font("lato", "b", true, "@gotpm/thesis:1.0.0", "@gotpm/gone:1.0.0"))

	removed := lock.PruneFonts(nil)

	assert.Empty(t, removed, "a dependency still reaches it")
	got, _ := lock.GetFont("lato")
	assert.False(t, got.Direct)
	assert.Equal(t, []string{"@gotpm/thesis:1.0.0"}, got.RequiredBy)
}
