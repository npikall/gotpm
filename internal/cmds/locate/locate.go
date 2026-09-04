// Package locate implements the locate command: it reports every path gotpm
// reads or writes.
package locate

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/log/v2"
)

var ErrUnknownKey = errors.New("unknown key")

// Run prints the paths gotpm uses: one key unstyled so it survives command
// substitution, or all of them grouped. Nothing here creates a directory.
func Run(key string, log *log.Logger) error {
	if key != "" {
		return printKey(key, log)
	}
	groups := entries(log)
	log.Debug("resolved", "groups", len(groups))
	render(groups)
	return nil
}

func printKey(key string, log *log.Logger) error {
	if !slices.Contains(Keys(), key) {
		return fmt.Errorf("%w %q\nvalid keys: %s", ErrUnknownKey, key, strings.Join(Keys(), ", "))
	}
	entry, err := lookup(key)
	if err != nil {
		return err
	}
	log.Debug("resolved", "key", key, "path", entry.Path)
	fmt.Println(entry.Path)
	return nil
}

func lookup(key string) (Entry, error) {
	machine := append([]Entry{packagesEntry()}, gotpmEntries()...)
	if entry, found := find(machine, key); found {
		return entry, entry.Err
	}
	project, err := projectEntries()
	if err != nil {
		return Entry{}, err
	}
	if entry, found := find(project, key); found {
		return entry, entry.Err
	}
	return Entry{}, fmt.Errorf("%w %q", ErrUnknownKey, key)
}

func find(entries []Entry, key string) (Entry, bool) {
	for _, entry := range entries {
		if entry.Key == key {
			return entry, true
		}
	}
	return Entry{}, false
}
