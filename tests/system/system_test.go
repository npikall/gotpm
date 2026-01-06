package system_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/system"
)

func TestGetTypstPath(t *testing.T) {
	cases := []struct {
		goos string
		path string
	}{
		{"darwin", "~/Library/Application Support"},
		{"windows", "~/AppData/Roaming"},
		{"linux", "~/.local/share"},
	}

	for _, tt := range cases {
		t.Run(tt.goos, func(t *testing.T) {
			got, _ := i.GetTypstPath(tt.goos, "~")
			if got != tt.path {
				t.Errorf("got %q want %q", got, tt.path)
			}
		})
	}

	t.Run("unsupported os", func(t *testing.T) {
		_, err := i.GetTypstPath("", "~")
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
		if err != i.ErrOperatingSystem {
			t.Fatalf("got %q want %q", err, i.ErrOperatingSystem)
		}
	})

	t.Run("custom windows path", func(t *testing.T) {
		// TODO: Test absolute and relative paths
		t.Setenv("APPDATA", "~/path/foo")
		got, _ := i.GetTypstPath(i.WINDOWS, "~")
		want := "~/path/foo"

		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("custom linux path", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "~/path/foo")
		got, _ := i.GetTypstPath(i.LINUX, "~")
		want := "~/path/foo"

		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
