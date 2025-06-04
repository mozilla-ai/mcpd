package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
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
			"This command sets up the basic structure required for an mcpd project.", flags.ConfigFile)
}

func (c *InitCmd) run(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, "Initializing mcpd project in current directory...")

	cwd, err := os.Getwd()
	if err != nil {
		c.Logger.Error("Failed to get working directory", "error", err)
		return fmt.Errorf("error getting current directory: %w", err)
	}

	if err := initializeProject(cwd); err != nil {
		c.Logger.Error("Project initialization failed", "error", err)
		return fmt.Errorf("error initializing mcpd project: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s created successfully.\n", flags.ConfigFile)

	return nil
}

func initializeProject(path string) error {
	if _, err := os.Stat(flags.ConfigFile); err == nil {
		return fmt.Errorf("%s already exists", flags.ConfigFile)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %w", flags.ConfigFile, err)
	}

	// TODO: Look at off-loading the data structure to the internal/config package
	content := `servers = []`

	if err := os.WriteFile(flags.ConfigFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", flags.ConfigFile, err)
	}

	return nil
}
