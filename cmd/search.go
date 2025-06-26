package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
)

type SearchCmd struct {
	*cmd.BaseCmd
	Version         string
	Runtime         string
	Tools           []string
	License         string
	Source          string
	registryBuilder registry.Builder
	packagePrinter  printer.Printer
}

func NewSearchCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	// Override printer options to show separator for search.
	if err = opts.Printer.SetOptions(printer.WithSeparator(true)); err != nil {
		return nil, err
	}

	c := &SearchCmd{
		BaseCmd:         baseCmd,
		registryBuilder: opts.RegistryBuilder,
		packagePrinter:  opts.Printer,
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

	cobraCommand.Flags().StringArrayVar(
		&c.Tools,
		"tool",
		nil,
		"Optional, specifies the tools the server must expose (can be repeated)",
	)

	cobraCommand.Flags().StringVar(
		&c.License,
		"license",
		"",
		"Optional, specify the license of the server package (e.g. MIT, Apache)",
	)

	cobraCommand.Flags().StringVar(
		&c.Source,
		"source",
		"",
		"Optional, specify the source registry of the server package (e.g. mcpm)",
	)

	return cobraCommand, nil
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
	if c.Tools != nil && len(c.Tools) > 0 {
		f["tools"] = strings.Join(c.Tools, ",")
	}
	if c.License != "" {
		f["license"] = c.License
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

	reg, err := c.registryBuilder.Build()
	if err != nil {
		return err
	}

	results, err := reg.Search(name, c.filters(), []options.SearchOption{options.WithSearchSource(c.Source)}...)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	if _, err = fmt.Fprintf(cmd.OutOrStdout(), "\n📦 Registry search results...\n"); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(cmd.OutOrStdout(), "\n────────────────────────────────────────────\n\n"); err != nil {
		return err
	}

	for _, pkg := range results {
		if err = c.packagePrinter.PrintPackage(pkg); err != nil {
			return err
		}
	}

	return nil
}
