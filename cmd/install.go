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
graph, which does not fit in a directory holding one package. -r/--remote
installs a graph too, once the repository has dependencies of its own, so
--install-dir is refused there unless it turns out to be single-package.

-r/--remote fetches a repository instead of using the working tree, and
installs everything it depends on alongside it, the same way 'add' does — a
dependency with no gotpm.lock at all is skipped with a warning, but one whose
lock is simply missing an entry still fails the install. Without -t/--rev the
newest release tag is used, or the current HEAD when the repository has none;
pass -t HEAD explicitly to keep pinning HEAD regardless of releases.
`,
	Example: `gotpm install
gotpm install . -e
gotpm install -n preview
gotpm install -r github.com/user/repo -t v0.1.2
gotpm install -r github.com/user/repo -t HEAD
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
	installCmd.Flags().StringP("rev", "t", "", "The revision (hash or tag) to check out. Defaults to the newest release.")
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
