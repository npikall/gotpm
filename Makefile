.PHONY: build test format install help

VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test:  ## run the test suite
	go test ./...

format:  ## run the go formatter
	go fmt ./...

build:  # build the binary
	go build -o gotpm

install: build  ## install
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
