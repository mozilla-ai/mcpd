//go:build docsgen_cli
// +build docsgen_cli

package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra/doc"

	"github.com/mozilla-ai/mcpd/v2/cmd"
	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
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

	err = doc.GenMarkdownTree(rootCmd, docsPath)
	if err != nil {
		logger.Error("failed to generate CLI docs", "error", err)
		return
	}

	logger.Info("CLI docs generated", "path", docsPath)
}
