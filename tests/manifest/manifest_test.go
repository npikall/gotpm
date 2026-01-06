package manifest_test

import (
	"bytes"
	"errors"
	"testing"

	i "github.com/npikall/gotpm/internal/manifest"
)

var SpyError = errors.New("spy error")

func SpyUnmarshal(data []byte, v any) error {
	return SpyError
}

func TestTypstTOMLUnmarshal(t *testing.T) {
	example := []byte(`[package]
name = "foo"
version = "0.0.0"
entrypoint = "bar"
`)
	t.Run("successful", func(t *testing.T) {
		want := i.PackageInfo{Name: "foo", Version: "0.0.0", Entrypoint: "bar"}
		got, err := i.TypstTOMLUnmarshal(example)
		assertNoErr(t, err)

		assertEqual(t, got, want)
	})
	t.Run("not successful", func(t *testing.T) {
		_, err := i.ConfigureableUnmarshal(example, SpyUnmarshal)
		assertErr(t, err, SpyError)
	})
}

func TestSetManifestFields(t *testing.T) {
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
		err := i.WriteTOML(buf, pkg, example)
		assertNoErr(t, err)

		if buf.String() != string(want) {
			t.Errorf("got %q want %q", buf.String(), string(want))
		}
	})
	t.Run("not successful", func(t *testing.T) {
		buf := new(bytes.Buffer)
		pkg := i.PackageInfo{}
		err := i.ConfigurableWriteTOML(buf, pkg, example, SpyUnmarshal)
		assertErr(t, err, SpyError)
	})
}

func TestValidation(t *testing.T) {
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

func assertNoErr(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Errorf("should not error")
	}
}
func assertErr(t *testing.T, got, want error) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func assertEqual(t *testing.T, got, want any) {
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
