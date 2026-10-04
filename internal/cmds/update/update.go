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
		return updateStdin(ctx, opts, log)
	}

	files, err := filesToUpdate(inputs, opts)
	if err != nil {
		return err
	}
	updateFiles(ctx, files, opts, log)
	return nil
}

func filesToUpdate(inputs []string, opts *Options) ([]string, error) {
	if len(inputs) == 0 {
		return nil, ErrMissingInput
	}
	files, err := collectInputFiles(inputs, opts)
	if err != nil {
		return nil, err
	}
	if opts.Output != "" && len(files) > 1 {
		return nil, ErrInvalidOutputOption
	}
	return files, nil
}

// updateFiles updates each file in turn. A file that fails is reported and
// skipped, so it does not hold up the others.
func updateFiles(ctx context.Context, files []string, opts *Options, log *log.Logger) {
	for _, file := range files {
		if err := updateFile(ctx, file, opts, log); err != nil {
			log.Error(err.Error(), "file", file)
		}
	}
}

func updateStdin(ctx context.Context, opts *Options, log *log.Logger) error {
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("could not read from stdin: %w", err)
	}
	announce("stdin")
	return writeOutput(update(ctx, content, opts, log), "", opts.Output)
}

func updateFile(ctx context.Context, file string, opts *Options, log *log.Logger) error {
	announce(file)
	content, err := os.ReadFile(file) //nolint: gosec
	if err != nil {
		return err //nolint: wrapcheck
	}
	return writeOutput(update(ctx, content, opts, log), file, opts.Output)
}

func update(ctx context.Context, content []byte, opts *Options, log *log.Logger) []byte {
	refs := typstsrc.FindRefs(content)

	updates, _ := ui.WithSpinner("", func() (map[string]Result, error) {
		return latestVersions(ctx, refs, opts.NoCache, log), nil
	})

	summarise(updates)

	latest := make(map[string]string, len(updates))
	for name, result := range updates {
		latest[name] = result.Latest
	}
	return typstsrc.RewriteRefs(content, latest)
}

func latestVersions(ctx context.Context, refs []pkg.Ref, noCache bool, log *log.Logger) map[string]Result {
	idx, _ := index.Load(ctx, index.Opts{NoCache: noCache})

	resultCh := make(chan Result, len(refs))

	var wg sync.WaitGroup
	for _, ref := range refs {
		wg.Go(func() {
			resolve(ctx, ref, idx, resultCh, log)
		})
	}
	wg.Wait()
	close(resultCh)

	results := make(map[string]Result)
	for result := range resultCh {
		results[result.Name] = result
	}
	return results
}

func resolve(ctx context.Context, ref pkg.Ref, idx index.Index, resultCh chan<- Result, log *log.Logger) {
	latest, source, err := lookup(ctx, idx, ref.Name)
	if err != nil {
		log.Error(err.Error(), "package", ref.Name)
		return
	}

	current := ref.Version.String()
	if latest == current {
		log.Debug("already at latest", "package", ref.Name)
		return
	}

	log.Info("update", "package", ref.Name, "from", current, "to", latest, "via", source)
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
	walk := &dirWalk{root: root, recursive: opts.Recursive, wanted: extensionSet(opts.Extensions)}
	if err := filepath.WalkDir(root, walk.visit); err != nil {
		return nil, fmt.Errorf("could not walk directory %q: %w", root, err)
	}
	return walk.files, nil
}

// extensionSet normalises extensions to lower case with a leading dot.
func extensionSet(extensions []string) map[string]bool {
	set := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		set[strings.ToLower(ext)] = true
	}
	return set
}

// dirWalk collects the files below root with a wanted extension.
type dirWalk struct {
	root      string
	recursive bool
	wanted    map[string]bool
	files     []string
}

func (w *dirWalk) visit(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		return w.enter(path)
	}
	if w.wanted[strings.ToLower(filepath.Ext(path))] {
		w.files = append(w.files, path)
	}
	return nil
}

// enter allows walking into the root, and into sub-directories only when
// recursing.
func (w *dirWalk) enter(path string) error {
	if path != w.root && !w.recursive {
		return filepath.SkipDir
	}
	return nil
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
