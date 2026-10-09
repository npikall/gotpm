// Package config reads and writes gotpm's own configuration file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/paths"
	tomledit "github.com/npikall/toml-edit"
	"github.com/npikall/toml-edit/eval"
)

var (
	ErrNotSettablePath  = errors.New("not a settable path")
	ErrUnknownKey       = errors.New("unknown config key")
	ErrUnsupportedField = errors.New("unsupported field type")
)

// Template is what a new config file starts as when it is edited by hand:
// every key, commented out and described.
const Template = `# gotpm config. Uncomment a key to set it.

# URL of the forked package repository submissions are pushed to.
# fork.url = "https://github.com/you/packages"

# Local directory the fork gets cloned into.
# fork.path = "/home/you/typst-packages"
`

type Config struct {
	Fork ForkConfig `toml:"fork,omitempty"`
}

type ForkConfig struct {
	Path string `toml:"path,omitempty"`
	URL  string `toml:"url,omitempty"`
}

func (cfg *Config) Set(key, value string) error {
	field, err := fieldByTOMLPath(cfg, key)
	if err != nil {
		return err
	}
	return assign(field, value)
}

func (cfg *Config) Unset(key string) error {
	field, err := fieldByTOMLPath(cfg, key)
	if err != nil {
		return err
	}
	field.Set(reflect.Zero(field.Type()))
	return nil
}

func (cfg *Config) Get(key string) (string, error) {
	field, err := fieldByTOMLPath(cfg, key)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", field.Interface()), nil
}

// Entries returns every leaf config value as a dotted key and its current
// value, in struct-declaration order.
func (cfg *Config) Entries() []KV {
	leaves := cfg.leaves()
	entries := make([]KV, 0, len(leaves))
	for _, leaf := range leaves {
		entries = append(entries, KV{Key: strings.Join(leaf.path, "."), Value: fmt.Sprintf("%v", leaf.value.Interface())})
	}
	return entries
}

type KV struct {
	Key   string
	Value string
}

// leaf is a config value together with the key path it is stored under.
type leaf struct {
	path  []string
	value reflect.Value
}

// leaves returns every config value in struct-declaration order.
func (cfg *Config) leaves() []leaf {
	var out []leaf
	flatten(nil, reflect.ValueOf(cfg).Elem(), &out)
	return out
}

func flatten(prefix []string, v reflect.Value, out *[]leaf) {
	t := v.Type()
	for i := range t.NumField() {
		tag, _, _ := strings.Cut(t.Field(i).Tag.Get("toml"), ",")
		path := append(slices.Clip(prefix), tag)
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			flatten(path, fv, out)
			continue
		}
		*out = append(*out, leaf{path: path, value: fv})
	}
}

func Path() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	return filepath.Join(configDir, "gotpm", "config.toml"), nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := readIfExists(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// readIfExists reads the file at path, which reads as empty if there is none.
func readIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint: gosec
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err //nolint: wrapcheck
}

// Parse decodes a config file's contents, rejecting any key gotpm does not
// know, so a typo is reported rather than silently ignored.
func Parse(data []byte) (*Config, error) {
	cfg := &Config{}
	meta, err := toml.Decode(string(data), cfg)
	if err != nil {
		return nil, err //nolint: wrapcheck
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, undecoded[0].String())
	}
	return cfg, nil
}

// Save writes cfg to the config file, editing it in place so the user's
// comments and layout survive. A value left empty is removed, and a table
// with it once nothing is left in it.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := readIfExists(path)
	if err != nil {
		return err
	}
	doc, err := tomledit.Parse(string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for _, leaf := range cfg.leaves() {
		if err := store(doc, leaf); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint: gosec, mnd
		return err //nolint: wrapcheck
	}
	return paths.WriteFile(path, []byte(doc.String()))
}

// store writes one config value to doc: set when the key holds another value,
// inserted when it is missing, removed when the value is empty. A value that
// is already there stays as the user spelled it.
func store(doc *tomledit.Document, value leaf) error {
	current, exists := doc.Get(value.path...)
	switch {
	case value.value.IsZero() && !exists:
		return nil
	case exists && fmt.Sprint(current) == fmt.Sprint(value.value.Interface()):
		return nil
	case value.value.IsZero():
		if err := doc.Delete(value.path...); err != nil {
			return err //nolint: wrapcheck
		}
		return deleteIfEmpty(doc, value.path[:len(value.path)-1])
	case exists:
		return doc.Set(value.path, value.value.Interface()) //nolint: wrapcheck
	default:
		return doc.Insert(value.path, value.value.Interface()) //nolint: wrapcheck
	}
}

// deleteIfEmpty removes the table at path if it holds nothing.
func deleteIfEmpty(doc *tomledit.Document, path []string) error {
	if len(path) == 0 {
		return nil
	}
	v, _ := doc.Get(path...)
	if table, ok := v.(*eval.Table); !ok || len(table.Keys()) > 0 {
		return nil
	}
	return doc.Delete(path...) //nolint: wrapcheck
}

func fieldByTOMLPath(cfg any, key string) (reflect.Value, error) {
	v := reflect.ValueOf(cfg).Elem()
	parts := strings.Split(key, ".")

	for i, part := range parts {
		if v.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("%w: %q", ErrNotSettablePath, key)
		}
		field, found := fieldByTag(v, part)
		if !found {
			return reflect.Value{}, fmt.Errorf("%w: %q", ErrUnknownKey, strings.Join(parts[:i+1], "."))
		}
		v = field
	}
	return v, nil
}

// fieldByTag finds the field of struct v whose toml tag names it.
func fieldByTag(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for f := range t.NumField() {
		tag, _, _ := strings.Cut(t.Field(f).Tag.Get("toml"), ",")
		if tag == name {
			return v.Field(f), true
		}
	}
	return reflect.Value{}, false
}

func assign(field reflect.Value, value string) error {
	switch field.Kind() { //nolint: exhaustive
	case reflect.String:
		field.SetString(value)
		return nil
	case reflect.Bool:
		return assignBool(field, value)
	case reflect.Int, reflect.Int64:
		return assignInt(field, value)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedField, field.Kind())
	}
}

func assignBool(field reflect.Value, value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("expected bool: %w", err)
	}
	field.SetBool(b)
	return nil
}

func assignInt(field reflect.Value, value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("expected int: %w", err)
	}
	field.SetInt(n)
	return nil
}
