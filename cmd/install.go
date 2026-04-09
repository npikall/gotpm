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
	"strings"
	"sync"

	cmdinternal "github.com/npikall/gotpm/cmd/internal"
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
	installCmd.Flags().StringP("namespace", "n", cmdinternal.DefaultNamespace, "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "Create a symlink to the source directory instead of copying files.")
}

func installRunner(cmd *cobra.Command, args []string) error {
	logger := cmdinternal.SetupLogger(cmd)
	sourceDir, err := resolveSourceDir(args)
	if err != nil {
		return err
	}
	logger.Debug("operating in source", "path", sourceDir)
	manifest, err := cmdinternal.LoadManifest(sourceDir)
	if err != nil {
		return err
	}
	logger.Debug("found package", "name", manifest.Package.Name, "version", manifest.Package.Version)
	dataDir, err := cmdinternal.ResolveLocalPackageDir()
	if err != nil {
		return err
	}
	logger.Debug("resolved local package directory", "path", dataDir)
	dest, err := resolveDestination(dataDir, manifest, cmd)
	if err != nil {
		return err
	}

	editable, err := cmd.Flags().GetBool("editable")
	if err != nil {
		return err
	}

	if editable {
		logger.Info("symlinking to destination", "path", dest.Path)
		if err := symlinkPackage(sourceDir, dest.Path); err != nil {
			return err
		}
		cmdinternal.PrintInfo("installed %s (editable)", cmdinternal.FormatImportStmt(dest.Namespace, manifest.Package.Name, manifest.Package.Version))
		return nil
	}

	logger.Info("copy to destination", "path", dest.Path)
	if err := copyPackageFiles(sourceDir, dest.Path); err != nil {
		return err
	}
	cmdinternal.PrintInfo("installed %s", cmdinternal.FormatImportStmt(dest.Namespace, manifest.Package.Name, manifest.Package.Version))
	return nil
}

var (
	ErrTooManyArguments        = errors.New("too many arguments: expected one directory path")
	ErrEmptyNamespace          = errors.New("namespace must not be empty")
	ErrPackageAlreadyInstalled = errors.New("package already installed at destination")
)

var ignoredFileNames = map[string]struct{}{
	".git":         {},
	".gitignore":   {},
	".typstignore": {},
}

// symlinkPackage creates a symlink at dest pointing to the absolute path of src.
// The parent directory of dest is created if it does not exist.
func symlinkPackage(src, dest string) error {
	if err := cmdinternal.EnsureDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("creating parent directory for symlink %q: %w", dest, err)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolving absolute path for symlink target %q: %w", src, err)
	}
	if err := os.Symlink(absSrc, dest); err != nil {
		return fmt.Errorf("creating symlink %q -> %q: %w", dest, absSrc, err)
	}
	return nil
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
	spinner := cmdinternal.SetupSpinner()
	spinner.Start()
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

type Destination struct {
	Namespace string
	Name      string
	Version   string
	Path      string
}

// resolveDestination builds the install destination using the namespace flag.
func resolveDestination(dataDir string, manifest cmdinternal.Manifest, cmd *cobra.Command) (Destination, error) {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return Destination{}, err
	}
	return resolveDestinationWithNamespace(dataDir, manifest, namespace)
}

func resolveDestinationWithNamespace(dataDir string, manifest cmdinternal.Manifest, namespace string) (Destination, error) {
	if err := validateNamespace(namespace); err != nil {
		return Destination{}, err
	}
	dest := buildDestination(dataDir, manifest, namespace)
	if err := validateDestinationConflict(dest.Path); err != nil {
		return Destination{}, err
	}
	return dest, nil
}

func buildDestination(dataDir string, manifest cmdinternal.Manifest, namespace string) Destination {
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
