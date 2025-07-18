.PHONY: build build-dev build-linux build-linux-arm64 clean docs docs-cli docs-local docs-nav install local-down local-up test uninstall

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

# Get commit hash and date
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags for injecting version and optimizations
# The path is MODULE_PATH/package.variableName
LDFLAGS := -s -w -X '$(MODULE_PATH)/internal/cmd.version=$(VERSION)' \
				-X '$(MODULE_PATH)/internal/cmd.commit=$(COMMIT)' \
				-X '$(MODULE_PATH)/internal/cmd.date=$(DATE)'

# Build flags for optimization
BUILDFLAGS := -trimpath

test:
	go test ./...

build:
	@echo "building mcpd (version: $(VERSION), commit: $(COMMIT))..."
	@go build $(BUILDFLAGS) -o mcpd -ldflags="$(LDFLAGS)" .

build-linux:
	@echo "building mcpd for amd64/linux (version: $(VERSION), commit: $(COMMIT))..."
	@GOOS=linux GOARCH=amd64 go build $(BUILDFLAGS) -o mcpd -ldflags="$(LDFLAGS)" .

build-linux-arm64:
	@echo "building mcpd for arm64/linux (version: $(VERSION), commit: $(COMMIT))..."
	@GOOS=linux GOARCH=arm64 go build $(BUILDFLAGS) -o mcpd -ldflags="$(LDFLAGS)" .

# For development builds without optimizations (for debugging)
build-dev:
	@echo "building mcpd for development (version: $(VERSION), commit: $(COMMIT))..."
	@go build -o mcpd -ldflags="-X '$(MODULE_PATH)/internal/cmd.version=$(VERSION)' \
		-X '$(MODULE_PATH)/internal/cmd.commit=$(COMMIT)' \
		-X '$(MODULE_PATH)/internal/cmd.date=$(DATE)'" .

install: build
	@# Copy the executable to the install directory
	@# Requires sudo if INSTALL_DIR is a system path like /usr/local/bin
	@echo "installing mcpd to $(INSTALL_DIR)..."
	@cp mcpd $(INSTALL_DIR)/mcpd
	@chmod +x $(INSTALL_DIR)/mcpd

clean:
	@# Remove the built executable and any temporary files
	@echo "cleaning up local build artifacts..."
	@rm -f mcpd # The executable itself

uninstall:
	@# Remove the installed executable from the system
	@# Requires sudo if INSTALL_DIR is a system path
	@echo "uninstalling mcpd from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/mcpd

# Runs MkDocs locally
docs: docs-local

# Runs MkDocs locally
docs-local: docs-nav
	@uv venv && \
		source .venv/bin/activate && \
		uv pip install mkdocs mkdocs-material && \
		uv run mkdocs serve

# Generates CLI markdown documentation
docs-cli:
	@go run -tags=docsgen_cli ./tools/docsgen/cli/cmds.go
	@echo "mcpd CLI command documentation generated"

## Updates mkdocs.yaml nav to match generated CLI docs
docs-nav: docs-cli
	@go run -tags=docsgen_nav ./tools/docsgen/cli/nav.go
	@echo "navigation updated for MkDocs site"

local-up: build-linux
	@echo "starting mcpd container in detached state"
	@docker compose up -d --build

local-down:
	@echo "stopping mcpd container"
	@docker compose down
