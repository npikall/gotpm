package compilers_test

import (
	"testing"

	"github.com/npikall/gotpm/internal/compilers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetName_MapsPlatformsToTypstTargets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		goos, goarch string
		want         string
	}{
		{"darwin", "arm64", "typst-aarch64-apple-darwin.tar.xz"},
		{"darwin", "amd64", "typst-x86_64-apple-darwin.tar.xz"},
		{"linux", "amd64", "typst-x86_64-unknown-linux-musl.tar.xz"},
		{"linux", "arm64", "typst-aarch64-unknown-linux-musl.tar.xz"},
		{"linux", "arm", "typst-armv7-unknown-linux-musleabi.tar.xz"},
		{"linux", "riscv64", "typst-riscv64gc-unknown-linux-gnu.tar.xz"},
		{"windows", "amd64", "typst-x86_64-pc-windows-msvc.zip"},
		{"windows", "arm64", "typst-aarch64-pc-windows-msvc.zip"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			t.Parallel()
			got, err := compilers.AssetName(tt.goos, tt.goarch)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAssetName_RefusesAPlatformTypstShipsNoBinaryFor(t *testing.T) {
	t.Parallel()
	_, err := compilers.AssetName("freebsd", "amd64")

	require.ErrorIs(t, err, compilers.ErrUnsupportedPlatform)
	assert.ErrorContains(t, err, "freebsd/amd64")
}
