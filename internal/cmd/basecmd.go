package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/provider/mcpm"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// version should not be moved/modified without consulting the Makefile,
// as the path to this var is set on the LDFLAGS variable in the script.
var version = "dev" // Set at build time using -ldflags

// Version is used by other packages to retrieve the build version of mcpd.
func Version() string {
	return version
}

// AppName returns the name of the mcpd application.
func AppName() string {
	return "mcpd"
}

var _ registry.Builder = (*BaseCmd)(nil)

type BaseCmd struct {
	logger hclog.Logger
}

// SetLogger updates the command's logger
func (c *BaseCmd) SetLogger(logger hclog.Logger) {
	c.logger = logger
}

// Logger returns the current logger for the command
func (c *BaseCmd) Logger() (hclog.Logger, error) {
	if c.logger != nil {
		return c.logger, nil
	}

	logLevel := flags.LogLevel
	logPath := flags.LogPath

	// Configure logger output based on the log file path
	output := io.Discard // Default to discarding log output.
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file (%s): %w", logPath, err)
		} else {
			output = f
		}
	}

	c.logger = hclog.New(&hclog.LoggerOptions{
		Name:   AppName(),
		Level:  hclog.LevelFromString(logLevel),
		Output: output,
	})

	return c.logger, nil
}

func (c *BaseCmd) Build() (registry.PackageProvider, error) {
	logger, err := c.Logger()
	if err != nil {
		return nil, err
	}

	supportedRuntimes := c.MCPDSupportedRuntimes()
	opts := runtime.WithSupportedRuntimes(supportedRuntimes...)
	l := logger.Named("registry")

	mcpmRegistry, err := mcpm.NewRegistry(l, mcpm.ManifestURL, opts)
	if err != nil {
		// TODO: Handle tolerating some failed registries, as long as we can meet a minimum requirement.
		return nil, err
	}

	// NOTE: The order the registries are added here determines their precedence when searching and resolving packages.
	registries := []registry.PackageProvider{
		mcpmRegistry,
	}

	aggregator, err := registry.NewRegistry(l, registries...)
	if err != nil {
		return nil, err
	}

	return aggregator, nil
}

// MCPDSupportedRuntimes returns the runtimes that are supported by the mcpd application.
func (c *BaseCmd) MCPDSupportedRuntimes() []runtime.Runtime {
	return []runtime.Runtime{
		runtime.NPX,
		runtime.UVX,
	}
}
