package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
)

type SearchCmd struct {
	*internalcmd.BaseCmd
	Version         string
	Runtime         string
	Tools           []string
	Tags            []string
	Categories      []string
	License         string
	Source          string
	Format          internalcmd.OutputFormat
	registryBuilder registry.Builder
	packagePrinter  printer.Printer
}

func NewSearchCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
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
		Format:          internalcmd.FormatText, // Default to plain text
		registryBuilder: opts.RegistryBuilder,
		packagePrinter:  opts.Printer,
	}

	cobraCommand := &cobra.Command{
		Use:   "search [server-name]",
		Short: "Searches all configured registries for matching MCP servers",
		Long: fmt.Sprintf("Searches all configured registries for matching MCP servers, "+
			"the wildcard '%s' is the default when name is not specified. "+
			"Returns aggregated results from all configured registries", options.WildcardCharacter),
		RunE: c.run,
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
		"Optional, specify a partial match for the license of the server package (e.g. MIT, Apache)",
	)

	cobraCommand.Flags().StringVar(
		&c.Source,
		"source",
		"",
		"Optional, specify the source registry of the server package (e.g. mcpm)",
	)

	cobraCommand.Flags().StringArrayVar(
		&c.Tags,
		"tag",
		nil,
		"Optional, specify a partial match for required tags (can be repeated)",
	)

	cobraCommand.Flags().StringArrayVar(
		&c.Categories,
		"category",
		nil,
		"Optional, specify a partial match for required categories (can be repeated)",
	)

	allowed := internalcmd.AllowedOutputFormats()
	cobraCommand.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowed.String()),
	)

	return cobraCommand, nil
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
	if c.Tags != nil && len(c.Tags) > 0 {
		f["tags"] = strings.Join(c.Tags, ",")
	}
	if c.Categories != nil && len(c.Categories) > 0 {
		f["categories"] = strings.Join(c.Categories, ",")
	}
	if c.License != "" {
		f["license"] = c.License
	}

	return f
}

func (c *SearchCmd) run(cmd *cobra.Command, args []string) (err error) {
	// Configure the handler based on the requested format.
	var handler output.Handler[packages.Package]
	switch c.Format {
	case internalcmd.FormatJSON:
		handler = output.NewJSONHandler[packages.Package](cmd.OutOrStdout(), 2)
	case internalcmd.FormatYAML:
		handler = output.NewYAMLHandler[packages.Package](cmd.OutOrStdout(), 2)
	case internalcmd.FormatText:
		pkgListPrinter := printer.NewPackageListPrinter(c.packagePrinter)
		handler = output.NewTextHandler[packages.Package](cmd.OutOrStdout(), pkgListPrinter)
	default:
		return fmt.Errorf("unexpected error, no handler for output format: %s", c.Format)
	}

	// Name not required, default to the wildcard.
	name := options.WildcardCharacter
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		name = strings.TrimSpace(args[0])
	}

	reg, err := c.registryBuilder.Build()
	if err != nil {
		return handler.HandleError(err)
	}

	results, err := reg.Search(name, c.filters(), []options.SearchOption{options.WithSearchSource(c.Source)}...)
	if err != nil {
		return handler.HandleError(err)
	}

	return handler.HandleResults(results)
}
