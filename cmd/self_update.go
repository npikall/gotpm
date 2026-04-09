/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gotpm to the latest version from GitHub Releases",
	Long:  "Download and install the latest gotpm release from GitHub, replacing the current binary in place.",
	RunE:  selfUpdateRunner,
}

func init() {
	selfCmd.AddCommand(selfUpdateCmd)
	selfUpdateCmd.Flags().Bool("check", false, "check for an update without installing it")
}

func selfUpdateRunner(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	ctx := context.Background()

	if gitTag == "dev" {
		return fmt.Errorf("cannot self-update a development build; install a tagged release first")
	}

	currentVersion := strings.TrimPrefix(gitTag, "v")

	filter := fmt.Sprintf("gotpm-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		filter += ".exe"
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Filters: []string{filter},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	s := internal.SetupSpinner()
	s.Suffix = internal.StyleMuted.Render(" Checking for updates...")
	s.Start()
	release, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug("npikall/gotpm"))
	s.Stop()

	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !found {
		internal.PrintWarn("no release found for %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}

	latestVersion := release.Version()
	if latestVersion == currentVersion {
		internal.PrintInfo("already up to date (%s)", gitTag)
		return nil
	}

	if checkOnly {
		internal.PrintInfo("update available: %s → %s",
			internal.StyleAccent.Render(gitTag),
			internal.StyleAccent.Render("v"+latestVersion))
		return nil
	}

	s.Suffix = internal.StyleMuted.Render(" Downloading update...")
	s.Start()
	_, err = updater.UpdateSelf(ctx, currentVersion, selfupdate.ParseSlug("npikall/gotpm"))
	s.Stop()

	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	internal.PrintInfo("updated gotpm %s → %s",
		internal.StyleAccent.Render(gitTag),
		internal.StyleAccent.Render("v"+latestVersion))
	return nil
}
