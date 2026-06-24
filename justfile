# Run commands or execute tasks related to the repository development
_default:
    @just --list

alias t := test
alias b := build
alias fmt := format

# Informations Embeded into Binary

LATEST_TAG := `git describe --tags --dirty --always --abbrev=0`
COMMIT_HASH := `git rev-parse --short HEAD`
BUILD_OS := `go env GOOS`
BUILD_ARCH := `go env GOARCH`
INSTALLER := env("INSTALLER", "source")

# Generic variables

GO_URL := `go list -m`
BIN_NAME := "gotpm"

# Build Flags

LDFLAGS := f"-s -w \
-X {{GO_URL}}/cmd.gitTag={{LATEST_TAG}} \
-X {{GO_URL}}/cmd.gitCommit={{COMMIT_HASH}} \
-X {{GO_URL}}/cmd.buildOS={{BUILD_OS}} \
-X {{GO_URL}}/cmd.buildARCH={{BUILD_ARCH}} \
-X {{GO_URL}}/cmd.installer={{INSTALLER}} "

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

# run the tests and inspect the code coverage
cover *args="./...":
    go test -coverprofile=c.out {{ args }}
    go tool cover -html=c.out

# run the go formatter
format:
    go fmt ./...
    golangci-lint fmt

# run the linter
lint:
    golangci-lint run --fix

# write the changelog from commit messages (https://git-cliff.org/)
changelog *args:
    git-cliff -o {{ args }}

_validate_semver version:
    @{{ if version =~ '^[0-9]+\.[0-9]+\.[0-9]+$' { "true" } else { error("invalid semver: '" + version + "' — expected major.minor.patch") } }}

_ensure_clean:
    @git diff --quiet
    @git diff --cached --quiet

_commit_and_tag version:
    git add CHANGELOG.md
    git commit -m "chore(release): bump version to {{ version }}"
    git tag -a "v{{ version }}"

# run CI checks: format, lint, vulns, vendor sync, test
ci:
    @files=$(gofmt -l . | grep -v '^vendor/'); if [ -n "$files" ]; then echo "unformatted files:"; echo "$files"; exit 1; fi
    golangci-lint fmt --diff
    golangci-lint run
    govulncheck ./...
    go mod vendor
    git diff --exit-code vendor/
    go test ./...

# make a new release (e.g. semver=0.1.2)
release semver: (_validate_semver semver) _ensure_clean ci
    @just changelog --tag {{ semver }}
    @just _commit_and_tag {{ semver }}
    @echo "{{ GREEN }}Release complete. Run 'git push && git push --tags'.{{ NORMAL }}"
