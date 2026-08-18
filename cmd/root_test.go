/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
// The command variables are package-private, so the flag surface has to be
// inspected from inside the package.
package cmd //nolint: testpackage

import (
	"testing"

	"github.com/npikall/gotpm/internal/paths"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestInstallDirIsRejectedByTheProjectCommands pins where an install dir may be
// named. It receives one package's files directly, so the commands that install
// or delete a whole dependency graph must not accept it.
func TestInstallDirIsRejectedByTheProjectCommands(t *testing.T) {
	t.Parallel()
	for _, cmd := range []*cobra.Command{addCmd, syncCmd, removeCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			t.Parallel()
			err := cmd.ParseFlags([]string{"--" + paths.InstallDirFlag, "./dist"})
			require.ErrorContains(t, err, "unknown flag")
		})
	}
}

// TestInstallDirIsAcceptedByTheSinglePackageCommands is the other half: the
// commands that operate on one package keep the flag.
func TestInstallDirIsAcceptedByTheSinglePackageCommands(t *testing.T) {
	t.Parallel()
	for _, cmd := range []*cobra.Command{installCmd, uninstallCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, cmd.ParseFlags([]string{"--" + paths.InstallDirFlag, "./dist"}))
		})
	}
}
