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
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/cmd/internal"
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
	RunE:  updateRunner,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("output", "o", "", "Output file (defaults to input file, or stdout when reading from stdin)")
	updateCmd.Flags().Bool("no-cache", false, "Skip reading and writing the package index cache")
	updateCmd.Flags().BoolP("recursive", "r", false, "Recursively process subdirectories (only applies when input is a directory)")
	updateCmd.Flags().StringSlice("ext", []string{".typ"}, "File extensions to process when input is a directory")
}

type logEvent struct {
	level   string
	msg     string
	keyvals []any
}

func emitLogEvent(e logEvent, logger *log.Logger) {
	switch e.level {
	case "debug":
		logger.Debug(e.msg, e.keyvals...)
	case "info":
		logger.Info(e.msg, e.keyvals...)
	case "error":
		logger.Error(e.msg, e.keyvals...)
	}
}

func updateRunner(cmd *cobra.Command, args []string) error {
	logger := internal.SetupLogger(cmd)
	ctx := context.Background()
	outputPath, _ := cmd.Flags().GetString("output")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	recursive, _ := cmd.Flags().GetBool("recursive")
	exts, _ := cmd.Flags().GetStringSlice("ext")

	if isStdinPiped() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		content = runUpdate(ctx, logger, content, noCache)
		return writeOutputContent(content, "", outputPath)
	}

	if len(args) == 0 {
		return errors.New("no input: provide a file argument or pipe content via stdin")
	}

	files, err := collectInputFiles(args, exts, recursive)
	if err != nil {
		return err
	}
	if outputPath != "" && len(files) > 1 {
		return errors.New("--output cannot be used with multiple files or a directory")
	}

	for _, f := range files {
		str := lipgloss.Sprintln(internal.StyleYellow.Render("Updating"), fmt.Sprintf("imports in %q", f))
		_, _ = lipgloss.Fprint(os.Stderr, str)
		content, err := os.ReadFile(f)
		if err != nil {
			logger.Error(err.Error(), "file", f)
			continue
		}
		content = runUpdate(ctx, logger, content, noCache)
		if err := writeOutputContent(content, f, outputPath); err != nil {
			logger.Error(err.Error(), "file", f)
		}
	}
	return nil
}

func collectInputFiles(args []string, exts []string, recursive bool) ([]string, error) {
	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			dirFiles, err := collectFiles(arg, exts, recursive)
			if err != nil {
				return nil, err
			}
			files = append(files, dirFiles...)
		} else {
			files = append(files, arg)
		}
	}
	return files, nil
}

func runUpdate(ctx context.Context, logger *log.Logger, content []byte, noCache bool) []byte {
	imports := extractImportStatements(content)

	s := internal.SetupSpinner()
	s.Start()
	updates, logEvents := fetchLatestVersionsConcurrently(ctx, imports, noCache)
	s.Stop()

	for _, event := range logEvents {
		emitLogEvent(event, logger)
	}

	// Write to stderr, because stdout is reserved for content piped from stdin.
	printUpdateSummary(updates)
	UpdateFileContent(&content, updates)
	return content
}

func printUpdateSummary(updates map[string]Result) {
	if len(updates) == 0 {
		prefix := internal.StyleBlueBold.Render("info")
		text := internal.StyleNormal.Render("all dependencies are up to date")
		_, _ = lipgloss.Fprintf(os.Stderr, "%s: %s\n", prefix, text)
		return
	}
	for pkg, res := range updates {
		str := lipgloss.Sprintln(internal.StyleGreen.Render("  Updated"), pkg, res.Current, "->", res.Latest)
		_, _ = lipgloss.Fprint(os.Stderr, str)
	}
}

func collectFiles(root string, exts []string, recursive bool) ([]string, error) {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[strings.ToLower(e)] = true
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if extSet[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
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
		return os.WriteFile(outputPath, content, 0o644)
	}
	if inputFilePath != "" {
		return os.WriteFile(inputFilePath, content, 0o644)
	}
	_, err := os.Stdout.Write(content)
	return err
}

func fetchLatestVersionsConcurrently(ctx context.Context, imports [][]byte, noCache bool) (map[string]Result, []logEvent) {
	index, _ := fetchTypstVersionIndex(ctx, noCache)

	resultCh := make(chan Result, len(imports))
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

func fetchTypstVersionIndex(ctx context.Context, noCache bool) (map[string]string, error) {
	if !noCache {
		if cache, err := internal.LoadIndexCache(); err == nil && cache != nil && cache.IsValid() {
			return cache.Index, nil
		}
	}
	entries, err := internal.FetchTypstIndex(ctx)
	if err != nil {
		return nil, err
	}
	index := internal.BuildVersionIndex(entries)
	if !noCache {
		_ = internal.SaveIndexCache(index)
	}
	return index, nil
}

func processImport(ctx context.Context, importStatement []byte, index map[string]string, resultCh chan<- Result, logCh chan<- logEvent) {
	pkgName, currentVersion := parsePackageRef(importStatement)

	latestVersion, source, err := lookupVersion(ctx, index, pkgName)
	if err != nil {
		logCh <- logEvent{"error", err.Error(), nil}
		return
	}

	if latestVersion == currentVersion {
		logCh <- logEvent{"debug", "already at latest", []any{"package", pkgName}}
		return
	}

	logCh <- logEvent{"info", "update", []any{"package", pkgName, "from", currentVersion, "to", latestVersion, "via", source}}
	resultCh <- Result{Name: pkgName, Latest: latestVersion, Current: currentVersion}
}

func lookupVersionFromGitHub(ctx context.Context, pkgName string) (string, error) {
	apiURL, err := url.JoinPath(internal.TypstPackageEndpoint, pkgName)
	if err != nil {
		return "", err
	}
	response, err := internal.FetchDataFromGitHub(apiURL, ctx)
	if err != nil {
		return "", fmt.Errorf("could not find package %q", pkgName)
	}
	return internal.GetLatestVersion(response)
}

func collectVersionResults(resultCh <-chan Result) map[string]Result {
	versions := make(map[string]Result)
	for r := range resultCh {
		versions[r.Name] = r
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

type Result struct {
	Name    string
	Current string
	Latest  string
}

// UpdateFileContent updates all typst package import statements in content
// with the versions provided by the name→version mapping.
func UpdateFileContent(content *[]byte, versions map[string]Result) {
	for key, value := range versions {
		rawPattern := fmt.Sprintf(`@preview/%s:[0-9]*.[0-9]*.[0-9]*`, key)
		pattern := regexp.MustCompile(rawPattern)

		namespacePkg := strings.Split(rawPattern, ":")[0]
		replacement := fmt.Sprintf("%s:%s", namespacePkg, value.Latest)

		*content = pattern.ReplaceAll(*content, []byte(replacement))
	}
}
