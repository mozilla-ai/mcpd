package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/cmd/config"
	"github.com/mozilla-ai/mcpd-cli/v2/cmd/server"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

var version = "dev" // Set at build time using -ldflags

type RootCmd struct {
	*cmd.BaseCmd
}

func Execute() {
	logger, err := configureLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error executing root command: %s", err)
		os.Exit(1)
	}

	rootCmd := NewRootCmd(logger)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd(logger hclog.Logger) *cobra.Command {
	c := &RootCmd{
		BaseCmd: &cmd.BaseCmd{Logger: logger},
	}

	rootCmd := &cobra.Command{
		Use:          "mcpd <command> [args]",
		Short:        "'mcpd' CLI is the primary interface for developers to interact with mcpd.",
		Long:         c.longDescription(),
		SilenceUsage: true,
		Version:      version,
	}

	// Global flags
	flags.InitFlags(rootCmd.PersistentFlags())

	// Add top-level commands that are NOT part of a resource group
	rootCmd.AddCommand(NewInitCmd(logger))
	// TODO: Re-add commands:
	// rootCmd.AddCommand(listToolsCmd)
	// rootCmd.AddCommand(loginCmd)

	// Add commands from specific resource/service packages, they remain top-level commands in the CLI's usage.
	// TODO: Re-add daemon
	// rootCmd.AddCommand(server.NewDaemonCmd(logger))
	rootCmd.AddCommand(server.NewAddCmd(logger))
	rootCmd.AddCommand(server.NewRemoveCmd(logger))
	// TODO: Update to add: NewConfigCmd etc.
	rootCmd.AddCommand(config.Cmd)

	return rootCmd
}

func (c *RootCmd) longDescription() string {
	return `The 'mcpd' CLI is the primary interface for developers to interact with the
mcpd Control Plane, define their agent projects, and manage MCP server dependencies.`
}

func configureLogger() (hclog.Logger, error) {
	logPath := strings.TrimSpace(os.Getenv(flags.EnvVarLogPath))

	// If MCPD_LOG_PATH is not set, don't log anywhere.
	logOutput := io.Discard

	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file (%s): %w", logPath, err)
		}
		logOutput = f
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "mcpd",
		Level:  hclog.LevelFromString(getLogLevel()),
		Output: logOutput,
	})

	return logger, nil
}

func getLogLevel() string {
	lvl := strings.ToLower(os.Getenv(flags.EnvVarLogLevel))
	switch lvl {
	case "trace", "debug", "info", "warn", "error", "off":
		return lvl
	default:
		return "info"
	}
}
