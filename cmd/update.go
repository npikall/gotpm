/*
Copyright © 2026 Nikolas Pikall <nikolas.pikall@gmail.com>

SPDX-License-Identifier: MIT License
See the LICENSE file in the repository root for full license text.
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/npikall/gotpm/cmd/internal"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use: "update [file]",
	Example: `# update import statements in a file (writes back in place)
gotpm update foo.typ

# pipe content via stdin, write result to stdout
cat foo.typ | gotpm update

# pipe content via stdin, write result to a file
cat foo.typ | gotpm update -o foo.typ

# read from a file, write result to a different file
gotpm update foo.typ -o bar.typ`,
	Short: "Update all dependencies from a file to their latest version.",
	Long:  "Update all dependencies from a file to their latest version.",
	RunE:  updateRunner,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("output", "o", "", "Output file (defaults to input file, or stdout when reading from stdin)")
}

type logEvent struct {
	level   string
	msg     string
	keyvals []any
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

func updateRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)
	ctx := context.Background()
	outputPath, _ := cmd.Flags().GetString("output")

	content, inputFilePath, err := readInputContent(args)
	if err != nil {
		return err
	}

	imports := extractImportStatements(content)

	s := internal.SetupSpinner()
	s.Start()
	newVersions, logEvents := fetchLatestVersionsConcurrently(ctx, imports)
	s.Stop()

	for _, event := range logEvents {
		logLogEvent(event, logger)
	}

	if len(newVersions) == 0 {
		logger.Info("all dependencies are up to date")
	}

	UpdateFileContent(&content, newVersions)
	return writeOutputContent(content, inputFilePath, outputPath)
}

func readInputContent(args []string) (content []byte, inputFilePath string, err error) {
	if len(args) > 0 {
		content, err = os.ReadFile(args[0])
		return content, args[0], err
	}
	if isStdinPiped() {
		content, err = io.ReadAll(os.Stdin)
		return content, "", err
	}
	return nil, "", fmt.Errorf("no input: provide a file argument or pipe content via stdin")
}

func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func writeOutputContent(content []byte, inputFilePath string, outputPath string) error {
	if outputPath != "" {
		return os.WriteFile(outputPath, content, 0644)
	}
	if inputFilePath != "" {
		return os.WriteFile(inputFilePath, content, 0644)
	}
	_, err := os.Stdout.Write(content)
	return err
}

func fetchLatestVersionsConcurrently(ctx context.Context, imports [][]byte) (map[string]string, []logEvent) {
	index, _ := fetchTypstVersionIndex(ctx)

	resultCh := make(chan result, len(imports))
	logCh := make(chan logEvent, len(imports))

	var wg sync.WaitGroup
	for _, importStatement := range imports {
		wg.Go(func() {
			processImport(ctx, importStatement, index, resultCh, logCh)
		})
	}
	wg.Wait()
	close(resultCh)
	close(logCh)

	return collectVersionResults(resultCh), collectLogEvents(logCh)
}

// lookupVersion queries the Typst package index first and falls back to the
// GitHub API for missing packages or when the index is unavailable.
// Returns the version, its source ("index" or "github"), and any error.
func lookupVersion(ctx context.Context, index map[string]string, pkgName string) (string, string, error) {
	if version, ok := index[pkgName]; ok {
		return version, "index", nil
	}
	version, err := lookupVersionFromGitHub(ctx, pkgName)
	return version, "github", err
}

func fetchTypstVersionIndex(ctx context.Context) (map[string]string, error) {
	entries, err := internal.FetchTypstIndex(ctx)
	if err != nil {
		return nil, err
	}
	return internal.BuildVersionIndex(entries), nil
}

func processImport(ctx context.Context, importStatement []byte, index map[string]string, resultCh chan<- result, logCh chan<- logEvent) {
	pkgName, pkgVersion := parsePackageRef(importStatement)

	latestVersion, source, err := lookupVersion(ctx, index, pkgName)
	if err != nil {
		logCh <- logEvent{"error", err.Error(), nil}
		return
	}

	if latestVersion == pkgVersion {
		logCh <- logEvent{"debug", "already at latest", []any{"package", pkgName}}
		return
	}

	logCh <- logEvent{"info", "update", []any{"package", pkgName, "from", pkgVersion, "to", latestVersion, "via", source}}
	resultCh <- result{name: pkgName, latest: latestVersion}
}

func lookupVersionFromGitHub(ctx context.Context, pkgName string) (string, error) {
	apiURL, err := url.JoinPath(internal.TypstPackageEndpoint, pkgName)
	if err != nil {
		return "", err
	}
	response, err := internal.FetchDataFromGitHub(apiURL, ctx)
	if err != nil {
		return "", err
	}
	return internal.GetLatestVersion(response)
}

func collectVersionResults(resultCh <-chan result) map[string]string {
	versions := make(map[string]string)
	for r := range resultCh {
		versions[r.name] = r.latest
	}
	return versions
}

func collectLogEvents(logCh <-chan logEvent) []logEvent {
	var events []logEvent
	for event := range logCh {
		events = append(events, event)
	}
	return events
}

func parsePackageRef(importStatement []byte) (name, version string) {
	pkgNameVersion := strings.Split(string(importStatement), "/")[1]
	pkgInfo := strings.Split(pkgNameVersion, ":")
	return pkgInfo[0], pkgInfo[1]
}

func extractImportStatements(targetFile []byte) [][]byte {
	pattern := regexp.MustCompile(`@preview/[a-zA-Z-]*:[0-9]*.[0-9]*.[0-9]*`)
	return pattern.FindAll(targetFile, -1)
}

type result struct {
	name   string
	latest string
}

// UpdateFileContent updates all typst package import statements in content
// with the versions provided by the name→version mapping.
func UpdateFileContent(content *[]byte, versions map[string]string) {
	for key, value := range versions {
		rawPattern := fmt.Sprintf(`@preview/%s:[0-9]*.[0-9]*.[0-9]*`, key)
		pattern := regexp.MustCompile(rawPattern)

		namespacePkg := strings.Split(rawPattern, ":")[0]
		replacement := fmt.Sprintf("%s:%s", namespacePkg, value)

		*content = pattern.ReplaceAll(*content, []byte(replacement))
	}
}
