package self //nolint: testpackage

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsHomebrewCellarPath pins which binary locations count as a Homebrew
// install for the purposes of self update's refusal — every prefix Homebrew
// itself uses on macOS Intel, macOS Apple Silicon and Linuxbrew, and nothing
// else.
func TestIsHomebrewCellarPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path string
		want bool
	}{
		"macOS Apple Silicon Cellar": {
			path: "/opt/homebrew/Cellar/gotpm/0.5.1/bin/gotpm",
			want: true,
		},
		"macOS Intel Cellar": {
			path: "/usr/local/Cellar/gotpm/0.5.1/bin/gotpm",
			want: true,
		},
		"Linuxbrew Cellar": {
			path: "/home/linuxbrew/.linuxbrew/Cellar/gotpm/0.5.1/bin/gotpm",
			want: true,
		},
		"curl|sh install under $HOME/.local/bin": {
			path: "/home/user/.local/bin/gotpm",
			want: false,
		},
		"go install under GOBIN": {
			path: "/home/user/go/bin/gotpm",
			want: false,
		},
		"Windows install, backslashes": {
			path: `C:\Users\user\.local\bin\gotpm.exe`,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isHomebrewCellarPath(tt.path))
		})
	}
}

// TestAssetFilterForMatchesGoReleaserNaming pins the filter against the
// exact asset names GoReleaser produces (see .goreleaser.yaml's archives
// name_template), for every OS/arch this project ships, and confirms it
// stays selective enough not to also match a neighboring platform's asset
// or an unrelated release file.
func TestAssetFilterForMatchesGoReleaserNaming(t *testing.T) {
	t.Parallel()

	tests := []struct {
		goos   string
		goarch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"_"+tt.goarch, func(t *testing.T) {
			t.Parallel()

			re := regexp.MustCompile(assetFilterFor(tt.goos, tt.goarch))

			want := "gotpm_v1.2.3_" + tt.goos + "_" + tt.goarch
			if tt.goos == "windows" {
				want += ".exe"
			}
			require.True(t, re.MatchString(want), "want match against %q", want)

			require.False(t, re.MatchString("checksums.txt"))
			require.False(t, re.MatchString("gotpm_v1.2.3_"+tt.goos+"_"+tt.goarch+"other"),
				"must not match past the intended arch, e.g. amd64 matching amd64le")
		})
	}
}
