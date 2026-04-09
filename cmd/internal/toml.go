package internal

import (
	"io"

	"github.com/BurntSushi/toml"
)

// Unmarshaler is a function that decodes TOML bytes into a value.
type Unmarshaler func([]byte, any) error

// ConfigurableUpdateToml writes the package metadata (name, version, entrypoint)
// back into the TOML document represented by data, using the provided writer and
// unmarshal function. The unmarshal parameter exists for testing.
func ConfigurableUpdateToml(w io.Writer, p PackageMeta, data []byte, unmarshal Unmarshaler, indent bool) error {
	var m map[string]any
	if err := unmarshal(data, &m); err != nil {
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
	if !indent {
		encoder.Indent = ""
	}
	return encoder.Encode(m)
}

// UpdateTOML writes the package metadata (name, version, entrypoint) back into
// the TOML document represented by data, using the provided writer.
func UpdateTOML(w io.Writer, p PackageMeta, data []byte, indent bool) error {
	return ConfigurableUpdateToml(w, p, data, toml.Unmarshal, indent)
}
