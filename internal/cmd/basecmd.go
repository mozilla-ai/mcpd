package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry"
)

type BaseCmd struct {
	logger hclog.Logger
}

// SetLogger updates the command's logger
func (c *BaseCmd) SetLogger(logger hclog.Logger) {
	c.logger = logger
}

// Logger returns the current logger for the command
func (c *BaseCmd) Logger() hclog.Logger {
	if c.logger != nil {
		return c.logger
	}

	// Get log level from flags first, then environment, then default
	logLevel := flags.LogLevel
	if logLevel == "" {
		logLevel = strings.ToLower(os.Getenv(flags.EnvVarLogLevel))
		if logLevel == "" {
			logLevel = flags.DefaultLogLevel
		}
	}

	// Get log path from flags first, then environment
	logPath := flags.LogPath
	if logPath == "" {
		logPath = strings.TrimSpace(os.Getenv(flags.EnvVarLogPath))
	}

	// Configure logger output
	var output io.Writer = io.Discard // os.Stderr
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file (%s): %v, using stderr\n", logPath, err)
		} else {
			output = f
		}
	}

	// Using flags/env for fallback logger
	c.logger = hclog.New(&hclog.LoggerOptions{
		Name:   "mcpd-default",
		Level:  hclog.LevelFromString(logLevel),
		Output: output,
	})

	return c.logger
}

func (c *BaseCmd) CreateRegistry() (registry.PackageResolver, error) {
	l := c.Logger().Named("registry")

	mcpm, err := registry.NewMCPMRegistry(l, "https://mcpm.sh/api/servers.json") // TODO: hard coded URL
	if err != nil {
		return nil, err
	}

	registries := []registry.PackageResolver{
		mcpm,
		// TODO: Add more registries...
	}

	aggregator, err := registry.NewRegistry(l, registries...)
	if err != nil {
		return nil, err
	}

	return aggregator, nil
}
