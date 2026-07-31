package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal"
)

var (
	ErrNotSettablePath = errors.New("not a settable path")
	ErrUnknownKey      = errors.New("unknown config key")
)

type Fork struct {
	Path string `toml:"path"`
	URL  string `toml:"url"`
}

type Config struct {
	Fork Fork `toml:"fork"`
}

func Path() (string, error) {
	dataDir, err := internal.ResolveDataDir()
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	return filepath.Join(dataDir, "gotpm", "config.toml"), nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err //nolint: wrapcheck
	}
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
			return reflect.Value{}, fmt.Errorf("%w: %s", ErrNotSettablePath, key)
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
			return reflect.Value{}, fmt.Errorf("%w: %q", key, strings.Join(parts[:i+1], "."))
		}
	}
	return v, nil
}
