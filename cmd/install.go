/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	ignore "github.com/sabhiram/go-gitignore"
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
	installCmd.Flags().StringP("namespace", "n", defaultNamespace, "The namespace in which the package should be available.")
}

func installRunner(cmd *cobra.Command, args []string) error {
	logger := setupLogger(cmd)
	sourceDir, err := resolveSourceDir(args)
	if err != nil {
		return err
	}
	logger.Debug("operating in source", "path", sourceDir)
	manifest, err := loadManifest(sourceDir)
	if err != nil {
		return err
	}
	logger.Debug("found package", "name", manifest.Package.Name, "version", manifest.Package.Version)
	dataDir, err := resolveLocalPackageDir()
	if err != nil {
		return err
	}
	logger.Debug("resolved local package directory", "path", dataDir)
	dest, err := resolveDestination(dataDir, manifest, cmd)
	if err != nil {
		return err
	}
	logger.Info("copy to destination", "path", dest.Path)
	err = copyPackageFiles(sourceDir, dest.Path)
	if err != nil {
		return err
	}
	importStmt := formatImportStmt(dest.Namespace, manifest.Package.Name, manifest.Package.Version)
	printInfo("installed %s", importStmt)
	return nil
}

const (
	manifestFileName     = "typst.toml"
	typstPackagesRelPath = "typst/packages"
	defaultNamespace     = "local"
)

var (
	ErrTooManyArguments        = errors.New("too many arguments: expected one directory path")
	ErrManifestNotFound        = errors.New("not found 'typst.toml': not a typst package directory")
	ErrInvalidManifest         = errors.New("invalid 'typst.toml'")
	ErrDataDirNotResolvable    = errors.New("could not resolve typst local package directory")
	ErrEmptyNamespace          = errors.New("namespace must not be empty")
	ErrPackageAlreadyInstalled = errors.New("package already installed at destination")
)

var ignoredFileNames = map[string]struct{}{
	".git":         {},
	".gitignore":   {},
	".typstignore": {},
}

func copyPackageFiles(src, dest string) error {
	matcher := buildIgnoreMatcher(src)
	jobs, err := collectJobs(src, dest, matcher)
	if err != nil {
		return err
	}
	return runTransferJobsWithSpinner(jobs)
}

func runTransferJobsWithSpinner(jobs []transferJob) error {
	spinner := setupSpinner()
	spinner.Start()
	time.Sleep(200 * time.Millisecond)
	defer spinner.Stop()
	return runTransferJobs(jobs)
}

func runTransferJobs(jobs []transferJob) error {
	n := len(jobs)
	errCh := make(chan error, n)

	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Go(func() {
			if err := copyFile(job.src, job.dst); err != nil {
				errCh <- err
				return
			}
		})
	}
	wg.Wait()
	close(errCh)
	return collectErrors(errCh)
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating parent directories for %q: %w", dest, err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %q: %w", src, err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("reading file info %q: %w", src, err)
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("creating destination file %q: %w", dest, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copying %q to %q: %w", src, dest, err)
	}
	return nil
}

func collectErrors(errCh <-chan error) error {
	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

func buildIgnoreMatcher(dir string) *ignore.GitIgnore {
	gitIgnorePath := filepath.Join(dir, ".gitignore")
	typstIgnorePath := filepath.Join(dir, ".typstignore")
	extraLines := readIgnoreLines(typstIgnorePath)
	if _, err := os.Stat(gitIgnorePath); err == nil {
		matcher, err := ignore.CompileIgnoreFileAndLines(gitIgnorePath, extraLines...)
		if err == nil {
			return matcher
		}
	}
	if len(extraLines) > 0 {
		return ignore.CompileIgnoreLines(extraLines...)
	}
	return nil
}

type transferJob struct {
	src string
	dst string
}

func collectJobs(src, dest string, matcher *ignore.GitIgnore) ([]transferJob, error) {
	var jobs []transferJob
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walking %q: %w", path, walkErr)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolving relative path %q: %w", path, err)
		}
		if shouldIgnore(rel, matcher) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			jobs = append(jobs, transferJob{
				src: path,
				dst: filepath.Join(dest, rel),
			})
		}
		return nil
	})
	return jobs, err
}

func shouldIgnore(rel string, matcher *ignore.GitIgnore) bool {
	if rel == "." {
		return false
	}
	if _, ok := ignoredFileNames[filepath.Base(rel)]; ok {
		return true
	}
	if matcher != nil && matcher.MatchesPath(rel) {
		return true
	}
	return false
}

func readIgnoreLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for line := range strings.Lines(string(data)) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

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

// Get the path to the directory in which a given Package will be stored into
func resolveDestination(dataDir string, manifest Manifest, cmd *cobra.Command) (Destination, error) {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return Destination{}, err
	}
	dest, err := resolveDestinationWithNamespace(dataDir, manifest, namespace)
	if err != nil {
		return Destination{}, err
	}
	return dest, nil
}

func resolveDestinationWithNamespace(dataDir string, manifest Manifest, namespace string) (Destination, error) {
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

// TODO: Move into internal package
// Get the path to the local directory, in which the packages are stored.
//
// ${data-dir}/typst/packages/
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

// Get the Source Directory either from the user provided argument or use the CWD
//
// Function will return an error if too many arguments have been provided
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
