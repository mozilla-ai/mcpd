package server

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/discover"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

// AddCmd should be used to represent the 'add' command.
type AddCmd struct {
	*cmd.BaseCmd
	Version string
	Tools   []string
}

// NewAddCmd creates a newly configured (Cobra) command.
func NewAddCmd(logger hclog.Logger) *cobra.Command {
	c := &AddCmd{
		BaseCmd: &cmd.BaseCmd{Logger: logger},
	}

	cobraCommand := &cobra.Command{
		Use:   "add <server-name>",
		Short: "Adds an MCP server dependency to the project.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCommand.Flags().StringVar(
		&c.Version,
		"version",
		"latest",
		"Specify the version of the server package",
	)
	cobraCommand.Flags().StringArrayVar(
		&c.Tools,
		"tool",
		nil,
		"Optional, when specified limits the available tools on the server (can be repeated)",
	)

	return cobraCommand
}

// longDescription returns the long version of the command description.
func (c *AddCmd) longDescription() string {
	return `Adds an MCP server dependency to the project. 
mcpd will search the registry for the server and attempt to return information on the version specified, 
or 'latest' if no version specified.`
}

// run is configured (via NewAddCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *AddCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// TODO: Fix this later.
	// Tweak the package name to fit with the general format in PyPi...
	packageName := fmt.Sprintf("mcp-server-%s", name)

	// PyPI mode - discover the package information
	discoveryResult, err := discover.DiscoverPackage(packageName, c.Version)
	if err != nil {
		c.Logger.Warn(
			"PyPI discovery failed",
			"name", name,
			"package-name", packageName,
			"version", c.Version,
			"error", err,
		)
		return fmt.Errorf("⚠️ PyPI discovery failed for package '%s@%s': %w", packageName, c.Version, err)
	}

	selectedTools, err := filterTools(c.Tools, discoveryResult.FoundTools)
	if err != nil {
		return err
	}

	entry := config.ServerEntry{
		Name:    name,
		Package: fmt.Sprintf("pypi::%s@%s", packageName, discoveryResult.Version),
		Tools:   selectedTools,
	}

	cfg, err := config.NewConfig(flags.ConfigFile)
	if err != nil {
		return err
	}

	err = cfg.AddServer(entry)
	if err != nil {
		return err
	}

	// TODO: Handle prompting for any required configuration for this server and securely storing it.

	// User-friendly output + logging
	c.Logger.Debug("Server added", "name", name, "version", discoveryResult.Version, "tools", selectedTools)

	var tools string
	if len(selectedTools) > 0 {
		plural := ""
		if len(selectedTools) > 1 {
			plural = "s"
		}
		tools = fmt.Sprintf(", exposing only tool%s: %s", plural, strings.Join(selectedTools, ", "))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Added server '%s' (version: %s)%s\n", name, discoveryResult.Version, tools)

	// Add PyPI discovery information if available
	fmt.Fprintf(cmd.OutOrStdout(), "\n📦 PyPI package information...\n")

	if len(discoveryResult.FoundStartupArgs) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  ⚙️ Found startup args: %s\n",
			strings.Join(discoveryResult.FoundStartupArgs, ", "))
	}

	if discoveryResult.FoundEnvVars {
		fmt.Fprintf(cmd.OutOrStdout(), "  ℹ️ This package may use environment variables for configuration\n")
	}

	if len(discoveryResult.FoundTools) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  🔨 Found tools: %s\n", strings.Join(discoveryResult.FoundTools, ", "))

		// Check if the requested tools exist in the discovered tools
		missingTools := discover.ValidateTools(c.Tools, discoveryResult.FoundTools)
		if len(missingTools) > 0 {
			fmt.Fprintf(cmd.OutOrStderr(), "  ⚠️ Warning: Requested tools not found in package description (have been ignored): %s\n",
				strings.Join(missingTools, ", "))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  ⚠️ Warning: No tools found in package description\n")
	}

	return nil
}

func filterTools(requested, discovered []string) ([]string, error) {
	if len(requested) == 0 {
		return discovered, nil
	}

	foundSet := make(map[string]struct{}, len(discovered))
	for _, tool := range discovered {
		foundSet[tool] = struct{}{}
	}

	var result []string
	for _, tool := range requested {
		if _, ok := foundSet[tool]; ok {
			result = append(result, tool)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("none of the requested tools were found")
	}

	return result, nil
}
