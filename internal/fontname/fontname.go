// Package fontname compares font family names the way the Google Fonts
// repository lays out its directories, so "Open Sans", "open sans" and
// "opensans" name the same font family.
package fontname

import (
	"strings"
	"unicode"
)

// Key is the comparable form of a font family name: lowercased, with everything
// but ASCII letters and digits dropped. It is empty for a name holding neither.
func Key(name string) string {
	return strings.Map(keep, strings.ToLower(name))
}

// keep maps a rune to itself when a key keeps it, and drops it otherwise.
func keep(r rune) rune {
	if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
		return r
	}
	return -1
}
