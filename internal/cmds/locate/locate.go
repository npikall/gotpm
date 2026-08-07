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

// Run prints the paths gotpm uses. With a key it prints that one path on its
// own, unstyled, so it survives command substitution; without one it prints
// them all, grouped.
//
// Nothing here creates a directory: locate answers where gotpm would look, and
// a path that does not exist yet is still the right answer.
func Run(key string, log *log.Logger) error {
	if key != "" {
		return printKey(key, log)
	}
	groups := entries(log)
	log.Debug("resolved", "groups", len(groups))
	render(groups)
	return nil
}

// printKey writes a single path to stdout with no prefix, styling or note.
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

// lookup resolves one key. The project paths are only touched when the key
// names one of them, so asking for a machine path outside a project still
// works. When it does touch them, a missing project is an error rather than
// the silent omission the dump does: the caller asked for that path by name
// and would otherwise get nothing back.
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
