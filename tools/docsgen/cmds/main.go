//go:build docsgen_cli
// +build docsgen_cli

package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/mozilla-ai/mcpd/cmd"
	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

// main assumes it is run from the repository root.
func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "mcpd.docsgen",
		Level:  hclog.Info,
		Output: os.Stderr,
	})

	// docsPath is the path to the commands documentation, relative to the repository root.
	docsPath := "./docs/commands/"

	// Resolve paths against a portable default before constructing descriptions
	// and inherited flags. This process only generates documentation.
	if err := os.Setenv(flags.EnvRuntimeFile, "~/.config/mcpd/"+flags.DefaultRuntimeVarsFile); err != nil {
		logger.Error("failed to configure runtime path for docs generation", "error", err)
		return
	}

	for _, name := range []string{flags.EnvVarConfigFile, flags.EnvVarLogPath} {
		if err := os.Unsetenv(name); err != nil {
			logger.Error("failed to clear local path for docs generation", "variable", name, "error", err)
			return
		}
	}

	rootCmd, err := cmd.NewRootCmd(&cmd.RootCmd{BaseCmd: &internalcmd.BaseCmd{}})
	if err != nil {
		logger.Error("failed to create root command for docs generation", "error", err)
		return
	}

	if err = os.RemoveAll(docsPath); err != nil {
		logger.Error("failed to clear docs directory", "path", docsPath, "error", err)
		return
	}

	if err = os.MkdirAll(docsPath, 0o755); err != nil {
		logger.Error("failed to create docs directory", "path", docsPath, "error", err)
		return
	}

	usePortableCachePath(rootCmd)
	err = doc.GenMarkdownTree(rootCmd, docsPath)
	if err != nil {
		logger.Error("failed to generate CLI docs", "error", err)
		return
	}

	logger.Info("CLI docs generated", "path", docsPath)
}

func usePortableCachePath(command *cobra.Command) {
	if flag := command.Flags().Lookup("cache-dir"); flag != nil {
		flag.DefValue = "~/.cache/mcpd/registries"
	}
	for _, child := range command.Commands() {
		usePortableCachePath(child)
	}
}
