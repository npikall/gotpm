package manifest

import (
	"errors"
	"io"
	"regexp"

	"github.com/BurntSushi/toml"
)

var ErrInvalidManifest = errors.New("not a valid typst manifest")

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

type UnmarshalFunc func([]byte, any) error

// The Default Unmarshal function.
func DefaultUnmarshaler(data []byte, v any) error {
	return toml.Unmarshal(data, v)
}

// Unmarshal a byte slice into a PackageInfo Struct
//
// The Unmarshal Function can be configured (e.g. for testing Purposes)
func ConfigureableUnmarshal(data []byte, unmarshal UnmarshalFunc) (PackageInfo, error) {
	var m Manifest
	err := unmarshal(data, &m)
	if err != nil {
		return PackageInfo{}, err
	}
	return m.Package, nil
}

// Unmarshal a byte slice into a PackageInfo Struct
func TypstTOMLUnmarshal(data []byte) (PackageInfo, error) {
	return ConfigureableUnmarshal(data, DefaultUnmarshaler)
}

// Write the Packageinfo (name, version and entrypoint) to io.Writer
//
// The Unmarshal Function can be configured (e.g. for testing Purposes)
func ConfigurableWriteTOML(w io.Writer, p PackageInfo, data []byte, unmarshal UnmarshalFunc) error {
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
	err = encoder.Encode(m)
	return err
}

// Write the Packageinfo (name, version and entrypoint) to io.Writer
func WriteTOML(w io.Writer, p PackageInfo, data []byte) error {
	return ConfigurableWriteTOML(w, p, data, DefaultUnmarshaler)
}
