/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
)

var GoTPMVersion string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display application's version information",
	Long: `
The version command provides information about the application's version.

GoTPM requires version information to be embedded at compile time.
For detailed version information, Go Blueprint needs to be built as specified in the README installation instructions.
If Go Blueprint is built within a version control repository and other version info isn't available,
the revision hash will be used instead.
`,
	Run: func(cmd *cobra.Command, args []string) {
		version := getGoTPMVersion()
		fmt.Printf("GoTPM CLI version: %v\n", InfoStyle.Render(version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func getGoTPMVersion() string {
	noVersionAvailable := "No version info available for this build, run 'gotpm help version' for additional info"

	if GoTPMVersion != "" {
		return GoTPMVersion
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return noVersionAvailable
	}

	// If no main version is available, Go defaults it to (devel)
	if bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}

	var vcsRevision string
	var vcsTime time.Time
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsRevision = setting.Value
		case "vcs.time":
			vcsTime, _ = time.Parse(time.RFC3339, setting.Value)
		}
	}

	if vcsRevision != "" {
		return fmt.Sprintf("%s, (%s)", vcsRevision, vcsTime)
	}

	return noVersionAvailable
}
