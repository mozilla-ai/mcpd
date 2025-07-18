package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type InitCmd struct {
	*cmd.BaseCmd
	cfgInitializer config.Initializer
}

func NewInitCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &InitCmd{
		BaseCmd:        baseCmd,
		cfgInitializer: opts.ConfigInitializer,
	}

	cobraCommand := &cobra.Command{
		Use:   "init",
		Short: "Initializes the current directory as an `mcpd` project",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	return cobraCommand, nil
}

func (c *InitCmd) longDescription() string {
	return fmt.Sprintf(
		"Initializes the current directory as an `mcpd` project, creating a %s configuration file.\n\n"+
			"This command sets up the basic structure required for an `mcpd` project.\n\n"+
			"The configuration file path can be overridden using the `--%s` flag or the `%s` environment variable",
		flags.DefaultConfigFile,
		flags.FlagNameConfigFile,
		flags.EnvVarConfigFile,
	)
}

func (c *InitCmd) run(cmd *cobra.Command, _ []string) error {
	logger, err := c.Logger()
	if err != nil {
		return err
	}

	var initFilePath string

	// If the config file flag just has the default value, we're expecting to create it in the current working directory.
	if flags.ConfigFile == flags.DefaultConfigFile {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"📄 Using default config file: '%s' in the current directory\n", flags.DefaultConfigFile,
		); err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			logger.Error("Failed to get working directory", "error", err)
			return fmt.Errorf("error getting current directory: %w", err)
		}
		initFilePath = filepath.Join(cwd, flags.DefaultConfigFile)
	} else {
		initFilePath = flags.ConfigFile
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"🚀 Initializing mcpd project at: %s\n", initFilePath,
	); err != nil {
		return err
	}
	if err := c.cfgInitializer.Init(initFilePath); err != nil {
		logger.Error("Project initialization failed", "error", err)
		return fmt.Errorf("error initializing mcpd project: %w", err)
	}
	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✅ Config file created: %s\n", initFilePath,
	); err != nil {
		return err
	}

	return nil
}
