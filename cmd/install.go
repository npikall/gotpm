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

	"github.com/go-git/go-git/v6"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/remote"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/ui"
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
	Example: `gotpm install
gotpm install . -e
gotpm install -n preview
gotpm install -r github.com/user/repo -t v0.1.2
gotpm install path/to/package -n preview
`,
	RunE: InstallRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "Create a symlink to the source directory instead of copying files.")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite an already-installed package.")
	installCmd.Flags().String(paths.InstallDirFlag, "", "Override the package directory (env: $"+paths.InstallDirEnvVar+")")
	installCmd.Flags().StringP("remote", "r", "", "The remote repository which should be installed.")
	installCmd.Flags().StringP("rev", "t", "HEAD", "The revision (hash or tag) that should be checked out.")
}

func InstallRunner(cmd *cobra.Command, args []string) error {
	logger := newLogger(cmd)
	opts := ReadInstallOptions(cmd)

	sourceDir, err := ResolveSourceDir(args, opts)
	if err != nil {
		return err
	}
	logger.Debug("operating in source", "path", sourceDir)

	manifestFile, err := manifest.FindFile(sourceDir)
	if err != nil {
		return fmt.Errorf("could not load manifest: %w", err)
	}
	m, err := manifest.LoadFile(manifestFile)
	if err != nil {
		return fmt.Errorf("could not load manifest: %w", err)
	}
	sourceDir = filepath.Dir(manifestFile)
	logger.Debug("found package", "name", m.Package.Name, "version", m.Package.Version, "root", sourceDir)

	s, err := store.Open(opts.InstallDir)
	if err != nil {
		return err
	}
	ref, err := pkg.New(opts.Namespace, m.Package.Name, m.Package.Version)
	if err != nil {
		return err
	}
	logger.Debug("resolved destination", "path", s.Dir(ref))

	if err := prepareDestination(s, ref, opts.Force); err != nil {
		return err
	}

	return performInstall(s, ref, sourceDir, opts)
}

var (
	ErrTooManyArguments = errors.New("too many arguments: expected one directory path")
	ErrNotADirectory    = errors.New("path is not a directory")
)

// InstallOptions holds the resolved install flags.
type InstallOptions struct {
	Force      bool
	Editable   bool
	Namespace  string
	Remote     string
	InstallDir string
	Revision   string
}

func ReadInstallOptions(cmd *cobra.Command) *InstallOptions {
	rev := Must(cmd.Flags().GetString("rev"))
	force := Must(cmd.Flags().GetBool("force"))
	remote := Must(cmd.Flags().GetString("remote"))
	editable := Must(cmd.Flags().GetBool("editable"))
	namespace := Must(cmd.Flags().GetString("namespace"))
	installDir := Must(cmd.Flags().GetString(paths.InstallDirFlag))
	return &InstallOptions{
		Force:      force,
		Editable:   editable,
		Namespace:  namespace,
		Remote:     remote,
		InstallDir: installDir,
		Revision:   rev,
	}
}

// prepareDestination clears an existing install when force is set, and rejects
// the install otherwise.
func prepareDestination(s store.Store, ref pkg.Ref, force bool) error {
	if !s.Has(ref) {
		return nil
	}
	if !force {
		return fmt.Errorf("%w: %q", store.ErrAlreadyInstalled, s.Dir(ref))
	}
	if err := s.Remove(ref); err != nil {
		return fmt.Errorf("removing existing package: %w", err)
	}
	return nil
}

// performInstall dispatches to the appropriate install mode.
func performInstall(s store.Store, ref pkg.Ref, sourceDir string, opts *InstallOptions) error {
	if opts.Editable {
		if err := s.Link(ref, sourceDir); err != nil {
			return err
		}
		ui.Infof("installed %s (editable)", ui.AccentBold.Render(ref.String()))
		return nil
	}

	spin := ui.Spinner("")
	spin.Start()
	err := s.Install(ref, sourceDir)
	spin.Stop()
	if err != nil {
		return err
	}
	ui.Infof("installed %s", ui.AccentBold.Render(ref.String()))
	return nil
}

func ResolveSourceDir(args []string, opts *InstallOptions) (string, error) {
	if opts.Remote != "" {
		return CloneRepoIntoDataDir(opts)
	}
	return ResolveLocalSourceDir(args)
}

func CloneRepoIntoDataDir(opts *InstallOptions) (string, error) {
	url := opts.Remote
	remotesDir, err := remote.CacheDir()
	if err != nil {
		return "", err
	}
	repoName, err := remote.RepoNameFromURL(url)
	if err != nil {
		return "", err
	}
	repoDir := filepath.Join(remotesDir, repoName)

	isDir := paths.IsDir(repoDir)
	if isDir && opts.Revision != "" {
		repo, err := git.PlainOpen(repoDir)
		if err != nil {
			return "", err //nolint: wrapcheck
		}
		_ = repo.Fetch(&git.FetchOptions{})
		if err = remote.CheckoutRevision(repo, opts.Revision); err != nil {
			return "", err
		}
		return repoDir, nil
	}
	if isDir {
		return repoDir, nil
	}

	ui.Infof("Cloning %q", url)
	cleanedURL := remote.DefaultHTTPCloneURL(url)
	err = remote.CloneRepo(cleanedURL, repoDir, opts.Revision)
	if err != nil {
		return "", err
	}
	return repoDir, nil
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
