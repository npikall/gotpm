package typstsrc

import (
	"fmt"
	"regexp"

	"github.com/npikall/gotpm/internal/pkg"
)

// refPattern matches a reference to a Typst Universe package as written in an
// import statement, e.g. "@preview/cetz:0.5.2".
var refPattern = regexp.MustCompile(`@preview/[a-zA-Z0-9_-]+:\d+\.\d+\.\d+`)

// FindRefs returns every Typst Universe package referenced in content, in the
// order they appear and without duplicates.
func FindRefs(content []byte) []pkg.Ref {
	var refs []pkg.Ref
	seen := make(map[string]struct{})
	for _, match := range refPattern.FindAll(content, -1) {
		statement := string(match)
		if _, dup := seen[statement]; dup {
			continue
		}
		seen[statement] = struct{}{}

		ref, err := pkg.ParseImport(statement)
		if err != nil {
			// Unreachable for anything refPattern matches, but a reference
			// that cannot be parsed is one this package has no business
			// rewriting either.
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

// RewriteRefs sets the version of every referenced package for which latest
// holds an entry, keyed by package name.
func RewriteRefs(content []byte, latest map[string]string) []byte {
	for name, version := range latest {
		pattern := regexp.MustCompile(fmt.Sprintf(`@preview/%s:\d+\.\d+\.\d+`, regexp.QuoteMeta(name)))
		content = pattern.ReplaceAll(content, fmt.Appendf(nil, "@preview/%s:%s", name, version))
	}
	return content
}
