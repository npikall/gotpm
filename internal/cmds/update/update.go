// Package update implements the update command: it rewrites the Typst
// Universe imports of a file to the latest published version of each package.
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/typstsrc"
	"github.com/npikall/gotpm/internal/ui"
)

var (
	ErrMissingInput        = errors.New("no input: provide a file argument or pipe content via stdin")
	ErrInvalidOutputOption = errors.New("option '--output' cannot be used with multiple files or a directory")
)

// Options holds the resolved update flags.
type Options struct {
	// Output writes the result elsewhere than the input file.
	Output string
	// NoCache skips the on-disk package index cache.
	NoCache bool
	// Recursive descends into sub-directories of a directory argument.
	Recursive bool
	// Extensions selects which files of a directory are processed.
	Extensions []string
}

// Result is one package whose imports can be moved forward.
type Result struct {
	Name    string
	Current string
	Latest  string
}

// Run updates the given files, or content piped via stdin when there are none.
func Run(inputs []string, opts *Options, log *log.Logger) error {
	ctx := context.Background()

	if isStdinPiped() {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("could not read from stdin: %w", err)
		}
		announce("stdin")
		return writeOutput(update(ctx, content, opts, log), "", opts.Output)
	}

	if len(inputs) == 0 {
		return ErrMissingInput
	}

	files, err := collectInputFiles(inputs, opts)
	if err != nil {
		return err
	}
	if opts.Output != "" && len(files) > 1 {
		return ErrInvalidOutputOption
	}

	for _, file := range files {
		announce(file)
		content, err := os.ReadFile(file) //nolint: gosec
		if err != nil {
			log.Error(err.Error(), "file", file)
			continue
		}
		if err := writeOutput(update(ctx, content, opts, log), file, opts.Output); err != nil {
			log.Error(err.Error(), "file", file)
		}
	}
	return nil
}

func update(ctx context.Context, content []byte, opts *Options, log *log.Logger) []byte {
	refs := typstsrc.FindRefs(content)

	spin := ui.Spinner("")
	spin.Start()
	updates, events := latestVersions(ctx, refs, opts.NoCache)
	spin.Stop()

	for _, event := range events {
		event.emit(log)
	}

	summarise(updates)

	latest := make(map[string]string, len(updates))
	for name, result := range updates {
		latest[name] = result.Latest
	}
	return typstsrc.RewriteRefs(content, latest)
}

func latestVersions(ctx context.Context, refs []pkg.Ref, noCache bool) (map[string]Result, []logEvent) {
	idx, _ := index.Load(ctx, index.Opts{NoCache: noCache})

	resultCh := make(chan Result, len(refs))
	logCh := make(chan logEvent, len(refs))

	var wg sync.WaitGroup
	for _, ref := range refs {
		wg.Go(func() {
			resolve(ctx, ref, idx, resultCh, logCh)
		})
	}
	wg.Wait()
	close(resultCh)
	close(logCh)

	results := make(map[string]Result)
	for result := range resultCh {
		results[result.Name] = result
	}
	var events []logEvent
	for event := range logCh {
		events = append(events, event)
	}
	return results, events
}

func resolve(ctx context.Context, ref pkg.Ref, idx index.Index, resultCh chan<- Result, logCh chan<- logEvent) {
	latest, source, err := lookup(ctx, idx, ref.Name)
	if err != nil {
		logCh <- logEvent{"error", err.Error(), nil}
		return
	}

	current := ref.Version.String()
	if latest == current {
		logCh <- logEvent{"debug", "already at latest", []any{"package", ref.Name}}
		return
	}

	logCh <- logEvent{"info", "update", []any{"package", ref.Name, "from", current, "to", latest, "via", source}}
	resultCh <- Result{Name: ref.Name, Current: current, Latest: latest}
}

func lookup(ctx context.Context, idx index.Index, name string) (string, string, error) {
	if version, ok := idx.Latest(name); ok {
		return version, "index", nil
	}
	version, err := index.LatestOnGitHub(ctx, name)
	return version, "github", err
}

func collectInputFiles(inputs []string, opts *Options) ([]string, error) {
	var files []string
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return nil, fmt.Errorf("could not read fileinfo: %w", err)
		}
		if !info.IsDir() {
			files = append(files, input)
			continue
		}
		found, err := collectDir(input, opts)
		if err != nil {
			return nil, err
		}
		files = append(files, found...)
	}
	return files, nil
}

func collectDir(root string, opts *Options) ([]string, error) {
	extensions := make(map[string]bool, len(opts.Extensions))
	for _, ext := range opts.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extensions[strings.ToLower(ext)] = true
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && !opts.Recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if extensions[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk directory %q: %w", root, err)
	}
	return files, nil
}

func writeOutput(content []byte, inputFile, outputPath string) error {
	if outputPath != "" {
		return paths.WriteFile(outputPath, content)
	}
	if inputFile != "" {
		return paths.WriteFile(inputFile, content)
	}
	if _, err := os.Stdout.Write(content); err != nil {
		return fmt.Errorf("could not write to 'stdout': %w", err)
	}
	return nil
}

func announce(name string) {
	_, _ = lipgloss.Fprint(os.Stderr, lipgloss.Sprintln(ui.ANSIGreenBold.Render("Updating"), fmt.Sprintf("%q", name)))
}

func summarise(updates map[string]Result) {
	if len(updates) == 0 {
		text := ui.Normal.Render("All dependencies are up to date")
		_, _ = lipgloss.Fprintf(os.Stderr, "  %s\n\n", text)
		return
	}
	for name, result := range updates {
		line := lipgloss.Sprintln(ui.Green.Render(" ", name), result.Current, "->", result.Latest)
		_, _ = lipgloss.Fprint(os.Stderr, line)
	}
	_, _ = lipgloss.Fprint(os.Stderr, "\n")
}

func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
