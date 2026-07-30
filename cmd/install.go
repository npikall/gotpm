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

	"github.com/npikall/gotpm/internal"
	"github.com/npikall/gotpm/internal/remote"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied to
$DATA_DIR/typst/packages, where the $DATA_DIR is dependent on
the machine's operating system.

The destination directory can be overridden via the --install-dir flag
or the GOTPM_INSTALL_DIR environment variable. The flag takes precedence.
`,
	Example: `# install Package located in the CWD
gotpm install
gotpm install --editable
gotpm install --namespace preview

# install a Package not in the CWD
gotpm install path/to/package/dir
gotpm install path/to/package/dir -n preview
`,
	RunE: InstallRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", internal.DefaultNamespace, "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "Create a symlink to the source directory instead of copying files.")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite an already-installed package.")
	installCmd.Flags().String(internal.InstallDirFlag, "", "Override the package directory (env: $"+internal.InstallDirEnvVar+")")
	installCmd.Flags().StringP("remote", "r", "", "The remote repository which should be installed.")
}

func InstallRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)
	opts := ReadInstallOptions(cmd)

	sourceDir, err := ResolveSourceDir(args, opts.Remote)
	if err != nil {
		return err
	}
	logger.Debug("operating in source", "path", sourceDir)

	manifest, err := internal.LoadManifest(sourceDir)
	if err != nil {
		return fmt.Errorf("could not load manifest: %w", err)
	}
	logger.Debug("found package", "name", manifest.Package.Name, "version", manifest.Package.Version)

	dest, err := resolveInstallDestination(cmd, manifest, opts)
	if err != nil {
		return err
	}
	logger.Debug("resolved destination", "path", dest.Path)

	if err := prepareDestination(dest.Path, opts.Force); err != nil {
		return err
	}

	return performInstall(sourceDir, dest, opts)
}

var (
	ErrTooManyArguments        = errors.New("too many arguments: expected one directory path")
	ErrEmptyNamespace          = errors.New("namespace must not be empty")
	ErrPackageAlreadyInstalled = errors.New("package already installed at destination")
	ErrNotADirectory           = errors.New("path is not a directory")
)

// InstallOptions holds the resolved install flags.
type InstallOptions struct {
	Force      bool
	Editable   bool
	Namespace  string
	Remote     string
	InstallDir string
}

func ReadInstallOptions(cmd *cobra.Command) *InstallOptions {
	force := internal.Must(cmd.Flags().GetBool("force"))
	remote := internal.Must(cmd.Flags().GetString("remote"))
	editable := internal.Must(cmd.Flags().GetBool("editable"))
	namespace := internal.Must(cmd.Flags().GetString("namespace"))
	installDir := internal.Must(cmd.Flags().GetString(internal.InstallDirFlag))
	return &InstallOptions{
		Force:      force,
		Editable:   editable,
		Namespace:  namespace,
		Remote:     remote,
		InstallDir: installDir,
	}
}

// resolveInstallDestination routes to the appropriate destination resolver based
// on whether an install-dir override was provided.
func resolveInstallDestination(cmd *cobra.Command, manifest internal.Manifest, opts InstallOptions) (Destination, error) {
	dataDir, overridden, err := internal.ResolvePackageDirPath(cmd)
	if err != nil {
		return Destination{}, fmt.Errorf("could not resolve package directory: %w", err)
	}
	if overridden {
		return ResolveOverriddenDestination(dataDir, manifest, opts)
	}
	return ResolveDefaultDestination(dataDir, manifest, opts)
}

// ResolveOverriddenDestination is used when --install-dir or $GOTPM_INSTALL_DIR is set.
// dataDir is used as the final install path without appending namespace/name/version.
func ResolveOverriddenDestination(dataDir string, manifest internal.Manifest, opts *InstallOptions) (Destination, error) {
	if !opts.Force {
		if err := ValidateDestinationConflict(dataDir); err != nil {
			return Destination{}, err
		}
	}
	return Destination{
		Namespace: opts.Namespace,
		Name:      manifest.Package.Name,
		Version:   manifest.Package.Version,
		Path:      dataDir,
	}, nil
}

// ResolveDefaultDestination is used for the standard install path.
// It appends namespace/name/version sub-directories to dataDir.
func ResolveDefaultDestination(dataDir string, manifest internal.Manifest, opts *InstallOptions) (Destination, error) {
	if err := internal.EnsureDir(dataDir); err != nil {
		return Destination{}, fmt.Errorf("could not ensure directory %q: %w", dataDir, err)
	}
	if err := ValidateNamespace(opts.Namespace); err != nil {
		return Destination{}, err
	}
	dest := BuildDestination(dataDir, manifest, opts.Namespace)
	if !opts.Force {
		if err := ValidateDestinationConflict(dest.Path); err != nil {
			return Destination{}, err
		}
	}
	return dest, nil
}

// prepareDestination removes an existing install when force is set.
// A missing destination is not an error.
func prepareDestination(path string, force bool) error {
	if !force {
		return nil
	}
	if err := RemoveTarget(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing existing package: %w", err)
	}
	return nil
}

// performInstall dispatches to the appropriate install mode.
func performInstall(sourceDir string, dest Destination, opts *InstallOptions) error {
	if opts.Editable {
		return installEditable(sourceDir, dest)
	}
	return installCopy(sourceDir, dest)
}

func installEditable(sourceDir string, dest Destination) error {
	if err := SymlinkPackage(sourceDir, dest.Path); err != nil {
		return err
	}
	internal.PrintInfof("installed %s (editable)", internal.FormatImportStmt(dest.Namespace, dest.Name, dest.Version))
	return nil
}

func installCopy(sourceDir string, dest Destination) error {
	if err := CopyPackageFiles(sourceDir, dest.Path); err != nil {
		return err
	}
	internal.PrintInfof("installed %s", internal.FormatImportStmt(dest.Namespace, dest.Name, dest.Version))
	return nil
}

// SymlinkPackage creates a symlink at dest pointing to the absolute path of src.
// The parent directory of dest is created if it does not exist.
func SymlinkPackage(src, dest string) error {
	if err := internal.EnsureDir(filepath.Dir(dest)); err != nil {
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

func CopyPackageFiles(src, dest string) error {
	matcher := BuildIgnoreMatcher(src)
	jobs, err := CollectJobs(src, dest, matcher)
	if err != nil {
		return err
	}
	return runTransferJobsWithSpinner(jobs)
}

func runTransferJobsWithSpinner(jobs []TransferJob) error {
	spinner := internal.SetupSpinner()
	spinner.Start()
	defer spinner.Stop()
	return RunTransferJobs(jobs)
}

func RunTransferJobs(jobs []TransferJob) error {
	n := len(jobs)
	errCh := make(chan error, n)

	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Go(func() {
			if err := CopyFile(job.Src, job.Dst); err != nil {
				errCh <- err
				return
			}
		})
	}
	wg.Wait()
	close(errCh)
	return collectErrors(errCh)
}

func CopyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), internal.DirPerm); err != nil {
		return fmt.Errorf("creating parent directories for %q: %w", dest, err)
	}

	srcFile, err := os.Open(src) //nolint: gosec
	if err != nil {
		return fmt.Errorf("opening source file %q: %w", src, err)
	}
	defer srcFile.Close() //nolint: errcheck

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("reading file info %q: %w", src, err)
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()) //nolint: gosec
	if err != nil {
		return fmt.Errorf("creating destination file %q: %w", dest, err)
	}
	defer destFile.Close() //nolint: errcheck

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

func BuildIgnoreMatcher(dir string) *ignore.GitIgnore {
	gitIgnorePath := filepath.Join(dir, ".gitignore")
	typstIgnorePath := filepath.Join(dir, ".typstignore")
	extraLines := ReadIgnoreLines(typstIgnorePath)
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

type TransferJob struct {
	Src string
	Dst string
}

func CollectJobs(src, dest string, matcher *ignore.GitIgnore) ([]TransferJob, error) {
	var jobs []TransferJob
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walking %q: %w", path, walkErr)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolving relative path %q: %w", path, err)
		}
		if ShouldIgnore(rel, matcher) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			jobs = append(jobs, TransferJob{
				Src: path,
				Dst: filepath.Join(dest, rel),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not collect all files to transfer: %w", err)
	}
	return jobs, nil
}

func ShouldIgnore(rel string, matcher *ignore.GitIgnore) bool {
	if rel == "." {
		return false
	}
	if _, ok := IgnoredFileNames[filepath.Base(rel)]; ok {
		return true
	}
	if matcher != nil && matcher.MatchesPath(rel) {
		return true
	}
	return false
}

var IgnoredFileNames = map[string]struct{}{
	".git":         {},
	".gitignore":   {},
	".typstignore": {},
}

func ReadIgnoreLines(path string) []string {
	data, err := os.ReadFile(path) //nolint: gosec
	if err != nil {
		return nil
	}
	var lines []string
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
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

func BuildDestination(dataDir string, manifest internal.Manifest, namespace string) Destination {
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

func ResolveSourceDir(args []string, remoteURL string) (string, error) {
	if remoteURL != "" {
		return CloneRepoIntoDataDir(remoteURL)
	}
	return ResolveLocalSourceDir(args)
}

func CloneRepoIntoDataDir(url string) (string, error) {
	dataDir, err := internal.ResolveDataDir()
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	repoName, err := remote.RepoNameFromURL(url)
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	appDataDir := filepath.Join(dataDir, "gotpm", "remotes", repoName)

	if isDir := internal.IsDir(appDataDir); isDir {
		return appDataDir, nil
	}

	internal.PrintInfof("Cloning %q", url)
	err = remote.CloneRepo(url, appDataDir)
	if err != nil {
		return "", err //nolint: wrapcheck
	}
	return appDataDir, nil
}

func ResolveLocalSourceDir(args []string) (string, error) {
	numberOfArgs := len(args)
	maxArguments := 1
	switch {
	case numberOfArgs > maxArguments:
		return "", ErrTooManyArguments
	case numberOfArgs == maxArguments:
		return ResolveProvidedPath(args[0])
	case numberOfArgs == 0:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current working directory: %w", err)
		}
		return cwd, nil
	default:
		return "", ErrTooManyArguments
	}
}

func ResolveProvidedPath(rawPath string) (string, error) {
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", rawPath, err)
	}
	if err := ValidateIsDirectory(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func ValidateDestinationConflict(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("%w: %q", ErrPackageAlreadyInstalled, path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("validate destination %q: %w", path, err)
}

func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return ErrEmptyNamespace
	}
	return nil
}

func ValidateIsDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist %q: %w", path, err)
		}
		return fmt.Errorf("accessing path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory %q: %w", path, ErrNotADirectory)
	}
	return nil
}
