package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/font"
	"github.com/spf13/cobra"
)

// fontCmd represents the font command
var fontCmd = &cobra.Command{
	Use:   "font",
	Short: "Manage fonts for typst projects.",
	Long: `Download and install fonts, and make typst projects reproducible.

Fonts come from the Google Fonts repository (github.com/google/fonts) and are
kept in gotpm's font directory, one directory per family. Typst does not look
there on its own; point $TYPST_FONT_PATHS at it, e.g. in ~/.bashrc:

  export TYPST_FONT_PATHS="$(gotpm locate fonts)"

Names are matched the way the repository lays out its directories, ignoring
case, spaces and punctuation, so "Open Sans" and "opensans" are the same font.
Quote names that contain spaces.`,
}

var fontInstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install a font family into the font directory.",
	Long: `Install the newest version of a font family into the font directory,
without recording it in any project. Installing a family again replaces it.

A family directory gotpm did not install is left alone unless --force is
given.`,
	Example: `gotpm font install Roboto
gotpm font install "Open Sans"`,
	Args: cobra.ExactArgs(1),
	RunE: FontInstallRunner,
}

var fontUninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: "Delete a font family from the font directory.",
	Long: `Delete a font family from the font directory. Projects that declare it
get it back with 'gotpm sync'.

A family directory gotpm did not install is left alone unless --force is
given.`,
	Example: `gotpm font uninstall "Open Sans"`,
	Args:    cobra.ExactArgs(1),
	RunE:    FontUninstallRunner,
}

var fontSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the font families Google Fonts offers.",
	Long: `List the font families whose name contains the query, with the license
each is published under. Without a query, every family is listed.

The list is fetched from GitHub and cached for a day; --refresh fetches it
again. Set $GITHUB_TOKEN to lift GitHub's rate limit.`,
	Example: `gotpm font search "open sans"
gotpm font search mono --refresh`,
	Args: cobra.MaximumNArgs(1),
	RunE: FontSearchRunner,
}

var fontAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a font family to this project.",
	Long: `The font family is installed into the font directory and recorded in two
files next to typst.toml:

  typst.toml   gains the family under [tool.gotpm].fonts
  gotpm.lock   pins it to the exact commit it was fetched from

Commit both: 'gotpm sync' then installs the same files on any machine. Adding a
family again pins its newest commit.`,
	Example: `gotpm font add Roboto
gotpm font add "Open Sans"`,
	Args: cobra.ExactArgs(1),
	RunE: FontAddRunner,
}

var fontRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a font family from this project.",
	Long: `The family is dropped from typst.toml and gotpm.lock. Its files stay in
the font directory, because other projects on this machine may use it;
'gotpm font uninstall' deletes them.`,
	Example: `gotpm font remove "Open Sans"`,
	Args:    cobra.ExactArgs(1),
	RunE:    FontRemoveRunner,
}

func init() {
	rootCmd.AddCommand(fontCmd)
	fontCmd.AddCommand(fontInstallCmd, fontUninstallCmd, fontSearchCmd, fontAddCmd, fontRemoveCmd)

	for _, c := range []*cobra.Command{fontInstallCmd, fontUninstallCmd, fontAddCmd} {
		c.Flags().BoolP("force", "f", false, "Replace or delete a font family gotpm did not install.")
	}
	fontSearchCmd.Flags().Bool("refresh", false, "Fetch the family list even when the cached one is fresh.")
}

func fontOptions(cmd *cobra.Command) *font.Options {
	return &font.Options{Force: Must(cmd.Flags().GetBool("force"))}
}

func FontInstallRunner(cmd *cobra.Command, args []string) error {
	return font.Install(cmd.Context(), args[0], fontOptions(cmd), newLogger(cmd))
}

func FontUninstallRunner(cmd *cobra.Command, args []string) error {
	return font.Uninstall(args[0], fontOptions(cmd), newLogger(cmd))
}

func FontSearchRunner(cmd *cobra.Command, args []string) error {
	return font.Search(cmd.Context(), firstArg(args), Must(cmd.Flags().GetBool("refresh")), newLogger(cmd))
}

func FontAddRunner(cmd *cobra.Command, args []string) error {
	return font.Add(cmd.Context(), args[0], fontOptions(cmd), newLogger(cmd))
}

func FontRemoveRunner(cmd *cobra.Command, args []string) error {
	return font.Remove(args[0], newLogger(cmd))
}
