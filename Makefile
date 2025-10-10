.PHONY: build build-dev build-linux build-linux-arm64 clean docs docs-cli docs-local docs-nav install lint local-down local-up test uninstall validate-registry check-licenses check-notice notice

MODULE_PATH := github.com/mozilla-ai/mcpd/v2

# /usr/local/bin is a common default for user-installed binaries
INSTALL_DIR := /usr/local/bin

# Target platform for local Docker builds (matches Docker buildx platform format)
TARGET_PLATFORM := linux/amd64

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

# The license types allowed to be imported by the project
ALLOWED_LICENSES := Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ZeroBSD,Unlicense

check-licenses:
	@echo "Checking licenses..."
	@go install github.com/google/go-licenses/v2@latest
	@set -e; \
	if go-licenses check ./... --ignore github.com/mozilla-ai/mcpd/v2 --allowed_licenses=$(ALLOWED_LICENSES); then \
		echo "✓ All licenses are allowed."; \
	else \
		echo "License check failed: some dependencies have disallowed licenses."; \
		exit 1; \
	fi

check-notice:
	@echo "Checking NOTICE..."
	@go install github.com/google/go-licenses/v2@latest
	@tmp=$$(mktemp); \
	trap "rm -f $$tmp" EXIT; \
	go-licenses report ./... --ignore github.com/mozilla-ai/mcpd/v2 --template build/licenses/notice.tpl > $$tmp; \
	if ! cmp -s NOTICE $$tmp; then \
		echo "NOTICE is out of date. Regenerate it with 'make notice'"; \
		exit 1; \
	else \
		echo "✓ NOTICE is up to date"; \
	fi

notice:
	@echo "Generating NOTICE..."
	@go install github.com/google/go-licenses/v2@latest
	@go-licenses report ./... --ignore github.com/mozilla-ai/mcpd/v2 --template build/licenses/notice.tpl > NOTICE
	@echo "✓ NOTICE generated"

lint: check-notice
	golangci-lint run --fix -v

test: lint
	go test ./...

validate-registry:
	@echo "Validating Mozilla AI registry against schema..."
	@go run -tags=validate_registry ./tools/validate/registry.go \
		internal/provider/mozilla_ai/data/schema.json \
		internal/provider/mozilla_ai/data/registry.json

build: lint
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
	@rm -rf $(TARGET_PLATFORM) # Remove any orphaned Docker build directories

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
	@echo "organizing binary for docker build"
	@mkdir -p $(TARGET_PLATFORM)
	@cp mcpd $(TARGET_PLATFORM)/mcpd
	@echo "starting mcpd container in detached state"
	@TARGETPLATFORM=$(TARGET_PLATFORM) docker compose up -d --build
	@echo "cleaning up temporary platform directory"
	@rm -rf $(TARGET_PLATFORM)

local-down:
	@echo "stopping mcpd container"
	@docker compose down
