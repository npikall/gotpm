package run_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/cmds/run"
	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/paths"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// project writes a manifest requiring floor, pinning pin, either "" to leave
// the key out, and declaring declared, and makes a subdirectory of the
// project the working directory. It returns the manifest's path.
func project(t *testing.T, floor, pin string, declared ...string) string {
	t.Helper()
	root := t.TempDir()
	var content strings.Builder
	content.WriteString(`# my project
[package]
name = "notes"   # what it is called
version = "0.1.0"
entrypoint = "main.typ"
`)
	if floor != "" {
		content.WriteString(`compiler = "` + floor + `"` + "\n")
	}
	if pin != "" || len(declared) > 0 {
		content.WriteString("\n[tool.gotpm]\n")
	}
	if pin != "" {
		content.WriteString("# chosen by hand\ncompiler-pin = \"" + pin + "\"\n")
	}
	if len(declared) > 0 {
		content.WriteString("dependencies = [\"" + strings.Join(declared, `", "`) + "\"]\n")
	}
	file := filepath.Join(root, manifest.FileName)
	require.NoError(t, paths.WriteFile(file, []byte(content.String())))
	sub := filepath.Join(root, "chapters", "one")
	require.NoError(t, paths.EnsureDir(sub))
	t.Chdir(sub)
	return file
}

func pinOf(t *testing.T, file string) string {
	t.Helper()
	m, err := manifest.LoadFile(file)
	require.NoError(t, err)
	return m.CompilerPin()
}

func readFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	return string(data)
}

func runIn(t *testing.T) (string, *[]run.Launch) {
	t.Helper()
	var out bytes.Buffer
	launches := record(t)
	require.NoError(t, run.Run(context.Background(), []string{"compile", "main.typ"}, &out, discardLogger()))
	require.Len(t, *launches, 1)
	return out.String(), launches
}

func TestRun_InsideAProjectLaunchesASatisfyingPinAsIs(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.12.0", "0.12.0")
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	for _, version := range []string{"0.12.0", "0.13.1", "0.14.0"} {
		srv.Publish(version)
	}
	install(t, srv, "0.12.0", "0.13.1")
	requests := len(srv.Requests())

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.12.0"), (*launches)[0].Binary,
		"the pin wins over newer installed and released compilers")
	assert.Equal(t, before, readFile(t, file), "the manifest is not touched")
	assert.Empty(t, out)
	assert.Len(t, srv.Requests(), requests, "an installed pin needs no network")
}

func TestRun_InsideAProjectInstallsAMissingPin(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.12.0", "0.12.0")
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.12.0")
	srv.Publish("0.14.0")

	_, launches := runIn(t)

	assert.Equal(t, binary(t, "0.12.0"), (*launches)[0].Binary)
	assert.Equal(t, before, readFile(t, file))
}

func TestRun_InsideAProjectWithoutAPinPinsTheNewestInstalledSatisfyingCompiler(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.12.0", "")
	srv := compilertest.Serve(t)
	for _, version := range []string{"0.11.0", "0.13.1", "0.14.0"} {
		srv.Publish(version)
	}
	srv.Publish("0.15.0-rc.1", compilertest.Prerelease())
	install(t, srv, "0.11.0", "0.13.1", "0.15.0-rc.1")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary)
	assert.Equal(t, "0.13.1", pinOf(t, file))
	assert.Equal(t, 1, strings.Count(out, "\n"), "announced in one line")
	assert.Contains(t, out, "0.13.1")
	assert.Contains(t, out, file)
}

func TestRun_InsideAProjectWithoutAPinPinsTheLatestSatisfyingRelease(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.12.0", "")
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	srv.Publish("0.15.0-rc.1", compilertest.Prerelease())

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.14.0"), (*launches)[0].Binary, "never a pre-release")
	assert.Equal(t, "0.14.0", pinOf(t, file))
	assert.Contains(t, out, "0.14.0")
}

func TestRun_InsideAProjectRaisesAPinBelowTheRequirement(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.13.0", "0.12.0")
	srv := compilertest.Serve(t)
	srv.Publish("0.12.0")
	srv.Publish("0.13.1")
	install(t, srv, "0.12.0")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary)
	assert.Equal(t, "0.13.1", pinOf(t, file))
	assert.Equal(t, 1, strings.Count(out, "\n"), "announced in one line")
	assert.Contains(t, out, "raised")
	assert.Contains(t, out, "0.12.0", "names the old pin")
	assert.Contains(t, out, "0.13.0", "names the requirement that raised it")
	assert.Contains(t, out, "0.13.1")
	assert.Contains(t, out, file)
}

func TestRun_InsideAProjectWithoutARequirementStillPins(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "", "")
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	install(t, srv, "0.13.1")

	_, launches := runIn(t)

	assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary, "chosen as outside a project")
	assert.Equal(t, "0.13.1", pinOf(t, file))
}

func TestRun_WritingThePinKeepsTheManifestsCommentsAndLayout(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.13.0", "0.12.0")
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")

	runIn(t)

	assert.Equal(t, strings.Replace(before, `compiler-pin = "0.12.0"`, `compiler-pin = "0.13.1"`, 1),
		readFile(t, file))
}

func TestRun_InsideAProjectWritesNoPinWhenTheInstallFails(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	file := project(t, "0.12.0", "")
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1", compilertest.WrongDigest())
	launches := record(t)

	require.Error(t, run.Run(context.Background(), nil, &bytes.Buffer{}, discardLogger()))

	assert.Empty(t, *launches)
	assert.Equal(t, before, readFile(t, file))
}
