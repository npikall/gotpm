package manifest

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
)

// Namespace is the only namespace gotpm installs into and therefore the only
// one a dependency may be written with.
const Namespace = "gotpm"

var (
	ErrInvalidDependency = errors.New("invalid dependency")
	ErrInlineToolSection = errors.New("'tool.gotpm' must be written as a '[tool.gotpm]' section")
)

const (
	toolTable    = "tool"
	gotpmTable   = "gotpm"
	dependencies = "dependencies"
	arrayIndent  = "  "
	// arrayDelimiters is the number of lines the opening and closing bracket
	// of a rendered array take up.
	arrayDelimiters = 2
)

// Dependencies returns the packages the manifest declares, in the order they
// are written.
func (m *Manifest) Dependencies() []string {
	return m.Tool.Gotpm.Dependencies
}

// ParseDependencies turns the declared dependency strings into package
// references, rejecting anything gotpm cannot install. It is deliberately not
// part of loading a manifest: an unrelated command must not fail because of a
// section only add, sync and remove care about.
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

// SetDependencies rewrites the dependency array of the [tool.gotpm] section of
// a manifest file, leaving every other byte of the document exactly as it was.
//
// It edits the file as text rather than decoding and re-encoding it, because
// the TOML encoder discards comments and reorders keys, and typst.toml is a
// file people write and comment by hand.
//
// An empty deps removes the array, and the section too when nothing but blank
// lines would be left in it.
func SetDependencies(file string, deps []string) error {
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not read %q: %w", file, err)
	}

	newline := "\n"
	if strings.Contains(string(content), "\r\n") {
		newline = "\r\n"
	}
	// A trailing newline splits into a final empty element. Dropping it here
	// and adding it back on join keeps the file's ending as it was.
	text := string(content)
	trailing := strings.HasSuffix(text, newline)
	lines := strings.Split(strings.TrimSuffix(text, newline), newline)

	updated, err := setDependencyLines(lines, deps)
	if err != nil {
		return fmt.Errorf("%q: %w", file, err)
	}

	out := strings.Join(updated, newline)
	if trailing || out != "" {
		out += newline
	}
	return paths.WriteFile(file, []byte(out))
}

// setDependencyLines performs the edit on the split document.
func setDependencyLines(lines, deps []string) ([]string, error) {
	header := findTable(lines, toolTable, gotpmTable)
	if header < 0 {
		if err := rejectInlineToolSection(lines); err != nil {
			return nil, err
		}
		if len(deps) == 0 {
			return lines, nil
		}
		return appendSection(lines, deps), nil
	}

	end := tableEnd(lines, header)
	start, stop, found := findArray(lines, header+1, end)

	switch {
	case !found && len(deps) == 0:
		return lines, nil
	case !found:
		// Insert as the first key of the existing section.
		return splice(lines, header+1, header+1, renderArray(deps)), nil
	case len(deps) == 0:
		return removeArray(lines, header, start, stop, end), nil
	default:
		return splice(lines, start, stop+1, renderArray(deps)), nil
	}
}

// rejectInlineToolSection refuses a manifest that writes the section in a form
// this editor cannot rewrite, e.g. a [tool] table holding an inline
// "gotpm = { ... }". Silently appending a second [tool.gotpm] would produce a
// document TOML rejects as a duplicate key.
func rejectInlineToolSection(lines []string) error {
	tool := findTable(lines, toolTable)
	if tool < 0 {
		return nil
	}
	for _, line := range lines[tool+1 : tableEnd(lines, tool)] {
		if key, _, ok := strings.Cut(line, "="); ok && strings.TrimSpace(key) == gotpmTable {
			return ErrInlineToolSection
		}
	}
	return nil
}

// appendSection adds a [tool.gotpm] section at the end of the document,
// separated from what came before by one blank line.
func appendSection(lines, deps []string) []string {
	out := append([]string{}, lines...)
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	out = append(out, "["+toolTable+"."+gotpmTable+"]")
	return append(out, renderArray(deps)...)
}

// removeArray drops the dependency array, and the section header with it when
// only blank lines would remain in the section. Comments are kept: a section
// holding nothing but a comment is still something the user wrote.
func removeArray(lines []string, header, start, stop, end int) []string {
	rest := make([]string, 0, end-header)
	rest = append(rest, lines[header+1:start]...)
	rest = append(rest, lines[stop+1:end]...)

	for _, line := range rest {
		if strings.TrimSpace(line) != "" {
			return splice(lines, start, stop+1, nil)
		}
	}

	// Section is now empty: drop it, and the blank line that preceded it.
	from := header
	if from > 0 && strings.TrimSpace(lines[from-1]) == "" {
		from--
	}
	return splice(lines, from, end, nil)
}

// renderArray writes the dependency array as a multi-line TOML array, one
// entry per line, so adding or removing a dependency is a one-line diff.
func renderArray(deps []string) []string {
	out := make([]string, 0, len(deps)+arrayDelimiters)
	out = append(out, dependencies+" = [")
	for _, dep := range deps {
		out = append(out, arrayIndent+strconv.Quote(dep)+",")
	}
	return append(out, "]")
}

// findArray locates the dependency array between two line indices, returning
// the first and last line it spans and whether it was there at all.
func findArray(lines []string, from, to int) (int, int, bool) {
	for i := from; i < to; i++ {
		key, value, ok := strings.Cut(lines[i], "=")
		if !ok || strings.TrimSpace(key) != dependencies || isComment(lines[i]) {
			continue
		}
		depth := bracketDepth(value, 0)
		for j := i; ; j++ {
			if depth <= 0 || j+1 >= to {
				return i, j, true
			}
			depth = bracketDepth(lines[j+1], depth)
		}
	}
	return 0, 0, false
}

// findTable returns the line index of a table header, e.g. ["tool", "gotpm"]
// for "[tool.gotpm]".
func findTable(lines []string, want ...string) int {
	for i, line := range lines {
		if name, ok := tableName(line); ok && slices.Equal(name, want) {
			return i
		}
	}
	return -1
}

// tableEnd returns the line index at which the table opened at header ends,
// which is the next table header or the end of the document.
func tableEnd(lines []string, header int) int {
	for i := header + 1; i < len(lines); i++ {
		if _, ok := tableName(lines[i]); ok {
			return i
		}
	}
	return len(lines)
}

// tableName reads a table header line into its dotted parts. Array-of-table
// headers ("[[x]]") are reported as headers too, so a section is not mistaken
// for continuing past one.
func tableName(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
	inner = strings.TrimSuffix(strings.TrimPrefix(inner, "["), "]")
	if inner == "" {
		return nil, false
	}

	parts := strings.Split(inner, ".")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part == "" {
			return nil, false
		}
		parts[i] = part
	}
	return parts, true
}

// bracketDepth adds the bracket balance of a line to depth, ignoring brackets
// inside strings and after a comment marker.
func bracketDepth(line string, depth int) int {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"', '\'':
			i = skipString(line, i)
		case '#':
			return depth
		case '[':
			depth++
		case ']':
			depth--
		}
	}
	return depth
}

// skipString returns the index of the closing quote of the string literal
// opening at i, or the end of the line if it is unterminated.
func skipString(line string, i int) int {
	quote := line[i]
	for i++; i < len(line); i++ {
		switch {
		case line[i] == '\\' && quote == '"':
			i++
		case line[i] == quote:
			return i
		}
	}
	return i
}

func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// splice replaces lines[from:to] with replacement.
func splice(lines []string, from, to int, replacement []string) []string {
	out := make([]string, 0, len(lines)-(to-from)+len(replacement))
	out = append(out, lines[:from]...)
	out = append(out, replacement...)
	return append(out, lines[to:]...)
}
