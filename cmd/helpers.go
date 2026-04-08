package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/spf13/cobra"
)

// Helper to log in GoRoutines
type logEvent struct {
	level   string
	msg     string
	keyvals []any
}

var (
	Blue    = charmtone.Malibu // Sardine
	Green   = charmtone.Bok    // Guac, Julep
	Yellow  = charmtone.Zest   // Citron, Mustard
	Red     = charmtone.Coral  // Sriracha, Chili
	Violet  = charmtone.Charple
	Magenta = charmtone.Cheeky
	Normal  = charmtone.Smoke // White
	Muted   = charmtone.Squid // Darker White
	Accent  = charmtone.Ash   // Brighter White
)

var (
	StyleBlueBold    = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	StyleBlue        = lipgloss.NewStyle().Foreground(Blue)
	StyleGreen       = lipgloss.NewStyle().Foreground(Green)
	StyleYellow      = lipgloss.NewStyle().Foreground(Yellow)
	StyleRed         = lipgloss.NewStyle().Foreground(Red)
	StyleNormal      = lipgloss.NewStyle().Foreground(Normal)
	StyleMuted       = lipgloss.NewStyle().Foreground(Muted)
	StyleAccent      = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	StyleLogo        = lipgloss.NewStyle().Foreground(Violet)
	StyleDescription = lipgloss.NewStyle().Foreground(Magenta)
)

func printInfo(format string, a ...any) {
	prefix := StyleBlueBold.Render("info")
	text := StyleNormal.Render(fmt.Sprintf(format, a...))
	fmt.Printf("%s: %s\n", prefix, text)
}

func printWarn(format string, a ...any) {
	prefix := StyleBlueBold.Render("warning")
	text := StyleNormal.Render(fmt.Sprintf(format, a...))
	fmt.Printf("%s: %s\n", prefix, text)
}

func formatImportStmt(namespace, name, version string) string {
	return StyleAccent.Render(fmt.Sprintf("@%s/%s:%s", namespace, name, version))
}

func setupLogger(cmd *cobra.Command) *log.Logger {
	logger := log.New(os.Stdout)
	logger.SetReportTimestamp(true)
	verboseCount, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		logger.SetLevel(log.WarnLevel)
		return logger
	}
	switch {
	case verboseCount >= 2:
		logger.SetLevel(log.DebugLevel)
	case verboseCount == 1:
		logger.SetLevel(log.InfoLevel)
	default:
		logger.SetLevel(log.WarnLevel)
	}
	return logger
}

func setupSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = StyleMuted.Render(" Loading...")
	_ = s.Color("cyan")
	return s
}

// A given Function must return no error.
// When an error occurs the program is exited.
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}

const (
	manifestFileName     = "typst.toml"
	typstPackagesRelPath = "typst/packages"
	defaultNamespace     = "local"
)

var (
	ErrManifestNotFound     = errors.New("not found 'typst.toml': not a typst package directory")
	ErrInvalidManifest      = errors.New("invalid 'typst.toml'")
	ErrDataDirNotResolvable = errors.New("could not resolve typst local package directory")
)

type Manifest struct {
	Package PackageMeta `toml:"package"`
}

type PackageMeta struct {
	Name       string `toml:"name"`
	Version    string `toml:"version"`
	Entrypoint string `toml:"entrypoint"`
}

// Read and validates the typst.toml in the given directory.
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

// Return the path to the local Typst packages
// directory without creating it. Respects the $TYPST_PACKAGE_PATH override.
func resolveLocalPackageDirPath() (string, error) {
	if override := os.Getenv("TYPST_PACKAGE_PATH"); override != "" {
		return override, nil
	}
	base, err := resolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, typstPackagesRelPath), nil
}
