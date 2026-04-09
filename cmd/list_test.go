package cmd

import (
	"os"
	"path/filepath"
	"testing"
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
