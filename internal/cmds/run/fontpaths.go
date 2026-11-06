package run

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/npikall/gotpm/internal/fonts"
)

const fontPathsKey = "TYPST_FONT_PATHS"

// environment is gotpm's own environment with the font directory added to
// TYPST_FONT_PATHS after the user's own paths, unless they already name it.
func environment() ([]string, error) {
	dir, err := fonts.OpenDir()
	if err != nil {
		return nil, err
	}
	env := slices.DeleteFunc(os.Environ(), func(pair string) bool {
		return strings.HasPrefix(pair, fontPathsKey+"=")
	})
	return append(env, fontPathsKey+"="+withFontDir(os.Getenv(fontPathsKey), dir.Root)), nil
}

// withFontDir is the user's list of font paths with fontDir appended, unless
// one of them already is fontDir. Empty entries are dropped.
func withFontDir(userPaths, fontDir string) string {
	list := slices.DeleteFunc(filepath.SplitList(userPaths), func(path string) bool { return path == "" })
	if !slices.ContainsFunc(list, func(path string) bool { return filepath.Clean(path) == filepath.Clean(fontDir) }) {
		list = append(list, fontDir)
	}
	return strings.Join(list, string(os.PathListSeparator))
}
