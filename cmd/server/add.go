package server

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"
)

// AddCmd should be used to represent the 'add' command.
type AddCmd struct {
	*cmd.BaseCmd
	Version string
	Tools   []string
}

// NewAddCmd creates a newly configured (Cobra) command.
func NewAddCmd(baseCmd *cmd.BaseCmd) *cobra.Command {
	c := &AddCmd{
		BaseCmd: baseCmd,
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

	logger := c.Logger()

	reg, err := c.CreateRegistry()
	if err != nil {
		return err
	}

	pkg, err := reg.Get(name, types.WithVersion(c.Version))
	if err != nil {
		logger.Warn(
			"package retrieval from registry failed",
			"name", name,
			"version", c.Version,
			"tools", strings.Join(c.Tools, ","),
			"error", err,
		)
		return fmt.Errorf("⚠️ Failed to get package '%s@%s' from registry: %w", name, c.Version, err)
	}

	selectedTools, err := filterTools(c.Tools, pkg.Tools)
	if err != nil {
		return err
	}

	// TODO: Support 'runtime' flag
	// TODO: Sort out preference for runtimes
	var selectedRuntime string
	supportedRuntimes := registry.DefaultSupportedRuntimes()
	for k, v := range pkg.InstallationDetails {
		if _, ok := supportedRuntimes[types.Runtime(k)]; ok {
			// We'll always end up with a supported runtime,
			// and hopefully one of them is the recommended installation.
			selectedRuntime = k
			if v.Recommended {
				break
			}
		}
	}
	// We shouldn't end up in this situation, but just in case.
	if selectedRuntime == "" {
		return fmt.Errorf("no supported runtimes found for '%s'", pkg.Name)
	}

	version := "latest"
	if pkg.Version != "" {
		version = pkg.Version
	}

	runtimePackageVersion := fmt.Sprintf("%s::%s@%s", selectedRuntime, pkg.Name, version)

	entry := config.ServerEntry{
		Name:    pkg.ID,
		Package: runtimePackageVersion,
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
	logger.Debug("Server added", "name", name, "version", version, "tools", selectedTools)

	var tools string
	if len(selectedTools) > 0 {
		plural := ""
		if len(selectedTools) > 1 {
			plural = "s"
		}
		tools = fmt.Sprintf(", exposing tool%s: %s", plural, strings.Join(selectedTools, ", "))
	}

	// TODO: Refactor.
	getArgs := func(args map[string]types.ArgumentMetadata, required bool) []string {
		res := make([]string, 0, len(args))
		for name, meta := range args {
			if meta.Required == required {
				res = append(res, name)
			}
		}
		return res
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Added server '%s' (version: %s)%s\n", name, version, tools)
	fmt.Fprintf(cmd.OutOrStdout(), "  🆔 %s\n", pkg.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "  🏷️ Name: %s\n", pkg.DisplayName)
	fmt.Fprintf(cmd.OutOrStdout(), "  ℹ️ Description: %s\n", pkg.Description)
	if strings.TrimSpace(pkg.License) != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  📄 License: %s\n", pkg.License)
	}
	if len(pkg.Runtimes) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  🏗️ Runtimes: %s\n", strings.Join(pkg.Runtimes, ", "))
		fmt.Fprintf(cmd.OutOrStdout(), "  ✅ Selected: %s\n", selectedRuntime)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  ⚠️ Warning: No supported runtimes found in package description\n")
	}
	if len(pkg.Tools) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  🔨 Tools: %s\n", strings.Join(pkg.Tools, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  ⚠️ Warning: No tools found in package description\n")
	}
	if len(pkg.Arguments) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  ⚙️ Found startup args...\n")
		requiredArgs := getArgs(pkg.Arguments, true)
		if len(requiredArgs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  ❗ Required: %s\n", strings.Join(requiredArgs, ", "))
		}
		optionalArgs := getArgs(pkg.Arguments, false)
		if len(optionalArgs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  🔹️ Optional: %s\n", strings.Join(optionalArgs, ", "))
		}
	}
	if len(pkg.ConfigurableEnvVars) > 0 {
		// 🌍
		fmt.Fprintf(cmd.OutOrStdout(), "  📋 Args configurable via environment variables (mcpd config set-env)...\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  🌍 %s\n", strings.Join(pkg.ConfigurableEnvVars, ", "))
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
