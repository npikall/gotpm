package lockfile

import (
	"slices"
	"strings"

	"github.com/npikall/gotpm/internal/fontname"
)

// FontFile is one file of a pinned font family, with the digest it was
// fetched with.
type FontFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Font pins one font family to the exact commit of its font source it was
// fetched from (ADR 0007).
type Font struct {
	// Family is the name the family is declared with, e.g. "Open Sans". It is
	// the primary key of a font pin, compared loosely through fontname.Key.
	Family string `json:"family"`
	// Source names the font source, e.g. "google-fonts".
	Source string `json:"source"`
	// URL locates the family inside its source, without a scheme, e.g.
	// "github.com/google/fonts//ofl/opensans".
	URL   string     `json:"url"`
	Hash  string     `json:"hash"`
	Files []FontFile `json:"files"`
	// Direct marks a font the project declares itself.
	Direct bool `json:"direct"`
	// RequiredBy holds the imports of the packages that declare this font.
	RequiredBy []string `json:"required_by"`
}

// Key is the comparable form of the family name.
func (f Font) Key() string {
	return fontname.Key(f.Family)
}

// GetFont returns the pin of a font family, matching its name loosely.
func (l *Lock) GetFont(family string) (Font, bool) {
	i := l.indexOfFont(fontname.Key(family))
	if i < 0 {
		return Font{}, false
	}
	return l.Fonts[i], true
}

// UpsertFont adds a font pin, or merges it into the one already recorded for
// the family. A direct pin replaces what is recorded; a transitive pin never
// does, and a conflict is reported when it names another commit. Direct and
// RequiredBy accumulate either way. It returns the pin now recorded.
func (l *Lock) UpsertFont(f Font) (Font, bool) {
	i := l.indexOfFont(f.Key())
	if i < 0 {
		f.RequiredBy = normalise(f.RequiredBy)
		l.Fonts = append(l.Fonts, f)
		l.sortFonts()
		return f, false
	}

	existing := l.Fonts[i]
	kept, conflict := f, false
	if !f.Direct {
		kept, conflict = existing, existing.Hash != f.Hash
	}
	kept.Direct = existing.Direct || f.Direct
	kept.RequiredBy = normalise(append(slices.Clone(existing.RequiredBy), f.RequiredBy...))
	l.Fonts[i] = kept
	return kept, conflict
}

// PruneFonts drops every font pin neither declared by the project nor required
// by a package still in the lock, and returns what it removed. Run it after
// Prune, so the packages it checks against are the ones that survived. It also
// re-marks Direct and forgets dependants that are gone.
func (l *Lock) PruneFonts(declared []string) []Font {
	direct := fontKeys(declared)
	kept := make([]Font, 0, len(l.Fonts))
	var removed []Font
	for _, f := range l.Fonts {
		f.Direct = direct[f.Key()]
		f.RequiredBy = slices.DeleteFunc(slices.Clone(f.RequiredBy), l.isGone)
		if f.Direct || len(f.RequiredBy) > 0 {
			kept = append(kept, f)
			continue
		}
		removed = append(removed, f)
	}
	l.Fonts = kept
	return removed
}

// isGone reports whether no package is locked under imp.
func (l *Lock) isGone(imp string) bool {
	return l.indexOf(imp) < 0
}

func fontKeys(families []string) map[string]bool {
	keys := make(map[string]bool, len(families))
	for _, family := range families {
		keys[fontname.Key(family)] = true
	}
	return keys
}

func (l *Lock) indexOfFont(key string) int {
	return slices.IndexFunc(l.Fonts, func(f Font) bool { return f.Key() == key })
}

func (l *Lock) sortFonts() {
	slices.SortFunc(l.Fonts, func(a, b Font) int { return strings.Compare(a.Key(), b.Key()) })
}
