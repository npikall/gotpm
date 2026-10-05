package depgraph_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/depgraph"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/resolve"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fontPin(family, hash string) lockfile.Font {
	return lockfile.Font{
		Family: family, Source: "google-fonts", URL: "github.com/google/fonts//ofl/" + family, Hash: hash,
		Files: []lockfile.FontFile{{Name: family + ".ttf", SHA256: "sum"}},
	}
}

func walkFonts(t *testing.T, root *testrepo.Package) depgraph.Result {
	t.Helper()
	result, err := depgraph.Walk(resolve.Request{URL: root.URL()}, depgraph.Options{}, discardLogger())
	require.NoError(t, err)
	return result
}

func TestWalk_CollectsTheFontsEveryPackageDeclares(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").ReleaseFonts(fontPin("lato", "a"))
	lock := lockfile.New()
	lock.Upsert(leaf.LockEntry())
	lock.UpsertFont(fontPin("inter", "b"))
	root := testrepo.New(t, "root", "1.0.0").ReleaseDeclaring([]string{leaf.Import()}, []string{"Inter"}, lock)

	result := walkFonts(t, root)

	require.Len(t, result.Fonts, 2)
	assert.Equal(t, "inter", result.Fonts[0].Family)
	assert.Equal(t, []string{root.Import()}, result.Fonts[0].RequiredBy)
	assert.False(t, result.Fonts[0].Direct, "a dependency's font is transitive for the project")
	assert.Equal(t, []string{leaf.Import()}, result.Fonts[1].RequiredBy)
}

func TestWalk_KeepsTheFirstPinOfAFamilyPinnedTwice(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	leaf := testrepo.New(t, "leaf", "1.0.0").ReleaseFonts(fontPin("lato", "leafs"))
	lock := lockfile.New()
	lock.Upsert(leaf.LockEntry())
	lock.UpsertFont(fontPin("lato", "roots"))
	root := testrepo.New(t, "root", "1.0.0").ReleaseDeclaring([]string{leaf.Import()}, []string{"lato"}, lock)

	result := walkFonts(t, root)

	require.Len(t, result.Fonts, 1)
	assert.Equal(t, "roots", result.Fonts[0].Hash)
	assert.Equal(t, []string{leaf.Import(), root.Import()}, result.Fonts[0].RequiredBy)
}

func TestWalk_SkipsAFontDeclaredWithoutAPin(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	root := testrepo.New(t, "root", "1.0.0").ReleaseDeclaring(nil, []string{"Lato"}, lockfile.New())

	assert.Empty(t, walkFonts(t, root).Fonts)
}
