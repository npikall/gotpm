/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied to
$DATA_DIR/typst/packages, where the $DATA_DIR is dependend on
the machines operating system.
`,
	Example: `# install Package located in the CWD
gotpm install
gotpm install --editable
gotpm install --namespace preview

# install a Package not in the CWD
gotpm install path/to/package/dir
gotpm install path/to/package/dir -n preview
`,
	RunE: installRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func installRunner(cmd *cobra.Command, args []string) error {
	sourceDir, err := resolveSourceDir(args)
	if err != nil {
		return err
	}
	manifest, err := loadManifest(sourceDir)
	if err != nil {
		return err
	}
	dataDir, err := resolveLocalPackageDir()
	if err != nil {
		return err
	}
	_, err = resolveDestination(dataDir, manifest, defaultNamespace) // TODO: make namespace variable
	if err != nil {
		return err
	}
	return nil
}

const (
	manifestFileName     = "typst.toml"
	typstPackagesRelPath = "typst/packages"
	defaultNamespace     = "local"
)

var (
	ErrTooManyArguments        = errors.New("too many arguments: expected one directory path")
	ErrManifestNotFound        = errors.New("'typst.toml' not found: not a typst package directory")
	ErrInvalidManifest         = errors.New("invalid 'typst.toml'")
	ErrDataDirNotResolvable    = errors.New("could not resolve typst local package directory")
	ErrEmptyNamespace          = errors.New("namespace must not be empty")
	ErrPackageAlreadyInstalled = errors.New("package already installed at destination")
)

type Manifest struct {
	Package PackageMeta `toml:"package"`
}

type PackageMeta struct {
	Name       string `toml:"name"`
	Version    string `toml:"version"`
	Entrypoint string `toml:"entrypoint"`
}

// Read and validate the typst.toml in the given directory.
func loadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, manifestFileName)
	raw, err := readManifestFile(path)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := parseManifest(raw)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func readManifestFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrManifestNotFound
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	return data, nil
}

func parseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %s", ErrInvalidManifest, err)
	}
	return m, nil
}

type Destination struct {
	Namespace string
	Name      string
	Version   string
	Path      string
}

func resolveDestination(dataDir string, manifest Manifest, namespace string) (Destination, error) {
	if err := validateNamespace(namespace); err != nil {
		return Destination{}, err
	}
	dest := buildDestination(dataDir, manifest, namespace)
	if err := validateDestinationConflict(dest.Path); err != nil {
		return Destination{}, err
	}
	return dest, nil
}

func buildDestination(dataDir string, manifest Manifest, namespace string) Destination {
	path := filepath.Join(
		dataDir,
		typstPackagesRelPath,
		namespace,
		manifest.Package.Name,
		manifest.Package.Version,
	)
	return Destination{
		Namespace: namespace,
		Name:      manifest.Package.Name,
		Version:   manifest.Package.Version,
		Path:      path,
	}
}

func resolveLocalPackageDir() (string, error) {
	base, err := resolveDataDir()
	if err != nil {
		return "", err
	}
	localPkgDir := filepath.Join(base, typstPackagesRelPath)
	if err := ensureDir(localPkgDir); err != nil {
		return "", err
	}
	return localPkgDir, nil
}

func resolveDataDir() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return resolveLinuxDataDir()
	case "darwin":
		return resolveDarwinDataDir()
	case "windows":
		return resolveWindowsDataDir()
	default:
		return "", fmt.Errorf("%w: unsupported OS %q", ErrDataDirNotResolvable, runtime.GOOS)
	}
}

func resolveLinuxDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, ".local", "share"), nil
}

func resolveDarwinDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDataDirNotResolvable, err)
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}

func resolveWindowsDataDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("%w: %%APPDATA%% is not set", ErrDataDirNotResolvable)
	}
	return appData, nil
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("creating directory %q: %w", path, err)
	}
	return nil
}

func resolveSourceDir(args []string) (string, error) {
	numberOfArgs := len(args)
	if numberOfArgs > 1 {
		return "", ErrTooManyArguments
	}
	if numberOfArgs == 0 {
		return os.Getwd()
	}
	return resolveProvidedPath(args[0])
}

func resolveProvidedPath(rawPath string) (string, error) {
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", rawPath, err)
	}
	if err := validateIsDirectory(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func validateDestinationConflict(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("%w: %q", ErrPackageAlreadyInstalled, path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("validate destination %q: %w", path, err)
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return ErrEmptyNamespace
	}
	return nil
}

func validateManifest(m Manifest) error {
	var errs []error
	if m.Package.Name == "" {
		errs = append(errs, errors.New("missing required field: package.name"))
	}
	if m.Package.Version == "" {
		errs = append(errs, errors.New("missing required field: package.version"))
	}
	if m.Package.Entrypoint == "" {
		errs = append(errs, errors.New("missing required field: package.entrypoint"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidManifest, errors.Join(errs...))
	}
	return nil
}

func validateIsDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %q", path)
		}
		return fmt.Errorf("accessing path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %q", path)
	}
	return nil
}
