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
	IsOfficial      bool
	registryBuilder registry.Builder
	packagePrinter  output.Printer[packages.Package]
}

func NewSearchCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	pkgPrinter := printer.NewPackagePrinter()

	c := &SearchCmd{
		BaseCmd:         baseCmd,
		Format:          internalcmd.FormatText, // Default to plain text
		registryBuilder: opts.RegistryBuilder,
		packagePrinter:  printer.NewPackageResultsPrinter(pkgPrinter),
	}

	cobraCommand := &cobra.Command{
		Use:   "search [server-name]",
		Short: "Searches all configured registries for matching MCP servers",
		Long: fmt.Sprintf("Searches all configured registries for matching MCP servers, "+
			"when name is not specified, the wildcard character (`%s`) is used. "+
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

	cobraCommand.Flags().BoolVar(
		&c.IsOfficial,
		"official",
		false,
		"Optional, only official server packages are included in results",
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
	if len(c.Tools) > 0 {
		f["tools"] = strings.Join(c.Tools, ",")
	}
	if len(c.Tags) > 0 {
		f["tags"] = strings.Join(c.Tags, ",")
	}
	if len(c.Categories) > 0 {
		f["categories"] = strings.Join(c.Categories, ",")
	}
	if c.License != "" {
		f["license"] = c.License
	}
	if c.IsOfficial {
		f["isOfficial"] = "true"
	}

	return f
}

func (c *SearchCmd) run(cmd *cobra.Command, args []string) (err error) {
	handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.Format, c.packagePrinter)
	if err != nil {
		return err
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

	return handler.HandleResults(results...)
}
