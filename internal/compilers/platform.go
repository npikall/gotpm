package compilers

import (
	"fmt"
	"runtime"
)

// targets are the target triples Typst publishes a compiler for, by
// GOOS/GOARCH. Linux builds are the static musl ones, except riscv64, which
// Typst only publishes for glibc.
var targets = map[string]string{
	"darwin/arm64":  "aarch64-apple-darwin",
	"darwin/amd64":  "x86_64-apple-darwin",
	"linux/amd64":   "x86_64-unknown-linux-musl",
	"linux/arm64":   "aarch64-unknown-linux-musl",
	"linux/arm":     "armv7-unknown-linux-musleabi",
	"linux/riscv64": "riscv64gc-unknown-linux-gnu",
	"windows/amd64": "x86_64-pc-windows-msvc",
	"windows/arm64": "aarch64-pc-windows-msvc",
}

// AssetName is the name of the release archive holding the compiler for a
// platform: a zip on windows, a tar.xz everywhere else.
func AssetName(goos, goarch string) (string, error) {
	target, ok := targets[goos+"/"+goarch]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, goos, goarch)
	}
	if goos == "windows" {
		return "typst-" + target + ".zip", nil
	}
	return "typst-" + target + ".tar.xz", nil
}

// The compiler's binary is typst.exe on windows, which Typst zips, and typst
// everywhere else, which Typst packs as a tar.xz.
const (
	windowsBinary = "typst.exe"
	unixBinary    = "typst"
)

// binaryName is the name of the compiler's binary on this platform.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return windowsBinary
	}
	return unixBinary
}

// Asset returns the archive of the release built for this platform.
func (r Release) Asset() (Asset, error) {
	name, err := AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return Asset{}, err
	}
	for _, asset := range r.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("%w: compiler %s has no %s for %s/%s",
		ErrUnsupportedPlatform, r.Version, name, runtime.GOOS, runtime.GOARCH)
}
