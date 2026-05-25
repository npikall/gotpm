package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildPackageDir creates a fake packages directory with the layout:
//
//	root/@ns/pkg/0.1.0/
func buildPackageDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "@ns", "pkg", "0.1.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestScanPackages_Regular(t *testing.T) {
	root := buildPackageDir(t)

	namespaces, err := scanPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(namespaces) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(namespaces))
	}
	ns := namespaces[0]
	if ns.Name != "@ns" {
		t.Errorf("namespace name: got %q, want %q", ns.Name, "@ns")
	}
	if len(ns.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(ns.Packages))
	}
	pkg := ns.Packages[0]
	if len(pkg.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(pkg.Versions))
	}
	v := pkg.Versions[0]
	if v.Name != "0.1.0" {
		t.Errorf("version name: got %q, want %q", v.Name, "0.1.0")
	}
	if v.Editable {
		t.Error("regular version should not be marked editable")
	}
}

func TestScanPackages_Editable(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "main.typ"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	pkgDir := filepath.Join(root, "@local", "mypkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Symlink the version directory to srcDir (editable install)
	if err := os.Symlink(srcDir, filepath.Join(pkgDir, "0.2.0")); err != nil {
		t.Fatal(err)
	}

	namespaces, err := scanPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(namespaces) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(namespaces))
	}
	pkg := namespaces[0].Packages[0]
	if len(pkg.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(pkg.Versions))
	}
	v := pkg.Versions[0]
	if v.Name != "0.2.0" {
		t.Errorf("version name: got %q, want %q", v.Name, "0.2.0")
	}
	if !v.Editable {
		t.Error("symlinked version should be marked editable")
	}
}

func TestScanPackages_MixedVersions(t *testing.T) {
	srcDir := t.TempDir()

	root := t.TempDir()
	pkgDir := filepath.Join(root, "@ns", "pkg")

	if err := os.MkdirAll(filepath.Join(pkgDir, "0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(srcDir, filepath.Join(pkgDir, "0.2.0")); err != nil {
		t.Fatal(err)
	}

	namespaces, err := scanPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	pkg := namespaces[0].Packages[0]
	if len(pkg.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(pkg.Versions))
	}

	// Versions are sorted: 0.1.0 < 0.2.0
	if pkg.Versions[0].Editable {
		t.Error("0.1.0 should not be editable")
	}
	if !pkg.Versions[1].Editable {
		t.Error("0.2.0 (symlink) should be editable")
	}
}

func TestScanPackages_EmptyDir(t *testing.T) {
	root := t.TempDir()
	namespaces, err := scanPackages(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 0 {
		t.Errorf("expected 0 namespaces, got %d", len(namespaces))
	}
}

func TestScanPackages_NonDirInNamespace_Skipped(t *testing.T) {
	root := t.TempDir()
	nsDir := filepath.Join(root, "local")
	require.NoError(t, os.MkdirAll(nsDir, 0o755))
	// plain file in namespace dir — should not be treated as package
	require.NoError(t, os.WriteFile(filepath.Join(nsDir, "not-a-pkg"), []byte(""), 0o644))

	namespaces, err := scanPackages(root)
	require.NoError(t, err)
	// namespace exists but has no valid packages
	if len(namespaces) > 0 {
		assert.Empty(t, namespaces[0].Packages)
	}
}

func TestScanPackages_PackageWithNoVersions_Excluded(t *testing.T) {
	root := t.TempDir()
	// pkg dir exists but contains only a plain file (no version dirs)
	pkgDir := filepath.Join(root, "local", "empty-pkg")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "not-a-version"), []byte(""), 0o644))

	namespaces, err := scanPackages(root)
	require.NoError(t, err)
	if len(namespaces) > 0 {
		assert.Empty(t, namespaces[0].Packages, "package with no valid version dirs must be excluded")
	}
}

func TestScanPackages_MultipleNamespacesSortedAlphabetically(t *testing.T) {
	root := t.TempDir()
	for _, ns := range []string{"preview", "alpha", "local"} {
		dir := filepath.Join(root, ns, "pkg", "0.1.0")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}

	namespaces, err := scanPackages(root)
	require.NoError(t, err)
	require.Len(t, namespaces, 3)
	assert.Equal(t, "alpha", namespaces[0].Name)
	assert.Equal(t, "local", namespaces[1].Name)
	assert.Equal(t, "preview", namespaces[2].Name)
}

func TestScanPackages_PackagesSortedAlphabetically(t *testing.T) {
	root := t.TempDir()
	for _, pkg := range []string{"zoo", "alpha", "middle"} {
		dir := filepath.Join(root, "local", pkg, "0.1.0")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}

	namespaces, err := scanPackages(root)
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	pkgs := namespaces[0].Packages
	require.Len(t, pkgs, 3)
	assert.Equal(t, "alpha", pkgs[0].Name)
	assert.Equal(t, "middle", pkgs[1].Name)
	assert.Equal(t, "zoo", pkgs[2].Name)
}

func TestScanPackages_VersionsSortedAlphabetically(t *testing.T) {
	root := t.TempDir()
	for _, ver := range []string{"0.3.0", "0.1.0", "0.2.0"} {
		dir := filepath.Join(root, "local", "pkg", ver)
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}

	namespaces, err := scanPackages(root)
	require.NoError(t, err)
	versions := namespaces[0].Packages[0].Versions
	require.Len(t, versions, 3)
	assert.Equal(t, "0.1.0", versions[0].Name)
	assert.Equal(t, "0.2.0", versions[1].Name)
	assert.Equal(t, "0.3.0", versions[2].Name)
}

func TestIsDirPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(file, []byte(""), 0o644))

	assert.True(t, isDirPath(dir))
	assert.False(t, isDirPath(file))
	assert.False(t, isDirPath(filepath.Join(dir, "nonexistent")))
}
