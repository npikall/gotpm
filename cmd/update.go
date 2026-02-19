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
	"sync"

	"github.com/charmbracelet/log"
	"github.com/npikall/gotpm/internal/request"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use: "update [file]",
	Example: `# update import statements
gotpm update foo.typ

# defaults to 'dependencies.typ'
gotpm update`,
	Short: "Update all dependencies from a file to their latest version.",
	Long:  "Update all dependencies from a file to their latest version.",
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
	targetFilePath = getAbsolutePath(args, targetFilePath, cwd)
	logger.Debug("update", "target", targetFilePath)

	if _, err := os.Stat(targetFilePath); errors.Is(err, fs.ErrNotExist) {
		return err
	}

	targetFileContent, err := os.ReadFile(targetFilePath)
	if err != nil {
		return err
	}

	foundImports := extractImportStatements(targetFileContent)

	var wg sync.WaitGroup
	resultCh := make(chan result, len(foundImports))
	logCh := make(chan logEvent, len(foundImports))
	for _, importStatement := range foundImports {
		wg.Go(func() {
			pkgName, pkgVersion := getPackageInfos(importStatement)

			apiURL, err := url.JoinPath(request.TypstPackageEndpoint, pkgName)
			if err != nil {
				logCh <- logEvent{"error", err.Error(), nil}
				return
			}

			response, err := request.FetchDataFromGitHub(apiURL, ctx)
			if err != nil {
				logCh <- logEvent{"error", err.Error(), nil}
				return
			}

			latestVersion, err := request.GetLatestVersion(response)
			if err != nil {
				logCh <- logEvent{"error", err.Error(), nil}
				return
			}

			if latestVersion == pkgVersion {
				logCh <- logEvent{"debug", "already at latest", []any{"package", pkgName}}
				return
			}

			logCh <- logEvent{"info", "update", []any{"package", pkgName, "from", pkgVersion, "to", latestVersion}}
			resultCh <- result{name: pkgName, latest: latestVersion}
		})
	}
	wg.Wait()
	close(resultCh)
	close(logCh)

	for event := range logCh {
		logLogEvent(event, logger)
	}

	var newVersions = make(map[string]string)
	for r := range resultCh {
		newVersions[r.name] = r.latest
	}

	UpdateFileContent(&targetFileContent, newVersions)
	err = os.WriteFile(targetFilePath, targetFileContent, 0644)
	if err != nil {
		return err
	}

	return nil
}

func logLogEvent(l logEvent, logger *log.Logger) {
	switch l.level {
	case "debug":
		logger.Debug(l.msg, l.keyvals...)
	case "info":
		logger.Info(l.msg, l.keyvals...)
	case "error":
		logger.Error(l.msg, l.keyvals...)
	}
}

func getPackageInfos(importStatement []byte) (string, string) {
	pkgNameVersion := strings.Split(string(importStatement), "/")[1]
	pkgInfo := strings.Split(pkgNameVersion, ":")
	pkgName := pkgInfo[0]
	pkgVersion := pkgInfo[1]
	return pkgName, pkgVersion
}

func extractImportStatements(targetFile []byte) [][]byte {
	pattern := regexp.MustCompile(`@preview/[a-zA-Z-]*:[0-9]*.[0-9]*.[0-9]*`)
	foundImports := pattern.FindAll(targetFile, -1)
	return foundImports
}

func getAbsolutePath(args []string, targetFilePath string, cwd string) string {
	if len(args) > 0 {
		targetFilePath = filepath.Join(cwd, args[0])
	} else {
		targetFilePath = filepath.Join(cwd, "dependencies.typ")
	}
	return targetFilePath
}

type result struct {
	name   string
	latest string
}
type logEvent struct {
	level   string
	msg     string
	keyvals []any
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
