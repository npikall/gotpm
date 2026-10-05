// Package fonts obtains font families from a font source and keeps them in the
// font directory, the machine-wide directory Typst reads through
// $TYPST_FONT_PATHS.
//
// A font family is fetched at the exact commit a font pin names and its files
// are checked against the digests the pin recorded, so a project's fonts are as
// reproducible as its packages. Like the package directory, the font directory
// is shared, so a family is only ever replaced or deleted when its provenance
// shows gotpm put it there.
package fonts

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/npikall/gotpm/internal/fontname"
	"github.com/npikall/gotpm/internal/lockfile"
)

var (
	ErrInvalidName    = errors.New("invalid font name")
	ErrFontNotFound   = errors.New("font not found")
	ErrNoFontFiles    = errors.New("font has no files listed")
	ErrInvalidFileRef = errors.New("invalid font file name")
	ErrFailedRequest  = errors.New("http request failed")
	ErrUnknownSource  = errors.New("unknown font source")
)

// Source is somewhere font families are obtained from.
type Source interface {
	// Name identifies the source in a font pin.
	Name() string
	// Resolve pins the newest commit of a family. The pin names its files
	// without digests; those are recorded when the files are fetched.
	Resolve(ctx context.Context, family string) (lockfile.Font, error)
	// Download fetches one file of a pinned family.
	Download(ctx context.Context, pin lockfile.Font, file string) ([]byte, error)
	// Families lists every family the source offers.
	Families(ctx context.Context) ([]Listing, error)
}

// Listing is one family a source offers.
type Listing struct {
	// Family is the family's key, e.g. "opensans".
	Family string `json:"family"`
	// License is the license the family is published under, e.g. "ofl".
	License string `json:"license"`
}

var (
	sourcesMu sync.RWMutex
	sources   = map[string]Source{GoogleSource: NewGoogle()}
)

// Default is the source a font is added from when none is named.
func Default() Source { //nolint: ireturn // sources are interchangeable by design
	s, _ := SourceFor(GoogleSource)
	return s
}

// SourceFor returns the source a font pin names.
func SourceFor(name string) (Source, error) { //nolint: ireturn // a pin names its source at run time
	sourcesMu.RLock()
	defer sourcesMu.RUnlock()
	s, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownSource, name)
	}
	return s, nil
}

// Use replaces the source registered under s.Name() and returns a function
// restoring the previous one. It exists for tests.
func Use(s Source) func() {
	sourcesMu.Lock()
	defer sourcesMu.Unlock()
	previous := sources[s.Name()]
	sources[s.Name()] = s
	return func() {
		sourcesMu.Lock()
		defer sourcesMu.Unlock()
		sources[s.Name()] = previous
	}
}

// Key is the comparable form of a family name, which is also the name of the
// family's directory: "Open Sans" is kept in "opensans".
func Key(family string) string {
	return fontname.Key(family)
}

func validKey(family string) (string, error) {
	key := Key(family)
	if key == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, family)
	}
	return key, nil
}
