# Run commands or execute tasks related to the repository development
_default:
    @just --list

alias t := test
alias fmt := format

# Informations Embeded into Binary

LATEST_TAG := `git describe --tags --dirty --always --abbrev=0`
COMMIT_HASH := `git rev-parse --short HEAD`
BUILD_OS := `go env GOOS`
BUILD_ARCH := `go env GOARCH`

# Generic variables

GO_URL := `go list -m`
BIN_NAME := "gotpm"

# Build Flags

LDFLAGS := f"-s -w \
-X {{GO_URL}}/cmd.gitTag={{LATEST_TAG}} \
-X {{GO_URL}}/cmd.gitCommit={{COMMIT_HASH}} \
-X {{GO_URL}}/cmd.buildOS={{BUILD_OS}} \
-X {{GO_URL}}/cmd.buildARCH={{BUILD_ARCH}} "

# show version information
info:
    @echo "Latest git tag: {{ LATEST_TAG }}"
    @echo "Latest commit hash: {{ COMMIT_HASH }}"

# build the binary
build:
    go build -ldflags="{{ LDFLAGS }}" -o {{ BIN_NAME }}

# install the binary locally
install:
    go install -ldflags="{{ LDFLAGS }}"

# run the test suite
test:
    go test ./...

# run the go formatter
format:
    go fmt ./...

# write the changelog from commit messages (https://git-cliff.org/)
changelog *args:
    git-cliff -o {{ args }}

_ensure_clean:
    @git diff --quiet
    @git diff --cached --quiet

_commit_and_tag version:
    git add CHANGELOG.md
    git commit -m "chore(release): bump version to {{ version }}"
    git tag -a "v{{ version }}"

# make a new release (e.g. semver=0.1.2)
release semver:
    @just _ensure_clean
    @just changelog --tag {{ semver }}
    @just _commit_and_tag {{ semver }}
    @echo "{{ GREEN }}Release complete. Run 'git push && git push --tags'.{{ NORMAL }}"
