package export

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type Cmd struct {
	*cmd.BaseCmd
	Format         cmd.ExportFormat
	ContractOutput string
	ContextOutput  string
	cfgLoader      config.Loader
	ctxLoader      context.Loader
}

func NewCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &Cmd{
		BaseCmd:   baseCmd,
		Format:    cmd.FormatDotEnv, // Default to dotenv
		cfgLoader: opts.ConfigLoader,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "export",
		Short: "Exports current configuration, generating a pair of safe and portable configuration files",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCmd.Flags().StringVar(
		&c.ContextOutput,
		"context-output",
		"secrets.prod.toml",
		"Optional, specify the output path for the templated execution context config file",
	)

	cobraCmd.Flags().StringVar(
		&c.ContractOutput,
		"contract-output",
		".env",
		"Optional, specify the output path for the templated environment file",
	)

	allowed := cmd.AllowedExportFormats()
	cobraCmd.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the format of the contract output file (one of: %s)", allowed.String()),
	)

	return cobraCmd, nil
}

func (c *Cmd) longDescription() string {
	return "Exports current configuration, generating a pair of safe and portable configuration files.\n\n" +
		"Using a project's required configuration (e.g. `.mcpd.toml`) and the locally configured runtime values from the " +
		"execution context file (e.g. `~/.config/mcpd/secrets.dev.toml`), the export command outputs two files:\n\n" +
		"Environment Contract:\n\n" +
		"Lists all required and configured environment variables as secure, namespaced placeholders:\n\n" +
		"`MCPD__{SERVER_NAME}__{ENV_VAR}` - Creates placeholders for command line arguments to be populated with env vars\n\n" +
		"`MCPD__{SERVER_NAME}__ARG_{ARG_NAME}` - This file is intended for the platform operator or CI/CD system\n\n" +
		"Portable Execution Context:\n\n" +
		"- A new secrets `.toml` file that defines sanitized runtime args and env sections for each server using the " +
		"placeholders aligned with the environment contract\n" +
		"- These files are safe to check into version control if required.\n\n" +
		"This allows running an mcpd project in any environment, cleanly separating the configuration structure " +
		"from the secret values"
}

func (c *Cmd) run(cmd *cobra.Command, _ []string) error {
	contextPath := c.ContextOutput
	// contractPath := c.ContractOutput

	if err := exportPortableExecutionContext(c.ctxLoader, flags.RuntimeFile, contextPath); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Portable Execution Context exported: %s\n", contextPath,
	); err != nil {
		return err
	}

	// Export 'Environment Contract'
	// TODO: export to contractPath based on format
	// fmt.Fprintf(cmd.OutOrStdout(), "✓ Environment Contract exported: %s\n", contractPath)

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Export completed successfully!\n",
	); err != nil {
		return err
	}

	return nil
}

func exportPortableExecutionContext(loader context.Loader, src string, dest string) error {
	mod, err := loader.Load(src)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	exp, ok := mod.(context.Exporter)
	if !ok {
		return fmt.Errorf("execution context config does not support exporting")
	}

	return exp.Export(dest)
}
