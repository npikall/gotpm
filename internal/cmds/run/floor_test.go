package run_test

import (
	"strings"
	"testing"

	"github.com/npikall/gotpm/internal/compilers/compilertest"
	"github.com/npikall/gotpm/internal/pkg"
	"github.com/npikall/gotpm/internal/store"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installed releases p and installs it into the package directory, the way a
// Sync would have left it, and returns its package ref.
func installed(t *testing.T, p *testrepo.Package) string {
	t.Helper()
	p.Release()
	ref, err := pkg.ParseImport(p.Import())
	require.NoError(t, err)
	s, err := store.OpenPackageDir()
	require.NoError(t, err)
	require.NoError(t, s.Install(ref, p.Dir()))
	return p.Import()
}

func TestRun_ADependencyRequiringANewerCompilerRaisesThePin(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	newer := installed(t, testrepo.New(t, "newer", "1.0.0").RequireCompiler("0.14.0"))
	older := installed(t, testrepo.New(t, "older", "1.0.0").RequireCompiler("0.11.0"))
	file := project(t, "0.12.0", "0.13.1", older, newer)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	install(t, srv, "0.13.1")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.14.0"), (*launches)[0].Binary)
	assert.Equal(t, "0.14.0", pinOf(t, file))
	assert.Equal(t, 1, strings.Count(out, "\n"), "announced in one line")
	assert.Contains(t, out, "raised")
	assert.Contains(t, out, "0.13.1", "names the old pin")
	assert.Contains(t, out, newer, "names the dependency that raised the floor")
	assert.NotContains(t, out, older)
}

func TestRun_ADependencyRaisesTheFloorAProjectWithoutAPinIsPinnedAt(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	newer := installed(t, testrepo.New(t, "newer", "1.0.0").RequireCompiler("0.14.0"))
	file := project(t, "0.12.0", "", newer)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	install(t, srv, "0.13.1")

	_, launches := runIn(t)

	assert.Equal(t, binary(t, "0.14.0"), (*launches)[0].Binary)
	assert.Equal(t, "0.14.0", pinOf(t, file))
}

func TestRun_NeverReadsADependencysCompilerPin(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := installed(t, testrepo.New(t, "dep", "1.0.0").RequireCompiler("0.12.0").PinCompiler("0.14.0"))
	file := project(t, "0.12.0", "0.13.1", dep)
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.13.1")
	srv.Publish("0.14.0")
	install(t, srv, "0.13.1", "0.14.0")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.13.1"), (*launches)[0].Binary, "the dependency's pin is not a floor")
	assert.Equal(t, before, readFile(t, file))
	assert.Empty(t, out)
}

func TestRun_ADependencyWithoutARequirementContributesNothing(t *testing.T) { //nolint: paralleltest
	testrepo.Isolate(t)
	dep := installed(t, testrepo.New(t, "dep", "1.0.0"))
	file := project(t, "0.12.0", "0.12.0", dep)
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.12.0")
	install(t, srv, "0.12.0")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.12.0"), (*launches)[0].Binary)
	assert.Equal(t, before, readFile(t, file))
	assert.Empty(t, out)
}

func TestRun_ADependencyThatIsNotInstalledIsNotedAndContributesNothing(t *testing.T) { //nolint: paralleltest
	packages := testrepo.Isolate(t)
	missing := testrepo.New(t, "missing", "1.0.0").RequireCompiler("0.14.0").Release().Import()
	file := project(t, "0.12.0", "0.12.0", missing)
	before := readFile(t, file)
	srv := compilertest.Serve(t)
	srv.Publish("0.12.0")
	srv.Publish("0.14.0")
	install(t, srv, "0.12.0")

	out, launches := runIn(t)

	assert.Equal(t, binary(t, "0.12.0"), (*launches)[0].Binary)
	assert.Equal(t, before, readFile(t, file))
	assert.Equal(t, 1, strings.Count(out, "\n"), "noted in one line")
	assert.Contains(t, out, missing)
	assert.Contains(t, out, "gotpm sync", "says how to install it")
	assert.NoDirExists(t, packages, "run never Syncs")
}
