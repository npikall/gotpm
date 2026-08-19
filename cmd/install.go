package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/install"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install a Typst Package locally.",
	Long: `All files that are not specifically excluded get copied into the package
directory, $DATA_DIR/typst/packages, where the $DATA_DIR is dependent on the
machine's operating system, under namespace/name/version. Set
$TYPST_PACKAGE_PATH to put that package directory somewhere else: the layout
inside it is unchanged, and typst imports from it the same way.

--install-dir, and the GOTPM_INSTALL_DIR environment variable behind it, are a
different thing. The directory named there receives the package's files
directly, with no namespace/name/version layout around them, which is not a
directory typst imports from and not one gotpm scans. It is an output
destination — vendoring one package into a build directory, or looking at what
an install would produce. The flag takes precedence over the environment
variable. Neither reaches 'add', 'sync' or 'remove': those work on a dependency
graph, which does not fit in a directory holding one package.
`,
	Example: `gotpm install
gotpm install . -e
gotpm install -n preview
gotpm install -r github.com/user/repo -t v0.1.2
gotpm install path/to/package -n preview
`,
	Args: cobra.MaximumNArgs(1),
	RunE: InstallRunner,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("namespace", "n", paths.DefaultNamespace, "The namespace in which the package should be available.")
	installCmd.Flags().BoolP("editable", "e", false, "Create a symlink to the source directory instead of copying files.")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite an already-installed package.")
	installCmd.Flags().String(paths.InstallDirFlag, "", installDirUsage)
	installCmd.Flags().StringP("remote", "r", "", "The remote repository which should be installed.")
	installCmd.Flags().StringP("rev", "t", "HEAD", "The revision (hash or tag) that should be checked out.")
}

func InstallRunner(cmd *cobra.Command, args []string) error {
	opts := &install.Options{
		Force:      Must(cmd.Flags().GetBool("force")),
		Editable:   Must(cmd.Flags().GetBool("editable")),
		Namespace:  Must(cmd.Flags().GetString("namespace")),
		Remote:     Must(cmd.Flags().GetString("remote")),
		InstallDir: Must(cmd.Flags().GetString(paths.InstallDirFlag)),
		Revision:   Must(cmd.Flags().GetString("rev")),
	}

	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	return install.Run(path, opts, newLogger(cmd))
}
