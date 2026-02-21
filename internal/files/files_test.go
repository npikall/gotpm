package files_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	i "github.com/npikall/gotpm/internal/files"
	"github.com/stretchr/testify/assert"
)

var ErrSpy = errors.New("spy error")

func SpyUnmarshal(data []byte, v any) error {
	return ErrSpy
}

func TestUnmarshalToPackage(t *testing.T) {
	example := []byte(`[package]
name = "foo"
version = "0.0.0"
entrypoint = "bar"
`)
	t.Run("successful", func(t *testing.T) {
		want := i.PackageInfo{Name: "foo", Version: "0.0.0", Entrypoint: "bar"}
		got, err := i.UnmarshalToPackage(example)
		assert.NoError(t, err)

		assert.Equal(t, want, got)
	})
	t.Run("not successful", func(t *testing.T) {
		_, err := i.ConfigureableUnmarshalToPackage(example, SpyUnmarshal)
		assertErr(t, err, ErrSpy)
	})
}

func TestUpdateToml(t *testing.T) {
	example := []byte(`[package]
name = "foo"
version = "0.0.0"
entrypoint = "bar"
extrafield = "extra"

[tool]
example = "str"
`)
	want := []byte(`[package]
entrypoint = "bar"
extrafield = "extra"
name = "foo"
version = "changed"

[tool]
example = "str"
`)

	t.Run("successful", func(t *testing.T) {
		buf := new(bytes.Buffer)
		pkg := i.PackageInfo{Name: "foo", Version: "changed", Entrypoint: "bar"}
		err := i.UpdateTOML(buf, pkg, example, false)
		assert.NoError(t, err)

		if buf.String() != string(want) {
			t.Errorf("got %q want %q", buf.String(), string(want))
		}
	})
	t.Run("not successful", func(t *testing.T) {
		buf := new(bytes.Buffer)
		pkg := i.PackageInfo{}
		err := i.ConfigurableUpdateToml(buf, pkg, example, SpyUnmarshal, false)
		assertErr(t, err, ErrSpy)
	})
}

func TestPackageInfo_ValidateVersion(t *testing.T) {
	cases := []struct {
		name    string
		version i.PackageInfo
		want    bool
	}{
		{"valid", i.PackageInfo{Version: "0.0.0"}, true},
		{"valid", i.PackageInfo{Version: "123.123.123"}, true},
		{"invalid", i.PackageInfo{Version: "a.0.0"}, false},
		{"invalid", i.PackageInfo{Version: "a.b.c"}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.ValidateVersion()
			if got != tt.want {
				t.Errorf("got %t wsnt %t given %s", got, tt.want, tt.version.Version)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	want := []byte("Hello World")
	_ = os.WriteFile(src, want, 0644)

	// Actual tested function
	if err := i.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	got, _ := os.ReadFile(dst)
	assert.Equal(t, want, got, "got %s want %s", string(got), string(want))
}

func assertErr(t *testing.T, got, want error) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPackageInfo_Bump(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		initialVersion string
		want           string
		increment      string
		wantErr        bool
	}{
		{"bump major", "0.1.0", "1.0.0", "major", false},
		{"bump minor", "0.1.0", "0.2.0", "minor", false},
		{"bump patch", "0.1.0", "0.1.1", "patch", false},
		{"bump 0.2.3", "0.1.0", "0.2.3", "0.2.3", false},
		{"bump error", "0.1.0", "0.1.0", "error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := i.NewPackageInfo()
			p.SetVersion(tt.initialVersion)
			gotErr := p.Bump(tt.increment)
			assert.Equal(t, tt.want, p.Version)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Bump() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Bump() succeeded unexpectedly")
			}
		})
	}
}
