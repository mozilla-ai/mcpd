package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

type InitCmd struct {
	*cmd.BaseCmd
}

func NewInitCmd(log hclog.Logger) *cobra.Command {
	c := &InitCmd{
		BaseCmd: &cmd.BaseCmd{Logger: log},
	}

	cobraCommand := &cobra.Command{
		Use:   "init",
		Short: "Initializes the current directory as an mcpd project.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	return cobraCommand
}

func (c *InitCmd) longDescription() string {
	return fmt.Sprintf(
		"Initializes the current directory as an mcpd project, creating an %s configuration file. "+
			"This command sets up the basic structure required for an mcpd project.", flags.DefaultConfigFile)
}

func (c *InitCmd) run(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, "Initializing mcpd project in current directory...")

	cwd, err := os.Getwd()
	if err != nil {
		c.Logger.Error("Failed to get working directory", "error", err)
		return fmt.Errorf("error getting current directory: %w", err)
	}

	initFilePath := filepath.Join(cwd, flags.DefaultConfigFile)

	if err := config.InitConfigFile(initFilePath); err != nil {
		c.Logger.Error("Project initialization failed", "error", err)
		return fmt.Errorf("error initializing mcpd project: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s created successfully.\n", flags.DefaultConfigFile)

	return nil
}
