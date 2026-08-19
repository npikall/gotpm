package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/uninstall"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a Typst Package from the local Storage",
	Long: `Removes a locally installed Typst package from the package directory.

Naming a namespace and nothing else removes the whole namespace, after asking
for confirmation. Adding a package, a version or --all narrows the removal back
to a package inside that namespace.

Which package directory is read follows $TYPST_PACKAGE_PATH, keeping the
namespace/name/version layout inside it.

--install-dir, and the GOTPM_INSTALL_DIR environment variable behind it, point
somewhere else entirely: at a directory holding one package's files directly,
without that layout. The flag takes precedence over the environment variable.
A namespace cannot be removed from such a directory, since there is no
namespace layout in it to remove.
`,
	Example: `# get package metadata from typst.toml
gotpm uninstall
gotpm uninstall foo

# uninstall specific package from 'local' or 'preview'
gotpm uninstall foo -V 0.1.2
gotpm uninstall foo -V 0.1.2 -n preview

# all versions of foo in namespace 'local' or 'preview'
gotpm uninstall foo --all
gotpm uninstall foo -n preview --all

# the whole 'preview' namespace, with and without the prompt
gotpm uninstall -n preview
gotpm uninstall -n preview --yes`,
	Args: cobra.MaximumNArgs(1),
	RunE: UninstallRunner,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "The namespace from which the package should be removed from. On its own, removes the whole namespace.")
	uninstallCmd.Flags().StringP("version", "V", "", "The specific version of a package that should be removed.")
	uninstallCmd.Flags().Bool("all", false, "Uninstall all Packages from a given namespace or all versions of a package.")
	uninstallCmd.Flags().Bool("dry-run", false, "Perform a dry run.")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt when removing a namespace.")
	uninstallCmd.Flags().String(paths.InstallDirFlag, "", installDirUsage)
}

func UninstallRunner(cmd *cobra.Command, args []string) error {
	opts := &uninstall.Options{
		Namespace:    Must(cmd.Flags().GetString("namespace")),
		NamespaceSet: cmd.Flags().Changed("namespace"),
		Version:      Must(cmd.Flags().GetString("version")),
		All:          Must(cmd.Flags().GetBool("all")),
		DryRun:       Must(cmd.Flags().GetBool("dry-run")),
		Yes:          Must(cmd.Flags().GetBool("yes")),
		InstallDir:   Must(cmd.Flags().GetString(paths.InstallDirFlag)),
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	return uninstall.Run(name, opts, newLogger(cmd))
}
