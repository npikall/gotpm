/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/npikall/gotpm/internal/request"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [file]",
	Short: "Update all dependencies from a file to their latest version.",
	RunE:  updateRunner,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolP("debug", "d", false, "Print Debug Level Information")
}

func updateRunner(cmd *cobra.Command, args []string) error {
	logger := setupVerboseLogger(cmd)
	cwd := Must(os.Getwd())
	ctx := context.Background()

	var targetFilePath string
	if len(args) > 0 {
		targetFilePath = filepath.Join(cwd, args[0])
	} else {
		targetFilePath = filepath.Join(cwd, "dependencies.typ")
	}
	logger.Debug("update", "target", targetFilePath)

	if _, err := os.Stat(targetFilePath); errors.Is(err, fs.ErrNotExist) {
		return err
	}

	targetFile, err := os.ReadFile(targetFilePath)
	if err != nil {
		return err
	}

	pattern := regexp.MustCompile(`@preview/[a-zA-Z-]*:[0-9]*.[0-9]*.[0-9]*`)
	foundImports := pattern.FindAll(targetFile, -1)

	// TODO: add spinner
	var newVersions = make(map[string]string)
	for _, importStatement := range foundImports {
		pkgNameVersion := strings.Split(string(importStatement), "/")[1]
		logger.Debug("found", "package", pkgNameVersion)
		pkgName := strings.Split(pkgNameVersion, ":")[0]
		apiURL, err := url.JoinPath(request.TypstPackageEndpoint, pkgName)
		if err != nil {
			return err
		}
		logger.Debug("fetching", "url", apiURL)
		response, err := request.FetchDataFromGitHub(apiURL, ctx)
		if err != nil {
			return err
		}
		latestVersion, err := request.GetLatestVersion(response)
		if err != nil {
			return err
		}
		newVersions[pkgName] = latestVersion
		logger.Info("update to", "package", pkgName, "latest", latestVersion)
	}

	UpdateFileContent(&targetFile, newVersions)
	err = os.WriteFile(targetFilePath, targetFile, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Update all typst package import statements in a file, with the values provided
// by a mapping of package names to the latest version.
func UpdateFileContent(content *[]byte, versions map[string]string) {
	for key, value := range versions {
		rawPattern := fmt.Sprintf(`@preview/%s:[0-9]*.[0-9]*.[0-9]*`, key)
		pattern := regexp.MustCompile(rawPattern)

		namespacePkg := strings.Split(rawPattern, ":")[0]
		replacement := fmt.Sprintf("%s:%s", namespacePkg, value)

		*content = pattern.ReplaceAll(*content, []byte(replacement))
	}
}
