package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
)

// ProvenanceFile records, inside an installed package, which repository and
// commit it was built from.
//
// The store is shared by every project on the machine and by packages the user
// placed there by hand, so a coordinate like @gotpm/cetz:0.3.1 is not enough to
// know whether an installed directory is the one a project asked for. Keeping
// the answer next to the files means it survives a project being deleted and
// disappears with the package itself.
const ProvenanceFile = ".gotpm.json"

var ErrInvalidProvenance = errors.New("invalid provenance file")

// Provenance is the source an installed package version came from.
type Provenance struct {
	// URL is the repository without a scheme, e.g. "github.com/a/cetz".
	URL string `json:"url"`
	// Revision is the tag or branch that was asked for.
	Revision string `json:"revision"`
	// Hash is the commit that was checked out, and the only field that
	// identifies the content exactly.
	Hash string `json:"hash"`
}

// ProvenancePath returns the location of the provenance file of an installed
// package version.
func (s Store) ProvenancePath(ref pkg.Ref) string {
	return filepath.Join(s.Dir(ref), ProvenanceFile)
}

// WriteProvenance records where an installed package came from.
func (s Store) WriteProvenance(ref pkg.Ref, p Provenance) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal provenance for %s: %w", ref, err)
	}
	return paths.WriteFile(s.ProvenancePath(ref), append(data, '\n'))
}

// ReadProvenance reports where an installed package came from. A package
// without a provenance file was not installed by gotpm; that is not an error
// here, but callers must treat it as an unknown origin rather than a match.
func (s Store) ReadProvenance(ref pkg.Ref) (Provenance, bool, error) {
	path := s.ProvenancePath(ref)
	data, err := os.ReadFile(path) //nolint: gosec
	if errors.Is(err, os.ErrNotExist) {
		return Provenance{}, false, nil
	}
	if err != nil {
		return Provenance{}, false, fmt.Errorf("could not read %q: %w", path, err)
	}

	var p Provenance
	if err := json.Unmarshal(data, &p); err != nil {
		return Provenance{}, false, fmt.Errorf("%w %q: %w", ErrInvalidProvenance, path, err)
	}
	return p, true, nil
}
