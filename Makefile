.PHONY: build clean docs docs-cli docs-nav docs-local install test uninstall

MODULE_PATH := github.com/mozilla-ai/mcpd/v2

# /usr/local/bin is a common default for user-installed binaries
INSTALL_DIR := /usr/local/bin

# Get the version string dynamically
# This will be:
#   - e.g., "v1.0.0" if on a tag
#   - e.g., "v0.1.0-2-gabcdef123" if 2 commits past tag v0.1.0 (with hash abcdef123)
#   - e.g., "abcdef123-dirty" if on a commit and dirty
#   - e.g., "dev" if git is not available or no commits yet
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Linker flags for injecting version
# The path is MODULE_PATH/package.variableName
LDFLAGS := -X '$(MODULE_PATH)/cmd.version=$(VERSION)'

test:
	go test ./...

build:
	@echo "building mcpd (with flags: ${LDFLAGS})..."
	@go build -o mcpd -ldflags="${LDFLAGS}" .

install: build
	@# Copy the executable to the install directory
	@# Requires sudo if INSTALL_DIR is a system path like /usr/local/bin
	@cp mcpd $(INSTALL_DIR)/mcpd
	@echo "mcpd installed to $(INSTALL_DIR)/mcpd"

clean:
	@# Remove the built executable and any temporary files
	@rm -f mcpd # The executable itself
	@# Add any other build artifacts here if they accumulate (e.g., cache files)
	@echo "Build artifacts cleaned"

uninstall:
	@# Remove the installed executable from the system
	@# Requires sudo if INSTALL_DIR is a system path
	@rm -f $(INSTALL_DIR)/mcpd
	@echo "mcpd uninstalled from $(INSTALL_DIR)/mcpd"

## Runs MkDocs locally
docs: docs-local

## Runs MkDocs locally
docs-local: docs-nav
	@uv venv && \
		source .venv/bin/activate && \
		uv pip install mkdocs mkdocs-material && \
		uv run mkdocs serve

## Generates CLI markdown documentation
docs-cli:
	@go run -tags=docsgen_cli ./tools/docsgen/cli/cmds.go
	@echo "mcpd CLI command documentation generated"

## Updates mkdocs.yaml nav to match generated CLI docs
docs-nav: docs-cli
	@go run -tags=docsgen_nav ./tools/docsgen/cli/nav.go
	@echo "navigation updated for MkDocs site"