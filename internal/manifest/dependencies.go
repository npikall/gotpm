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
	toolTable       = "tool"
	gotpmTable      = "gotpm"
	dependencies    = "dependencies"
	fonts           = "fonts"
	arrayIndent     = "  "
	arrayDelimiters = 2
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

// SetDependencies rewrites the dependency array of [tool.gotpm], leaving every
// other byte as it was: the TOML encoder discards comments and reorders keys, and
// typst.toml is written by hand. An empty deps removes the array, and the section
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

	lines, newline, trailing := splitLines(string(content))
	updated, err := setArrayLines(lines, key, values)
	if err != nil {
		return fmt.Errorf("%q: %w", file, err)
	}
	return paths.WriteFile(file, []byte(joinLines(updated, newline, trailing)))
}

// splitLines splits text into lines, and reports the line ending it uses and
// whether it ends with one, so joinLines can put it back the same way.
func splitLines(text string) ([]string, string, bool) {
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	trailing := strings.HasSuffix(text, newline)
	return strings.Split(strings.TrimSuffix(text, newline), newline), newline, trailing
}

func joinLines(lines []string, newline string, trailing bool) string {
	out := strings.Join(lines, newline)
	if trailing || out != "" {
		out += newline
	}
	return out
}

func setArrayLines(lines []string, key string, values []string) ([]string, error) {
	header := findTable(lines, toolTable, gotpmTable)
	if header < 0 {
		return setWithoutSection(lines, key, values)
	}
	return setInSection(lines, header, key, values), nil
}

// setWithoutSection adds a [tool.gotpm] section for values, if there are any.
func setWithoutSection(lines []string, key string, values []string) ([]string, error) {
	if err := rejectInlineToolSection(lines); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return lines, nil
	}
	return appendSection(lines, key, values), nil
}

// setInSection writes the key array into the [tool.gotpm] section starting at
// header.
func setInSection(lines []string, header int, key string, values []string) []string {
	end := tableEnd(lines, header)
	start, stop, found := findArray(lines, key, header+1, end)
	if !found {
		return insertArray(lines, header, key, values)
	}
	if len(values) == 0 {
		return removeArray(lines, header, start, stop, end)
	}
	return splice(lines, start, stop+1, renderArray(key, values))
}

// insertArray writes the key array right below a section header that has none
// yet.
func insertArray(lines []string, header int, key string, values []string) []string {
	if len(values) == 0 {
		return lines
	}
	return splice(lines, header+1, header+1, renderArray(key, values))
}

func rejectInlineToolSection(lines []string) error {
	tool := findTable(lines, toolTable)
	if tool < 0 {
		return nil
	}
	if slices.ContainsFunc(lines[tool+1:tableEnd(lines, tool)], isInlineGotpmKey) {
		return ErrInlineToolSection
	}
	return nil
}

// isInlineGotpmKey reports whether line is "gotpm = ..." inside [tool].
func isInlineGotpmKey(line string) bool {
	key, _, ok := strings.Cut(line, "=")
	return ok && strings.TrimSpace(key) == gotpmTable
}

func appendSection(lines []string, key string, values []string) []string {
	out := append([]string{}, lines...)
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	out = append(out, "["+toolTable+"."+gotpmTable+"]")
	return append(out, renderArray(key, values)...)
}

func removeArray(lines []string, header, start, stop, end int) []string {
	rest := make([]string, 0, end-header)
	rest = append(rest, lines[header+1:start]...)
	rest = append(rest, lines[stop+1:end]...)

	for _, line := range rest {
		if strings.TrimSpace(line) != "" {
			return splice(lines, start, stop+1, nil)
		}
	}

	from := header
	if from > 0 && strings.TrimSpace(lines[from-1]) == "" {
		from--
	}
	return splice(lines, from, end, nil)
}

func renderArray(key string, values []string) []string {
	out := make([]string, 0, len(values)+arrayDelimiters)
	out = append(out, key+" = [")
	for _, value := range values {
		out = append(out, arrayIndent+strconv.Quote(value)+",")
	}
	return append(out, "]")
}

func findArray(lines []string, key string, from, to int) (int, int, bool) {
	for i := from; i < to; i++ {
		if value, ok := arrayValue(lines[i], key); ok {
			return i, arrayEnd(lines, i, to, value), true
		}
	}
	return 0, 0, false
}

// arrayValue returns what follows "<key> =" on line.
func arrayValue(line, key string) (string, bool) {
	name, value, ok := strings.Cut(line, "=")
	if !ok || isComment(line) {
		return "", false
	}
	return value, strings.TrimSpace(name) == key
}

// arrayEnd is the line, at or after start and before to, closing the array
// whose first line holds value.
func arrayEnd(lines []string, start, to int, value string) int {
	depth := bracketDepth(value, 0)
	end := start
	for depth > 0 && end+1 < to {
		end++
		depth = bracketDepth(lines[end], depth)
	}
	return end
}

func findTable(lines []string, want ...string) int {
	for i, line := range lines {
		if name, ok := tableName(line); ok && slices.Equal(name, want) {
			return i
		}
	}
	return -1
}

func tableEnd(lines []string, header int) int {
	for i := header + 1; i < len(lines); i++ {
		if _, ok := tableName(lines[i]); ok {
			return i
		}
	}
	return len(lines)
}

func tableName(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
	inner = strings.TrimSuffix(strings.TrimPrefix(inner, "["), "]")
	return tableKeys(inner)
}

// tableKeys splits a table name into its dotted keys, unquoted.
func tableKeys(inner string) ([]string, bool) {
	if inner == "" {
		return nil, false
	}
	parts := strings.Split(inner, ".")
	for i, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if part == "" {
			return nil, false
		}
		parts[i] = part
	}
	return parts, true
}

func bracketDepth(line string, depth int) int {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"', '\'':
			i = skipString(line, i)
		case '#':
			return depth
		default:
			depth += bracketDelta(line[i])
		}
	}
	return depth
}

func bracketDelta(c byte) int {
	switch c {
	case '[':
		return 1
	case ']':
		return -1
	default:
		return 0
	}
}

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

func splice(lines []string, from, to int, replacement []string) []string {
	out := make([]string, 0, len(lines)-(to-from)+len(replacement))
	out = append(out, lines[:from]...)
	out = append(out, replacement...)
	return append(out, lines[to:]...)
}
