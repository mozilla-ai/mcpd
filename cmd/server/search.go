package server

import (
	"fmt"
	"strings"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/spf13/cobra"
)

type SearchCmd struct {
	*cmd.BaseCmd
	Version string
	Runtime string
	Tool    string
}

func NewSearchCmd(baseCmd *cmd.BaseCmd) *cobra.Command {
	c := &SearchCmd{
		BaseCmd: baseCmd,
	}

	cobraCommand := &cobra.Command{
		Use:   "search <server-name>",
		Short: "Searches all configured registries for matching MCP servers.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCommand.Flags().StringVar(
		&c.Version,
		"version",
		"",
		"Optional, specify the version of the server package",
	)

	cobraCommand.Flags().StringVar(
		&c.Runtime,
		"runtime",
		"",
		"Optional, specify the runtime of the server package (e.g. uvx, npx)",
	)

	cobraCommand.Flags().StringVar(
		&c.Tool,
		"tool",
		"",
		"Optional, specify the tool the server must expose",
	)

	return cobraCommand
}

// longDescription returns the long version of the command description.
func (c *SearchCmd) longDescription() string {
	return `Searches all configured registries for matching MCP servers. Returns aggregated results for matches.`
}

func (c *SearchCmd) filters() map[string]string {
	f := make(map[string]string)

	if c.Version != "" {
		f["version"] = c.Version
	}
	if c.Runtime != "" {
		f["runtime"] = c.Runtime
	}
	if c.Tool != "" {
		f["tool"] = c.Tool
	}

	return f
}

func (c *SearchCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("name is required and cannot be empty")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	reg, err := c.CreateRegistry()
	if err != nil {
		return err
	}

	results, err := reg.Search(name, c.filters())
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
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

	fmt.Fprintf(cmd.OutOrStdout(), "\n📦 Registry search results...\n")
	fmt.Fprintf(cmd.OutOrStdout(), "\n────────────────────────────────────────────\n\n")
	for _, pkg := range results {
		fmt.Fprintf(cmd.OutOrStdout(), "  🆔 %s\n", pkg.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "  🏷️ Name: %s\n", pkg.DisplayName)
		fmt.Fprintf(cmd.OutOrStdout(), "  ℹ️ Description: %s\n", pkg.Description)
		if strings.TrimSpace(pkg.License) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  📄 License: %s\n", pkg.License)
		}
		if len(pkg.Runtimes) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  🏗️ Runtimes: %s\n", strings.Join(pkg.Runtimes, ", "))
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
			fmt.Fprintf(cmd.OutOrStdout(), "  📋 Args configurable via environment variables...\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  🌍 %s\n", strings.Join(pkg.ConfigurableEnvVars, ", "))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n────────────────────────────────────────────\n\n")
	}

	return nil
}
