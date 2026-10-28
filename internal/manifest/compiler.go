package manifest

import (
	"fmt"
	"os"

	"github.com/npikall/gotpm/internal/paths"
	tomledit "github.com/npikall/toml-edit"
)

// CompilerPin returns the exact compiler the project is compiled with, or ""
// when the manifest pins none. Unlike Package.Compiler it is not a floor.
func (m *Manifest) CompilerPin() string {
	return m.Tool.Gotpm.CompilerPin
}

// SetCompilerPin writes version as the compiler pin of [tool.gotpm] in place,
// adding the key, and the section with it, when missing. Setting the pin it
// already holds leaves the file as it is.
func SetCompilerPin(file, version string) error {
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return fmt.Errorf("could not read %q: %w", file, err)
	}

	doc, err := tomledit.Parse(string(content))
	if err != nil {
		return fmt.Errorf("%q: %w: %w", file, ErrInvalidManifest, err)
	}
	changed, err := setGotpmString(doc, compilerPin, version)
	if err != nil {
		return fmt.Errorf("%q: %w: %w", file, ErrInvalidManifest, err)
	}
	if !changed {
		return nil
	}
	return paths.WriteFile(file, []byte(doc.String()))
}

// setGotpmString writes value to the key of [tool.gotpm] and reports whether
// the document changed. A key that holds anything but a string is an error.
func setGotpmString(doc *tomledit.Document, key, value string) (bool, error) {
	path := []string{toolTable, gotpmTable, key}
	if _, exists := doc.Get(path...); !exists {
		return true, doc.Insert(path, value) //nolint: wrapcheck
	}
	current, err := doc.GetString(path...)
	if err != nil || current == value {
		return false, err //nolint: wrapcheck
	}
	return true, doc.Set(path, value) //nolint: wrapcheck
}
