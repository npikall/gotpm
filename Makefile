.PHONY: build test format install help changelog changelog-auto

LATEST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
HEAD_TAG := $(shell git describe --tags --exact-match 2>/dev/null)

VERSION := $(if $(HEAD_TAG),$(HEAD_TAG),"$(LATEST_TAG)-dev")
LDFLAGS := -ldflags="-s -w \
	-X github.com/npikall/gotpm/cmd.GoTPMVersion=$(VERSION)"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test:  ## run the test suite
	go test ./...

format:  ## run the go formatter
	go fmt ./...

build:  ## build the binary (optimized)
	go build $(LDFLAGS) -o gotpm

changelog:  ## update the changelog (github.com/pawamoy/git-changelog)
	git-changelog --git-trailers -io CHANGELOG.md --bump="$(shell git-changelog --bumped-version)" --convention angular --versioning semver -Z

changelog-auto:  ## update the changelog (might write different version the 'make changelog')
	git-changelog --git-trailers -io CHANGELOG.md --bump="auto" --convention angular --versioning semver -Z

release: changelog ## update the Changelog and commit
	git add CHANGELOG.md
	git commit -m "chore(release): bump version and log the changes"
	@echo "Run 'git tag' and the 'git push && git push --tags'"

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
