package manifest

import (
	"errors"
	"fmt"
	"os"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	tomledit "github.com/npikall/toml-edit"
	"github.com/npikall/toml-edit/eval"
)

// Namespace is the only namespace gotpm installs into and therefore the only
// one a dependency may be written with.
const Namespace = "gotpm"

var ErrInvalidDependency = errors.New("invalid dependency")

const (
	toolTable    = "tool"
	gotpmTable   = "gotpm"
	dependencies = "dependencies"
	fonts        = "fonts"
)

// Dependencies returns the packages the manifest declares, in the order they
// are written.
func (m *Manifest) Dependencies() []string {
	return m.Tool.Gotpm.Dependencies
}

// Fonts returns the font families the manifest declares, in the order they
// are written.
func (m *Manifest) Fonts() []string {
	return m.Tool.Gotpm.Fonts
}

// ParseDependencies turns the declared dependency strings into package
// references, rejecting anything gotpm cannot install. It is deliberately not
// part of loading a manifest: unrelated commands must not fail on this section.
func ParseDependencies(deps []string) ([]pkg.Ref, error) {
	refs := make([]pkg.Ref, 0, len(deps))
	var errs []error
	for _, dep := range deps {
		ref, err := pkg.ParseImport(dep)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w %q: %w", ErrInvalidDependency, dep, err))
			continue
		}
		if ref.Namespace != Namespace {
			errs = append(errs, fmt.Errorf("%w %q: namespace must be %q, gotpm cannot resolve %q",
				ErrInvalidDependency, dep, Namespace, ref.Namespace))
			continue
		}
		refs = append(refs, ref)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return refs, nil
}

// SetDependencies rewrites the dependency array of [tool.gotpm] in place:
// typst.toml is written by hand, so the rest of it stays as the author wrote it.
// An empty deps removes the array with the comment above it, and the section
// with it when nothing else is left in it.
func SetDependencies(file string, deps []string) error {
	return setArray(file, dependencies, deps)
}

// SetFonts rewrites the font array of [tool.gotpm] the way SetDependencies
// rewrites the dependency array.
func SetFonts(file string, families []string) error {
	return setArray(file, fonts, families)
}

func setArray(file, key string, values []string) error {
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not read %q: %w", file, err)
	}

	doc, err := tomledit.Parse(string(content))
	if err != nil {
		return fmt.Errorf("%q: %w: %w", file, ErrInvalidManifest, err)
	}
	if err := setGotpmArray(doc, key, values); err != nil {
		return fmt.Errorf("%q: %w: %w", file, ErrInvalidManifest, err)
	}
	return paths.WriteFile(file, []byte(doc.String()))
}

// setGotpmArray writes values to the key array of [tool.gotpm]. A new array is
// written one value per line, so adding a value later is a one-line diff —
// except inside an inline table, where a line break is TOML 1.1 only.
func setGotpmArray(doc *tomledit.Document, key string, values []string) error {
	path := []string{toolTable, gotpmTable, key}
	_, exists := doc.Get(path...)
	switch {
	case len(values) == 0 && !exists:
		return nil
	case len(values) == 0:
		if err := doc.Delete(path...); err != nil {
			return err //nolint: wrapcheck
		}
		return deleteIfEmpty(doc, toolTable, gotpmTable)
	case exists:
		return doc.Set(path, values) //nolint: wrapcheck
	case inInlineTable(doc, toolTable, gotpmTable):
		return doc.Insert(path, values) //nolint: wrapcheck
	default:
		return doc.Insert(path, values, tomledit.Multiline()) //nolint: wrapcheck
	}
}

// inInlineTable reports whether a key below keys would be written inside an
// inline table.
func inInlineTable(doc *tomledit.Document, keys ...string) bool {
	for i := range keys {
		v, _ := doc.Get(keys[:i+1]...)
		if table, ok := v.(*eval.Table); ok && table.Inline() != nil {
			return true
		}
	}
	return false
}

// deleteIfEmpty removes the table at keys if it holds nothing.
func deleteIfEmpty(doc *tomledit.Document, keys ...string) error {
	v, _ := doc.Get(keys...)
	if table, ok := v.(*eval.Table); !ok || len(table.Keys()) > 0 {
		return nil
	}
	return doc.Delete(keys...) //nolint: wrapcheck
}
