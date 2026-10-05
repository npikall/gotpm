package fonts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/paths"
)

var (
	ErrNotOwned          = errors.New("font family not installed by gotpm")
	ErrNotInstalled      = errors.New("font family is not installed")
	ErrChecksum          = errors.New("font file does not match its pinned digest")
	ErrInvalidProvenance = errors.New("invalid font provenance")
)

// Outcome is what had to be done to make the font directory match a pin.
type Outcome int

const (
	// UpToDate means the pinned commit was already installed.
	UpToDate Outcome = iota
	// Installed means nothing was there and the family was fetched.
	Installed
	// Replaced means something else was there and was overwritten.
	Replaced
	// Skipped means a family gotpm did not install was left in place.
	Skipped
)

// Provenance records, inside an installed family, where it was fetched from.
// It is what makes a family gotpm's to replace or delete.
type Provenance struct {
	Source string `json:"source"`
	URL    string `json:"url"`
	Hash   string `json:"hash"`
}

// Result reports what happened to one family.
type Result struct {
	// Pin is the pin that was installed, with the digest of every file.
	Pin     lockfile.Font
	Outcome Outcome
	// ReplacedSource is where the replaced family came from, the zero value
	// when it carried no provenance.
	ReplacedSource Provenance
	// VariableOnly reports a family whose files are all variable fonts, which
	// Typst renders at their default instance only.
	VariableOnly bool
}

// Dir is a font directory, one directory per family.
type Dir struct {
	Root string
}

// OpenDir returns the font directory under gotpm's data directory.
func OpenDir() (Dir, error) {
	root, err := paths.GotpmFontsDir()
	if err != nil {
		return Dir{}, err
	}
	return Dir{Root: root}, nil
}

// Path is the directory a family is installed in.
func (d Dir) Path(family string) string {
	return filepath.Join(d.Root, Key(family))
}

// ReadProvenance reports where an installed family came from, and whether it
// carries a provenance record at all.
func (d Dir) ReadProvenance(family string) (Provenance, bool, error) {
	path := d.provenancePath(family)
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

// Ensure makes the font directory hold exactly the commit a pin names. The
// family is fetched into a staging directory and swapped in once every file
// arrived and matched its digest, so a failure leaves what was installed
// untouched. A family without provenance is not gotpm's and is replaced only
// when forced.
func (d Dir) Ensure(ctx context.Context, src Source, pin lockfile.Font, force bool) (Result, error) {
	result, err := d.plan(pin, force)
	if err != nil || (result.Outcome == UpToDate && hasDigests(pin)) {
		return result, err
	}
	result.Pin, err = d.install(ctx, src, pin)
	return result, err
}

// fetchInto downloads every file of pin into dir, checking each against its
// pinned digest, or recording the digest when the pin has none yet. A pin may
// come from a lock on disk, so every file name is checked before it is used.
func fetchInto(ctx context.Context, src Source, pin lockfile.Font, dir string) (lockfile.Font, error) {
	files := slices.Clone(pin.Files)
	for i := range files {
		if err := validFileName(files[i].Name); err != nil {
			return pin, err
		}
		digest, err := fetchFile(ctx, src, pin, files[i], dir)
		if err != nil {
			return pin, err
		}
		files[i].SHA256 = digest
	}
	pin.Files = files
	return pin, nil
}

// fetchFile downloads one file of pin into dir and returns its digest.
func fetchFile(ctx context.Context, src Source, pin lockfile.Font, file lockfile.FontFile, dir string) (string, error) {
	data, err := src.Download(ctx, pin, file.Name)
	if err != nil {
		return "", fmt.Errorf("could not download %s of %q: %w", file.Name, pin.Family, err)
	}
	digest, err := verify(pin, file, data)
	if err != nil {
		return "", err
	}
	return digest, paths.WriteFile(filepath.Join(dir, file.Name), data)
}

// verify returns the digest of data, refusing one that differs from the digest
// the pin recorded for the file.
func verify(pin lockfile.Font, file lockfile.FontFile, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if file.SHA256 != "" && file.SHA256 != digest {
		return "", fmt.Errorf("%w: %s of %q at %s", ErrChecksum, file.Name, pin.Family, pin.Hash)
	}
	return digest, nil
}

func writeProvenance(dir string, p Provenance) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal font provenance: %w", err)
	}
	return paths.WriteFile(filepath.Join(dir, paths.ProvenanceFile), append(data, '\n'))
}

// swap moves staging into target's place. What target held is moved aside
// first and only deleted once staging is in, so a failed move puts it back.
func swap(staging, target string) error {
	aside := staging + ".old"
	if err := moveAside(target, aside); err != nil {
		return err
	}
	if err := putInPlace(staging, target, aside); err != nil {
		return err
	}
	if err := os.RemoveAll(aside); err != nil {
		return fmt.Errorf("could not remove the replaced font %q: %w", aside, err)
	}
	return nil
}

// moveAside renames target to aside; a target that does not exist yet is
// nothing to move.
func moveAside(target, aside string) error {
	err := os.Rename(target, aside)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not replace %q: %w", target, err)
	}
	return nil
}

// putInPlace renames staging to target, restoring what was moved aside when
// that fails.
func putInPlace(staging, target, aside string) error {
	if err := os.Rename(staging, target); err != nil {
		_ = os.Rename(aside, target)
		return fmt.Errorf("could not move font into %q: %w", target, err)
	}
	return nil
}

// hasDigests reports whether the pin records a digest for every file, which
// is what makes installing it verifiable.
func hasDigests(pin lockfile.Font) bool {
	return len(pin.Files) > 0 && !slices.ContainsFunc(pin.Files, func(f lockfile.FontFile) bool { return f.SHA256 == "" })
}

// Uninstall deletes an installed family. One without provenance is deleted
// only when forced.
func (d Dir) Uninstall(family string, force bool) error {
	if _, err := validKey(family); err != nil {
		return err
	}
	if !paths.IsDir(d.Path(family)) {
		return fmt.Errorf("%w: %q", ErrNotInstalled, family)
	}
	if err := d.checkOwned(family, force); err != nil {
		return err
	}
	return paths.Remove(d.Path(family))
}

func provenanceOf(pin lockfile.Font) Provenance {
	return Provenance{Source: pin.Source, URL: pin.URL, Hash: pin.Hash}
}

// variableOnly reports whether every file is a variable font, which Google
// Fonts marks by naming its axes in brackets: "Roboto[wdth,wght].ttf".
func variableOnly(files []lockfile.FontFile) bool {
	return len(files) > 0 && !slices.ContainsFunc(files, isStatic)
}

func isStatic(f lockfile.FontFile) bool {
	return !strings.Contains(f.Name, "[")
}

// EnsureAll installs every pin in turn. A family gotpm did not install is
// skipped with a warning instead of failing the lot, unless forced, so one
// hand-installed font does not stop a sync.
func (d Dir) EnsureAll(ctx context.Context, pins []lockfile.Font, force bool) ([]Result, error) {
	results := make([]Result, 0, len(pins))
	for _, pin := range pins {
		result, err := d.ensureOne(ctx, pin, force)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (d Dir) provenancePath(family string) string {
	return filepath.Join(d.Path(family), paths.ProvenanceFile)
}

// plan works out what installing pin has to do, without doing it.
func (d Dir) plan(pin lockfile.Font, force bool) (Result, error) {
	if _, err := validKey(pin.Family); err != nil {
		return Result{}, err
	}
	prov, owned, err := d.ReadProvenance(pin.Family)
	if err != nil {
		return Result{}, err
	}
	outcome, err := d.outcome(pin, prov, owned, force)
	return Result{Pin: pin, Outcome: outcome, ReplacedSource: prov, VariableOnly: variableOnly(pin.Files)}, err
}

func (d Dir) outcome(pin lockfile.Font, prov Provenance, owned, force bool) (Outcome, error) {
	if !paths.IsDir(d.Path(pin.Family)) {
		return Installed, nil
	}
	if !owned {
		return Replaced, d.allowForeign(pin.Family, force)
	}
	if prov == provenanceOf(pin) {
		return UpToDate, nil
	}
	return Replaced, nil
}

// allowForeign refuses to touch a family without provenance unless forced.
func (d Dir) allowForeign(family string, force bool) error {
	if force {
		return nil
	}
	return fmt.Errorf("%w: %q holds no %s"+
		"\nnote: pass --force to replace it, which changes the font for every project on this machine",
		ErrNotOwned, d.Path(family), paths.ProvenanceFile)
}

func (d Dir) install(ctx context.Context, src Source, pin lockfile.Font) (lockfile.Font, error) {
	staging, err := d.newStaging(pin.Family)
	if err != nil {
		return pin, err
	}
	defer os.RemoveAll(staging) //nolint: errcheck

	pin, err = fetchInto(ctx, src, pin, staging)
	if err != nil {
		return pin, err
	}
	if err := writeProvenance(staging, provenanceOf(pin)); err != nil {
		return pin, err
	}
	return pin, swap(staging, d.Path(pin.Family))
}

// newStaging creates the directory a family is fetched into before it is
// swapped in, beside its final place so the swap is a rename.
func (d Dir) newStaging(family string) (string, error) {
	if err := paths.EnsureDir(d.Root); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(d.Root, "."+Key(family)+"-")
	if err != nil {
		return "", fmt.Errorf("could not create staging directory: %w", err)
	}
	if err := os.Chmod(staging, paths.DirPerm); err != nil {
		return staging, fmt.Errorf("could not set permissions on %q: %w", staging, err)
	}
	return staging, nil
}

func (d Dir) checkOwned(family string, force bool) error {
	_, owned, err := d.ReadProvenance(family)
	if err != nil || owned {
		return err
	}
	return d.allowForeign(family, force)
}

func (d Dir) ensureOne(ctx context.Context, pin lockfile.Font, force bool) (Result, error) {
	src, err := SourceFor(pin.Source)
	if err != nil {
		return Result{}, fmt.Errorf("%q: %w", pin.Family, err)
	}
	if !hasDigests(pin) {
		return Result{}, fmt.Errorf("%w: the pin of %q records no digest for some of its files", ErrChecksum, pin.Family)
	}
	result, err := d.Ensure(ctx, src, pin, force)
	if errors.Is(err, ErrNotOwned) {
		return Result{Pin: pin, Outcome: Skipped}, nil
	}
	return result, err
}

// Notes describes what installing a family changed, as an info line and a
// warning, either of them empty.
func (r Result) Notes(d Dir) (string, string) {
	if r.Outcome == Skipped {
		return "", fmt.Sprintf("skipped font %q: %q was not installed by gotpm; pass --force to replace it",
			r.Pin.Family, d.Path(r.Pin.Family))
	}
	var info string
	if r.Outcome == Replaced {
		info = fmt.Sprintf("replaced font %q %s -> %s", r.Pin.Family, shortHash(r.ReplacedSource.Hash), shortHash(r.Pin.Hash))
	}
	return info, r.VariableWarning()
}

// VariableWarning warns about a family Typst cannot render in all its weights,
// or is empty.
func (r Result) VariableWarning() string {
	if !r.VariableOnly {
		return ""
	}
	return fmt.Sprintf("%q ships only variable fonts; typst renders every weight of them as regular", r.Pin.Family)
}

const shortHashLen = 8

func shortHash(hash string) string {
	if hash == "" {
		return "an unknown source"
	}
	return hash[:min(len(hash), shortHashLen)]
}
