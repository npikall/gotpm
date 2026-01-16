package files

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

var ErrInvalidManifest = errors.New("not a valid typst manifest")
var ErrLoadingTypstToml = errors.New("could not load typst.toml")

type Manifest struct {
	Package PackageInfo `toml:"package"`
}

// Structure with the required Typst TOML fields
type PackageInfo struct {
	// The Name of the Package
	Name string `toml:"name"`
	// The Version of the Package
	Version string `toml:"version"`
	// The Entrypoint of the Package
	Entrypoint string `toml:"entrypoint"`
}

func (p *PackageInfo) ValidateVersion() bool {
	match, _ := regexp.MatchString("^[0-9]*.[0-9]*.[0-9]*$", p.Version)
	return match
}

type Unmarshaler func([]byte, any) error

// Unmarshal a byte slice into a PackageInfo Struct
//
// The Unmarshal Function can be configured (e.g. for testing Purposes)
func ConfigureableUnmarshalToPackage(data []byte, unmarshal Unmarshaler) (PackageInfo, error) {
	var m Manifest
	err := unmarshal(data, &m)
	if err != nil {
		return PackageInfo{}, err
	}
	return m.Package, nil
}

// Unmarshal a byte slice into a PackageInfo Struct
func UnmarshalToPackage(data []byte) (PackageInfo, error) {
	return ConfigureableUnmarshalToPackage(data, toml.Unmarshal)
}

// Write the Packageinfo (name, version and entrypoint) to io.Writer
//
// The Unmarshal Function can be configured (e.g. for testing Purposes)
func ConfigurableUpdateToml(w io.Writer, p PackageInfo, data []byte, unmarshal Unmarshaler) error {
	var m map[string]any
	err := unmarshal(data, &m)
	if err != nil {
		return err
	}
	pkg, ok := m["package"].(map[string]any)
	if !ok {
		return ErrInvalidManifest
	}
	pkg["version"] = p.Version
	pkg["name"] = p.Name
	pkg["entrypoint"] = p.Entrypoint

	encoder := toml.NewEncoder(w)
	encoder.Indent = ""
	return encoder.Encode(m)
}

// Update the Packageinfo (name, version and entrypoint)
func UpdateTOML(w io.Writer, p PackageInfo, data []byte) error {
	return ConfigurableUpdateToml(w, p, data, toml.Unmarshal)
}

// Load a typst Package from a directory. Returns an error if not existing.
func LoadPackageFromDirectory(directory string) (PackageInfo, error) {
	tomlPath := filepath.Join(directory, "typst.toml")
	if !Exists(tomlPath) {
		return PackageInfo{}, ErrLoadingTypstToml
	}

	tomlContent, err := os.ReadFile(tomlPath)
	if err != nil {
		return PackageInfo{}, err
	}

	return UnmarshalToPackage(tomlContent)
}

// Check if a file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		return false
	}
	return true
}

// Copy a file from src to dst
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer closeFile(srcFile)

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer closeFile(dstFile)

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Sync()
	return err
}

func closeFile(f *os.File) {
	err := f.Close()
	if err != nil {
		panic(err)
	}
}
