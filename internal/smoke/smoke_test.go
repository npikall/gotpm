// Package smoke runs the gotpm binary as a subprocess, once per command, and
// checks that it exits cleanly and did the one thing the command exists to do.
//
// It is deliberately shallow. What a command does is tested beside the command
// under internal/cmds, where a test can reach into the result; these tests
// cover what those cannot — that the flags reach the options, that the command
// is registered at all, and that the exit code a caller sees is the right one.
//
// Every command is exercised offline. Repositories are local and reached over
// file://, the Typst Universe index is seeded into the cache, and publish stops
// before it pushes.
package smoke_test

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/index"
	"github.com/npikall/gotpm/internal/lockfile"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/require"
)

const (
	// runTimeout bounds one gotpm invocation. Everything these tests reach is
	// on the local disk, so the budget is there to kill a command waiting on
	// input or on a network it should never have touched, not to allow for
	// slow work.
	runTimeout = 10 * time.Second
	// buildTimeout bounds the one-off build in TestMain, which has to compile
	// the whole module and links a much larger binary than any test runs.
	buildTimeout = 5 * time.Minute
)

// binary is the gotpm built by TestMain, shared by every test.
var binary string

func TestMain(m *testing.M) {
	// Parsed here so testing.Short is readable before any test runs: under
	// -short every test skips, and building the binary would be the only work
	// the package did.
	flag.Parse()
	if testing.Short() {
		m.Run()
		return
	}

	dir, err := os.MkdirTemp("", "gotpm-smoke")
	if err != nil {
		panic("smoke: could not create build directory: " + err.Error())
	}
	defer os.RemoveAll(dir)

	binary = filepath.Join(dir, "gotpm")
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	// Built without the release ldflags, so the binary reports itself as a
	// development build. That is what keeps 'self update' from reaching the
	// network, and it is why nothing here asserts on a version string.
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../..")
	if out, err := build.CombinedOutput(); err != nil {
		panic("smoke: could not build gotpm: " + err.Error() + "\n" + string(out))
	}

	m.Run()
}

// result is what one gotpm invocation produced.
type result struct {
	Stdout string
	Stderr string
	Code   int
}

// harness is one isolated machine to run gotpm against: its own package store,
// data directory, config and project.
type harness struct {
	t *testing.T
	// Root is the isolated stand-in for the user's home.
	Root string
	// Packages is the package directory the Typst compiler would resolve from.
	Packages string
	// Project is the working directory gotpm runs in.
	Project string

	env []string
}

// newHarness builds an isolated machine with an empty project on it. It is
// safe to call from a parallel test: nothing here touches process-wide state.
func newHarness(t *testing.T) *harness {
	t.Helper()
	root := t.TempDir()
	testrepo.WriteGitIdentity(t, root)
	return &harness{
		t:        t,
		Root:     root,
		Packages: testrepo.Packages(root),
		Project:  testrepo.ProjectAt(t, "my-doc"),
		// PATH is not part of isolation — it is what lets a subprocess be
		// started at all — so it is added here rather than in testrepo.Env.
		env: append(testrepo.Env(root), "PATH="+os.Getenv("PATH")),
	}
}

// Locate asks the binary for one of the paths it resolves. The alternative is
// for the harness to re-derive them, which means a second copy of the
// per-platform rules in internal/paths; asking is the only version that is
// right on every platform. A wrong answer here is loud rather than silent —
// the seeded index would land somewhere gotpm does not read, and the commands
// that need it would try to reach the network and fail.
func (h *harness) Locate(key string) string {
	h.t.Helper()
	return strings.TrimSpace(h.MustRun("locate", key).Stdout)
}

// Run invokes gotpm in the project and returns what it produced, whatever it
// exited with.
func (h *harness) Run(args ...string) result {
	h.t.Helper()
	return h.RunIn(h.Project, args...)
}

// RunIn invokes gotpm somewhere other than the project, for the commands whose
// whole job is to act on a directory that is not one yet.
func (h *harness) RunIn(dir string, args ...string) result {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(h.t.Context(), runTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = h.env

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		h.t.Fatalf("gotpm %s could not run: %v", strings.Join(args, " "), err)
	}
	require.NoError(h.t, ctx.Err(), "gotpm %s did not finish within %s", strings.Join(args, " "), runTimeout)

	return result{Stdout: stdout.String(), Stderr: stderr.String(), Code: cmd.ProcessState.ExitCode()}
}

// MustRun invokes gotpm and fails the test unless it succeeded.
func (h *harness) MustRun(args ...string) result {
	h.t.Helper()
	res := h.Run(args...)
	if res.Code != 0 {
		h.t.Fatalf("gotpm %s exited %d\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), res.Code, res.Stdout, res.Stderr)
	}
	return res
}

// SeedIndex writes a Typst Universe index into the cache, so the commands that
// consult the index run without reaching packages.typst.org. The cache carries
// the current time, so it is inside its TTL for the length of the test.
func (h *harness) SeedIndex(packages map[string]string) {
	h.t.Helper()
	require.NoError(h.t, index.SaveCacheAt(h.Locate("data-dir"), packages))
}

// Write puts a file into the project and returns its path.
func (h *harness) Write(name, content string) string {
	h.t.Helper()
	path := filepath.Join(h.Project, name)
	require.NoError(h.t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// Read returns the contents of a file in the project.
func (h *harness) Read(name string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.Project, name))
	require.NoError(h.t, err)
	return string(data)
}

// InstalledDir is where a package version lands in the package directory.
func (h *harness) InstalledDir(namespace, name, version string) string {
	return filepath.Join(h.Packages, namespace, name, version)
}

// Install puts a package into the package directory, for the commands that
// need one to already be there. It goes through internal/store rather than
// through gotpm itself, so a broken install command fails the install test and
// nothing else.
func (h *harness) Install(namespace, name, version, srcDir string) {
	h.t.Helper()
	ref, err := pkg.New(namespace, name, version)
	require.NoError(h.t, err)
	require.NoError(h.t, store.At(h.Packages).Install(ref, srcDir))
}

// Declare writes the manifest and lock of a project depending on deps, which
// is the state the commands that read a project's dependencies start from.
func (h *harness) Declare(deps ...*testrepo.Package) {
	h.t.Helper()
	lock := lockfile.New()
	imports := make([]string, 0, len(deps))
	for _, dep := range deps {
		imports = append(imports, dep.Import())
		entry := dep.LockEntry()
		entry.Direct = true
		lock.Upsert(entry)
	}

	var manifestFile strings.Builder
	manifestFile.WriteString("[package]\nname = \"my-doc\"\nversion = \"0.1.0\"\nentrypoint = \"main.typ\"\n")
	manifestFile.WriteString("\n[tool.gotpm]\ndependencies = [\n")
	for _, imp := range imports {
		manifestFile.WriteString("  \"" + imp + "\",\n")
	}
	manifestFile.WriteString("]\n")

	h.Write(manifest.FileName, manifestFile.String())
	require.NoError(h.t, lockfile.Save(h.Project, lock))
}

// Configure writes gotpm's config file, so a command reading it finds what the
// test put there rather than what the user has set.
func (h *harness) Configure(cfg config.Config) {
	h.t.Helper()
	path := h.Locate("config")
	require.NoError(h.t, paths.EnsureDir(filepath.Dir(path)))

	var buf strings.Builder
	require.NoError(h.t, toml.NewEncoder(&buf).Encode(cfg))
	require.NoError(h.t, os.WriteFile(path, []byte(buf.String()), 0o600))
}

// skipShort keeps the suite out of the fast inner loop. It is compiled either
// way, so it cannot drift out of the build the way a tagged suite does.
func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("smoke tests build and run the binary")
	}
}
