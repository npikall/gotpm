package fonts_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/fonts"
	"github.com/npikall/gotpm/internal/fonts/fonttest"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	regular = "OpenSans-Regular.ttf"
	italic  = "OpenSans-Italic.ttf"
)

func resolve(t *testing.T, srv *fonttest.Server, family string) lockfile.Font {
	t.Helper()
	pin, err := srv.Source().Resolve(context.Background(), family)
	require.NoError(t, err)
	return pin
}

func ensure(t *testing.T, srv *fonttest.Server, dir fonts.Dir, pin lockfile.Font, force bool) (fonts.Result, error) {
	t.Helper()
	return dir.Ensure(context.Background(), srv.Source(), pin, force)
}

func readFont(t *testing.T, dir fonts.Dir, family, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir.Path(family), file))
	require.NoError(t, err)
	return string(data)
}

// handInstalled puts a family directory in place that gotpm did not write.
func handInstalled(t *testing.T, dir fonts.Dir, family string) string {
	t.Helper()
	file := filepath.Join(dir.Path(family), "Mine.ttf")
	require.NoError(t, paths.EnsureDir(filepath.Dir(file)))
	require.NoError(t, paths.WriteFile(file, []byte("mine")))
	return file
}

func TestResolve_PinsTheNewestCommitOfTheFamily(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	newest := srv.Publish("ofl", "opensans", regular, italic)

	pin := resolve(t, srv, "Open Sans")

	assert.Equal(t, lockfile.Font{
		Family: "Open Sans",
		Source: fonts.GoogleSource,
		URL:    "github.com/google/fonts//ofl/opensans",
		Hash:   newest,
		Files:  []lockfile.FontFile{{Name: italic}, {Name: regular}},
	}, pin)
}

func TestResolve_FindsTheFamilyUnderAnyLicense(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("apache", "roboto", "Roboto.ttf")

	pin := resolve(t, srv, "Roboto")

	assert.Equal(t, "github.com/google/fonts//apache/roboto", pin.URL)
}

func TestResolve_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		family  string
		publish map[string]string
		wantErr error
	}{
		{"unknown family", "Nope", nil, fonts.ErrFontNotFound},
		{"name without letters", "  -- ", nil, fonts.ErrInvalidName},
		{"no files listed", "Empty", map[string]string{}, fonts.ErrNoFontFiles},
		{"file escaping its directory", "Evil", map[string]string{"../evil.ttf": "x"}, fonts.ErrInvalidFileRef},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := fonttest.New(t)
			if tt.publish != nil {
				srv.PublishContents("ofl", fonts.Key(tt.family), tt.publish)
			}
			_, err := srv.Source().Resolve(context.Background(), tt.family)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEnsure_InstallsThePinnedCommitAndRecordsDigests(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.PublishContents("ofl", "opensans", map[string]string{regular: "v1"})
	pin := resolve(t, srv, "Open Sans")
	srv.PublishContents("ofl", "opensans", map[string]string{regular: "v2"})
	dir := fonts.Dir{Root: t.TempDir()}

	result, err := ensure(t, srv, dir, pin, false)

	require.NoError(t, err)
	assert.Equal(t, fonts.Installed, result.Outcome)
	assert.Equal(t, "v1", readFont(t, dir, "Open Sans", regular), "a newer upstream commit must not leak in")
	// sha256("v1")
	assert.Equal(t, "3bfc269594ef649228e9a74bab00f042efc91d5acc6fbee31a382e80d42388fe", result.Pin.Files[0].SHA256)
	prov, owned, err := dir.ReadProvenance("Open Sans")
	require.NoError(t, err)
	assert.True(t, owned)
	assert.Equal(t, fonts.Provenance{Source: fonts.GoogleSource, URL: pin.URL, Hash: pin.Hash}, prov)
}

func TestEnsure_RejectsADigestMismatchAndKeepsTheInstalledFamily(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	dir := fonts.Dir{Root: t.TempDir()}
	first, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)
	require.NoError(t, err)
	srv.Publish("ofl", "opensans", regular)
	tampered := resolve(t, srv, "Open Sans")
	tampered.Files = []lockfile.FontFile{{Name: regular, SHA256: "not-the-digest"}}

	_, err = ensure(t, srv, dir, tampered, false)

	require.ErrorIs(t, err, fonts.ErrChecksum)
	prov, _, err := dir.ReadProvenance("Open Sans")
	require.NoError(t, err)
	assert.Equal(t, first.Pin.Hash, prov.Hash)
}

func TestEnsure_LeavesTheInstalledCommitAlone(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	dir := fonts.Dir{Root: t.TempDir()}
	first, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)
	require.NoError(t, err)
	downloads := srv.Downloads()

	result, err := ensure(t, srv, dir, first.Pin, false)

	require.NoError(t, err)
	assert.Equal(t, fonts.UpToDate, result.Outcome)
	assert.Equal(t, downloads, srv.Downloads())
}

func TestEnsure_ReplacesAnotherCommitAndSaysWhich(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	old := srv.Publish("ofl", "opensans", "OpenSans-Old.ttf")
	dir := fonts.Dir{Root: t.TempDir()}
	_, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)
	require.NoError(t, err)
	srv.Publish("ofl", "opensans", regular)

	result, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)

	require.NoError(t, err)
	assert.Equal(t, fonts.Replaced, result.Outcome)
	assert.Equal(t, old, result.ReplacedSource.Hash)
	assert.NoFileExists(t, filepath.Join(dir.Path("Open Sans"), "OpenSans-Old.ttf"), "no stale file survives")
	assert.FileExists(t, filepath.Join(dir.Path("Open Sans"), regular))
}

func TestEnsure_RefusesAFamilyGotpmDidNotInstall(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	dir := fonts.Dir{Root: t.TempDir()}
	mine := handInstalled(t, dir, "opensans")

	_, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)

	require.ErrorIs(t, err, fonts.ErrNotOwned)
	assert.FileExists(t, mine)
}

func TestEnsure_ForceReplacesAFamilyGotpmDidNotInstall(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", regular)
	dir := fonts.Dir{Root: t.TempDir()}
	mine := handInstalled(t, dir, "opensans")

	result, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), true)

	require.NoError(t, err)
	assert.Equal(t, fonts.Replaced, result.Outcome)
	assert.NoFileExists(t, mine)
}

func TestEnsure_ReportsAFamilyShippingOnlyVariableFonts(t *testing.T) {
	t.Parallel()
	srv := fonttest.New(t)
	srv.Publish("ofl", "opensans", "OpenSans[wdth,wght].ttf", "OpenSans-Italic[wdth,wght].ttf")
	srv.Publish("ofl", "lato", "Lato-Regular.ttf", "Lato-Bold.ttf")
	dir := fonts.Dir{Root: t.TempDir()}

	variable, err := ensure(t, srv, dir, resolve(t, srv, "Open Sans"), false)
	require.NoError(t, err)
	static, err := ensure(t, srv, dir, resolve(t, srv, "Lato"), false)
	require.NoError(t, err)

	assert.True(t, variable.VariableOnly)
	assert.False(t, static.VariableOnly)
	assert.FileExists(t, filepath.Join(dir.Path("Open Sans"), "OpenSans[wdth,wght].ttf"))
}
