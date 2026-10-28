package compilers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/ulikunitz/xz"
)

const (
	// binaryPerm makes the compiler executable by its owner.
	binaryPerm fs.FileMode = 0o755
	// maxBinarySize bounds what is unpacked from an archive, far above the
	// size of any Typst binary.
	maxBinarySize = 1 << 30
	digestPrefix  = "sha256:"
)

var errNoBinary = errors.New("no typst binary inside")

// Dir is a compiler directory, one directory per version.
type Dir struct {
	Root string
}

// OpenDir returns the compiler directory under gotpm's data directory.
func OpenDir() (Dir, error) {
	root, err := paths.GotpmCompilersDir()
	if err != nil {
		return Dir{}, err
	}
	return Dir{Root: root}, nil
}

// Path is the directory a version is installed in.
func (d Dir) Path(version string) string {
	return filepath.Join(d.Root, version)
}

// Binary is the path of an installed compiler's binary.
func (d Dir) Binary(version string) (string, error) {
	if err := validVersion(version); err != nil {
		return "", err
	}
	binary := d.binaryPath(version)
	if _, err := os.Stat(binary); err != nil {
		return "", fmt.Errorf("%w: %s", ErrNotInstalled, version)
	}
	return binary, nil
}

// Installed lists the versions of the installed compilers, oldest first.
func (d Dir) Installed() ([]string, error) {
	entries, err := os.ReadDir(d.Root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read %q: %w", d.Root, err)
	}
	var versions []string
	for _, entry := range entries {
		if _, err := d.Binary(entry.Name()); err == nil {
			versions = append(versions, entry.Name())
		}
	}
	slices.SortFunc(versions, compareVersions)
	return versions, nil
}

// Ensure installs version from src unless it is installed already, which it
// finds out without asking src. It reports whether it installed anything.
func (d Dir) Ensure(ctx context.Context, src Source, version string) (bool, error) {
	if _, err := d.Binary(version); !errors.Is(err, ErrNotInstalled) {
		return false, err
	}
	archive, asset, err := download(ctx, src, version)
	if err != nil {
		return false, err
	}
	return true, d.install(version, asset, archive)
}

func (d Dir) binaryPath(version string) string {
	return filepath.Join(d.Path(version), binaryName())
}

// install unpacks the binary into a staging directory beside its final place
// and renames it there. Losing the rename to a concurrent install of the same
// version is fine: what is in place is the same compiler.
func (d Dir) install(version string, asset Asset, archive []byte) error {
	staging, err := d.newStaging(version)
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging) //nolint: errcheck

	if err := unpack(asset, archive, filepath.Join(staging, binaryName())); err != nil {
		return err
	}
	if err := os.Rename(staging, d.Path(version)); err != nil {
		if _, statErr := d.Binary(version); statErr == nil {
			return nil // a concurrent install got there first
		}
		return fmt.Errorf("could not move compiler into %q: %w", d.Path(version), err)
	}
	return nil
}

func (d Dir) newStaging(version string) (string, error) {
	if err := paths.EnsureDir(d.Root); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(d.Root, "."+version+"-")
	if err != nil {
		return "", fmt.Errorf("could not create staging directory: %w", err)
	}
	if err := os.Chmod(staging, paths.DirPerm); err != nil {
		return staging, fmt.Errorf("could not set permissions on %q: %w", staging, err)
	}
	return staging, nil
}

// download fetches the archive of version for this platform and checks it
// against its digest.
func download(ctx context.Context, src Source, version string) ([]byte, Asset, error) {
	release, err := src.Release(ctx, version)
	if err != nil {
		return nil, Asset{}, err
	}
	asset, err := release.Asset()
	if err != nil {
		return nil, Asset{}, err
	}
	archive, err := src.Download(ctx, asset)
	if err != nil {
		return nil, Asset{}, fmt.Errorf("could not download compiler %s: %w", version, err)
	}
	return archive, asset, verify(version, asset, archive)
}

func verify(version string, asset Asset, archive []byte) error {
	want, ok := strings.CutPrefix(asset.Digest, digestPrefix)
	if !ok {
		return fmt.Errorf("%w: %s of compiler %s", ErrNoDigest, asset.Name, version)
	}
	sum := sha256.Sum256(archive)
	if hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("%w: %s of compiler %s", ErrChecksum, asset.Name, version)
	}
	return nil
}

// unpack writes the compiler binary held in archive to target.
func unpack(asset Asset, archive []byte, target string) error {
	var binary io.ReadCloser
	var err error
	if strings.HasSuffix(asset.Name, ".zip") {
		binary, err = fromZip(archive)
	} else {
		binary, err = fromTarXz(archive)
	}
	if err != nil {
		return fmt.Errorf("%w %s: %w", ErrInvalidArchive, asset.Name, err)
	}
	defer binary.Close() //nolint: errcheck
	return writeBinary(target, binary)
}

func fromTarXz(archive []byte) (io.ReadCloser, error) {
	xzr, err := xz.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("could not read xz: %w", err)
	}
	tr := tar.NewReader(xzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errNoBinary
		}
		if err != nil {
			return nil, fmt.Errorf("could not read tar: %w", err)
		}
		if header.Typeflag == tar.TypeReg && path.Base(header.Name) == binaryName() {
			return io.NopCloser(tr), nil
		}
	}
}

func fromZip(archive []byte) (io.ReadCloser, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("could not read zip: %w", err)
	}
	for _, file := range zr.File {
		if file.Mode().IsRegular() && path.Base(file.Name) == binaryName() {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("could not open %s: %w", file.Name, err)
			}
			return rc, nil
		}
	}
	return nil, errNoBinary
}

func writeBinary(target string, binary io.Reader) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, binaryPerm) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not create %q: %w", target, err)
	}
	if _, err := io.Copy(f, io.LimitReader(binary, maxBinarySize)); err != nil {
		_ = f.Close()
		return fmt.Errorf("could not write %q: %w", target, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("could not write %q: %w", target, err)
	}
	return nil
}
