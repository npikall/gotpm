/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/update"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use: "update [file|directory]",
	Example: `# update import statements in a file (writes back in place)
gotpm update foo.typ

# update all .typ files in a directory (writes back in place)
gotpm update src/

# recursively update all .typ files in a directory
gotpm update src/ -r

# update files with custom extensions
gotpm update src/ --ext .typ --ext .md

# pipe content via stdin, write result to stdout
cat foo.typ | gotpm update

# pipe content via stdin, write result to a file
cat foo.typ | gotpm update -o foo.typ

# read from a file, write result to a different file
gotpm update foo.typ -o bar.typ`,
	Short: "Update all dependencies from a file or directory to their latest version.",
	Long:  "Update all dependencies from a file or directory to their latest version.",
	RunE:  UpdateRunner,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("output", "o", "", "Output file (defaults to input file, or stdout when reading from stdin)")
	updateCmd.Flags().Bool("no-cache", false, "Skip reading and writing the package index cache")
	updateCmd.Flags().BoolP("recursive", "r", false, "Process recursively (only applies when input is a directory)")
	updateCmd.Flags().StringSlice("ext", []string{".typ"}, "File extensions to process when input is a directory")
}

func UpdateRunner(cmd *cobra.Command, args []string) error {
	opts := &update.Options{
		Output:     Must(cmd.Flags().GetString("output")),
		NoCache:    Must(cmd.Flags().GetBool("no-cache")),
		Recursive:  Must(cmd.Flags().GetBool("recursive")),
		Extensions: Must(cmd.Flags().GetStringSlice("ext")),
	}
	return update.Run(args, opts, newLogger(cmd))
}
