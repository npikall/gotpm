package system_test

import (
	"testing"

	i "github.com/npikall/gotpm/internal/system"
	"github.com/stretchr/testify/assert"
)

func TestGetTypstPathRoot(t *testing.T) {
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
			got, _ := i.GetDataDirectory(tt.goos, "~")
			assert.Equal(t, tt.path, got, "got %s want %s", got, tt.path)
		})
	}

	t.Run("unsupported os", func(t *testing.T) {
		_, err := i.GetDataDirectory("", "~")
		assert.Error(t, err, "expected an error, but got none")
		assert.Equal(t, err, i.ErrOperatingSystem, "got %q want %q", err, i.ErrOperatingSystem)
	})

	t.Run("custom windows path", func(t *testing.T) {
		// TODO: Test absolute and relative paths
		t.Setenv("APPDATA", "~/path/foo")
		got, _ := i.GetDataDirectory(i.WINDOWS, "~")
		want := "~/path/foo"

		assert.Equal(t, want, got, "got %s want %s", got, want)
	})
	t.Run("custom linux path", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "~/path/foo")
		got, _ := i.GetDataDirectory(i.LINUX, "~")
		want := "~/path/foo"

		assert.Equal(t, want, got, "got %q want %q", got, want)
	})
}
