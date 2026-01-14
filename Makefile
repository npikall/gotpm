.PHONY: build test format install help changelog changelog-auto installer-script

GIT_TAG    := $(shell git describe --tags --abbrev=0 2>/dev/null)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

VERSION := $(if $(GIT_TAG),$(GIT_TAG),dev)
COMMIT  := $(if $(GIT_COMMIT),$(GIT_COMMIT),unknown)

LDFLAGS := -ldflags="-s -w \
	-X github.com/npikall/gotpm/cmd.selfVersion=$(VERSION) \
	-X github.com/npikall/gotpm/cmd.selfCommit=$(COMMIT)"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test:  ## run the test suite
	go test ./...

format:  ## run the go formatter
	go fmt ./...

build:  ## build the binary (optimized)
	go build $(LDFLAGS) -o gotpm

installer-script:
	sed 's/{{VERSION}}/${{ github.ref_name }}/g' install.sh.tmpl > install.sh

changelog:  ## update the changelog (github.com/pawamoy/git-changelog)
	git-changelog --git-trailers -io CHANGELOG.md --bump="$(shell git-changelog --bumped-version)" --convention angular --versioning semver -Z

changelog-auto:  ## update the changelog (might write different version the 'make changelog')
	git-changelog --git-trailers -io CHANGELOG.md --bump="auto" --convention angular --versioning semver -Z

install: build  ## install to either $HOME/.local/bin or $HOME/.bin or $HOME/bin
	@INSTALL_DIR=""; \
	if command -v gotpm >/dev/null 2>&1; then \
		INSTALL_DIR=$$(dirname $$(which gotpm)); \
	else \
		for dir in "$$HOME/.local/bin" "$$HOME/.bin" "$$HOME/bin"; do \
			if [ -d "$$dir" ] && echo "$$PATH" | tr ':' '\n' | grep -qx "$$dir"; then \
				INSTALL_DIR="$$dir"; \
				break; \
			fi; \
		done; \
	fi; \
	if [ -z "$$INSTALL_DIR" ]; then \
		echo "error: no suitable install directory found on PATH"; \
		echo "hint: create ~/.local/bin and add it to your PATH"; \
		exit 1; \
	fi; \
	cp gotpm "$$INSTALL_DIR/gotpm"; \
	echo "installed to $$INSTALL_DIR/gotpm"
