package smoke_test

import (
	"path/filepath"
	"testing"

	"github.com/npikall/gotpm/internal/config"
	"github.com/npikall/gotpm/internal/manifest"
	"github.com/npikall/gotpm/internal/testrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	// Somewhere that is not a package yet, which is the only place init has
	// anything to do.
	dir := t.TempDir()

	h.RunIn(dir, "init", "my-package")

	assert.FileExists(t, filepath.Join(dir, "my-package", manifest.FileName))
}

func TestAdd(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()

	h.MustRun("add", dep.URL())

	assert.DirExists(t, h.InstalledDir("gotpm", "cetz", "0.3.1"))
}

func TestSync(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	h.Declare(dep)

	h.MustRun("sync")

	assert.DirExists(t, h.InstalledDir("gotpm", "cetz", "0.3.1"))
}

func TestRemove(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	dep := testrepo.New(t, "cetz", "0.3.1").Release()
	h.Declare(dep)

	h.MustRun("remove", dep.Import())

	assert.NotContains(t, h.Read(manifest.FileName), dep.Import())
}

func TestInstall(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	source := testrepo.New(t, "cetz", "0.3.1").Release()

	h.MustRun("install", source.Dir())

	assert.DirExists(t, h.InstalledDir("local", "cetz", "0.3.1"))
}

func TestUninstall(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	source := testrepo.New(t, "cetz", "0.3.1").Release()
	h.Install("local", "cetz", "0.3.1", source.Dir())

	h.MustRun("uninstall", "cetz", "--version", "0.3.1")

	assert.NoDirExists(t, h.InstalledDir("local", "cetz", "0.3.1"))
}

func TestList(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	source := testrepo.New(t, "cetz", "0.3.1").Release()
	h.Install("local", "cetz", "0.3.1", source.Dir())

	res := h.MustRun("list")

	assert.Contains(t, res.Stdout, "cetz")
}

func TestLocate(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)

	assert.Equal(t, h.Packages, h.Locate("packages"))
}

func TestBump(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)

	h.MustRun("bump", "patch")

	assert.Contains(t, h.Read(manifest.FileName), `version = "0.1.1"`)
}

func TestConfig(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)

	h.MustRun("config", "set", "fork.url", "https://github.com/user/packages")

	assert.Contains(t, h.MustRun("config", "get", "fork.url").Stdout, "https://github.com/user/packages")
}

func TestCacheClear(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	h.SeedIndex(map[string]string{"cetz": "0.4.0"})
	index := h.Locate("index")
	require.FileExists(t, index)

	h.MustRun("cache", "clear")

	assert.NoFileExists(t, index)
}

func TestCheck(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	h.SeedIndex(map[string]string{"cetz": "0.4.0"})
	h.Write("main.typ", `#import "@preview/cetz:0.3.1": *`)

	h.MustRun("check", "main.typ")
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	h.SeedIndex(map[string]string{"cetz": "0.4.0"})
	h.Write("main.typ", `#import "@preview/cetz:0.3.1": *`)

	h.MustRun("update", "main.typ")

	assert.Contains(t, h.Read("main.typ"), "@preview/cetz:0.4.0")
}

func TestPublishLocal(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)
	forkPath := filepath.Join(t.TempDir(), "fork-clone")
	h.Configure(config.Config{Fork: config.ForkConfig{URL: testrepo.Fork(t), Path: forkPath}})
	h.Write("lib.typ", "#let greet(name) = [Hello #name]")
	h.Write(manifest.FileName,
		"[package]\nname = \"my-doc\"\nversion = \"0.1.0\"\nentrypoint = \"lib.typ\"\n")

	// --local stages the submission and stops. Nothing here can push, and the
	// subprocess holds no credentials to push with.
	h.MustRun("publish", "--local")

	assert.DirExists(t, filepath.Join(forkPath, "packages", "preview", "my-doc", "0.1.0"))
}

func TestSelfVersion(t *testing.T) {
	t.Parallel()
	skipShort(t)
	h := newHarness(t)

	res := h.MustRun("self", "version")

	// Built without the release ldflags, so this is what the binary is.
	assert.Contains(t, res.Stdout+res.Stderr, "dev")
}
