// Package config implements the config command: it reads and writes gotpm's
// configuration values.
package config

import (
	"fmt"

	cfgfile "github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/ui"
)

// Set stores a value under a config key.
func Set(key, value string) error {
	cfg, err := cfgfile.Load()
	if err != nil {
		return err
	}
	if err := cfg.Set(key, value); err != nil {
		return err
	}
	if err := cfgfile.Save(cfg); err != nil {
		return err
	}
	ui.Infof("%s = %s", key, value)
	return nil
}

// Get prints the value stored under a config key.
func Get(key string) error {
	cfg, err := cfgfile.Load()
	if err != nil {
		return err
	}
	value, err := cfg.Get(key)
	if err != nil {
		return err
	}
	ui.Infof("%s = %s", key, value)
	return nil
}

// Unset clears the value stored under a config key.
func Unset(key string) error {
	cfg, err := cfgfile.Load()
	if err != nil {
		return err
	}
	if err := cfg.Unset(key); err != nil {
		return err
	}
	if err := cfgfile.Save(cfg); err != nil {
		return err
	}
	ui.Infof("%s = ", key)
	return nil
}

// List prints every config key and its value.
func List() error {
	cfg, err := cfgfile.Load()
	if err != nil {
		return err
	}
	path, err := cfgfile.Path()
	if err != nil {
		return err
	}
	ui.Infof("current config at %s\n", path)
	for _, entry := range cfg.Entries() {
		fmt.Printf("%s = %s\n", entry.Key, entry.Value)
	}
	return nil
}
