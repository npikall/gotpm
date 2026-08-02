package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	ErrNotSettablePath = errors.New("not a settable path")
	ErrUnknownKey      = errors.New("unknown config key")
)

type Config struct {
	Fork ForkConfig `toml:"fork"`
}

type ForkConfig struct {
	Path string `toml:"path"`
	URL  string `toml:"url"`
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
	var entries []KV
	flatten("", reflect.ValueOf(cfg).Elem(), &entries)
	return entries
}

type KV struct {
	Key   string
	Value string
}

func flatten(prefix string, v reflect.Value, out *[]KV) {
	t := v.Type()
	for i := range t.NumField() {
		tag, _, _ := strings.Cut(t.Field(i).Tag.Get("toml"), ",")
		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			flatten(key, fv, out)
			continue
		}
		*out = append(*out, KV{Key: key, Value: fmt.Sprintf("%v", fv.Interface())})
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
	cfg := &Config{}
	_, _ = toml.DecodeFile(path, cfg)
	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint: gosec, mnd
		return err //nolint: wrapcheck
	}
	f, err := os.Create(path) //nolint: gosec
	if err != nil {
		return err //nolint: wrapcheck
	}
	defer f.Close() //nolint: errcheck

	return toml.NewEncoder(f).Encode(cfg) //nolint: wrapcheck
}

func fieldByTOMLPath(cfg any, key string) (reflect.Value, error) {
	v := reflect.ValueOf(cfg).Elem()
	parts := strings.Split(key, ".")

	for i, part := range parts {
		if v.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("%w: %q", ErrNotSettablePath, key)
		}
		found := false
		t := v.Type()
		for f := range t.NumField() {
			tag := t.Field(f).Tag.Get("toml")
			tag, _, _ = strings.Cut(tag, ",")
			if tag == part {
				v = v.Field(f)
				found = true
				break
			}
		}
		if !found {
			return reflect.Value{}, fmt.Errorf("%w: %q", ErrUnknownKey, strings.Join(parts[:i+1], "."))
		}
	}
	return v, nil
}

func assign(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected bool: %w", err)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("expected int: %w", err)
		}
		field.SetInt(n)
	default:
		return fmt.Errorf("unsupported field type %s", field.Kind())
	}
	return nil
}
