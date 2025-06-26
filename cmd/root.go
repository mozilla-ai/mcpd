package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
)

var version = "dev" // Set at build time using -ldflags

// createCmdFunc aliases the signature for a new command function.
type createCmdFunc func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error)

type RootCmd struct {
	*cmd.BaseCmd
}

// Global variable to hold the root command instance
var rootCmdInstance *RootCmd

func Execute() {
	// Create the root command instance
	rootCmdInstance = &RootCmd{
		BaseCmd: &cmd.BaseCmd{},
	}

	// Create cobra command.
	rootCmd, err := NewRootCmd(rootCmdInstance)
	if err != nil {
		// TODO: Handle top level error.
	}

	// Add hook to update loggers after flag parsing
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Configure logger with parsed flags
		logger, err := configureLogger()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error configuring logger: %s\n", err)
			os.Exit(1)
		}

		// Update the root command instance
		rootCmdInstance.SetLogger(logger)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd(c *RootCmd) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:          "mcpd <command> [sub-command] [args]",
		Short:        "'mcpd' CLI is the primary interface for developers to interact with mcpd.",
		Long:         c.longDescription(),
		SilenceUsage: true,
		Version:      version,
	}

	// Global flags
	flags.InitFlags(rootCmd.PersistentFlags())

	// Add top-level commands
	fns := []createCmdFunc{
		NewInitCmd,
		NewSearchCmd,
		NewAddCmd,
		NewRemoveCmd,
		NewDaemonCmd,
		NewConfigCmd,
	}

	for _, fn := range fns {
		p, err := printer.NewPrinter(rootCmd.OutOrStdout())
		if err != nil {
			return nil, err
		}

		opts := []options.CmdOption{
			options.WithPrinter(p),
			options.WithRegistryBuilder(c.BaseCmd),
		}

		tempCmd, err := fn(c.BaseCmd, opts...)
		if err != nil {
			return nil, err
		}
		rootCmd.AddCommand(tempCmd)
	}

	return rootCmd, nil
}

// TODO: Remove and call RootCmd.Logger()?
func configureLogger() (hclog.Logger, error) {
	// Use flags first, then fall back to env vars
	logPath := flags.LogPath
	if logPath == "" {
		logPath = strings.TrimSpace(os.Getenv(flags.EnvVarLogPath))
	}

	// If log path is empty, don't log to a file
	logOutput := io.Discard

	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file (%s): %w", logPath, err)
		}
		logOutput = f
	}

	// Use flags first, then fall back to env vars
	logLevel := flags.LogLevel
	if logLevel == "" {
		logLevel = strings.ToLower(os.Getenv(flags.EnvVarLogLevel))
		if logLevel == "" {
			logLevel = flags.DefaultLogLevel
		}
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "mcpd",
		Level:  hclog.LevelFromString(logLevel),
		Output: logOutput,
	})

	return logger, nil
}

func (c *RootCmd) longDescription() string {
	return `The 'mcpd' CLI is the primary interface for developers to interact with the
mcpd Control Plane, define their agent projects, and manage MCP server dependencies.`
}
