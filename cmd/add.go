package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	regopts "github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// AddCmd should be used to represent the 'add' command.
type AddCmd struct {
	*internalcmd.BaseCmd
	Version         string
	Tools           []string
	Runtime         string
	Source          string
	Format          internalcmd.OutputFormat
	cfgLoader       config.Loader
	packagePrinter  output.Printer[config.ServerEntry]
	registryBuilder registry.Builder
}

// NewAddCmd creates a newly configured (Cobra) command.
func NewAddCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &AddCmd{
		BaseCmd:         baseCmd,
		Format:          internalcmd.FormatText, // Default to plain text
		cfgLoader:       opts.ConfigLoader,
		packagePrinter:  &printer.ServerEntryPrinter{},
		registryBuilder: opts.RegistryBuilder,
	}

	cobraCommand := &cobra.Command{
		Use:   "add <server-name>",
		Short: "Adds an MCP server dependency to the project",
		Long: "Adds an MCP server dependency to the project. " +
			"`mcpd` will search the registry for the named server and attempt to return information " +
			"on the version specified, or 'latest' if no version specified", // TODO: Remove 'latest' reference when we have our own registry
		RunE: c.run,
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

	cobraCommand.Flags().StringVar(
		&c.Runtime,
		"runtime",
		"",
		"Optional, specify the runtime of the server package (e.g. `uvx`, `npx`)",
	)

	cobraCommand.Flags().StringVar(
		&c.Source,
		"source",
		"",
		"Optional, specify the source registry of the server package (e.g. `mcpm`)",
	)

	allowed := internalcmd.AllowedOutputFormats()
	cobraCommand.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowed.String()),
	)

	return cobraCommand, nil
}

// run is configured (via NewAddCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *AddCmd) run(cmd *cobra.Command, args []string) error {
	handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.Format, c.packagePrinter)
	if err != nil {
		return err
	}

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return handler.HandleError(fmt.Errorf("server name is required and cannot be empty"))
	}

	name := strings.TrimSpace(args[0])

	logger, err := c.Logger()
	if err != nil {
		return handler.HandleError(err)
	}

	reg, err := c.registryBuilder.Build()
	if err != nil {
		return handler.HandleError(err)
	}

	pkg, err := reg.Resolve(name, c.options()...)
	if err != nil {
		logger.Warn(
			"package retrieval from registry failed",
			"name", name,
			"version", c.Version,
			"tools", strings.Join(c.Tools, ","),
			"runtime", c.Runtime,
			"source", c.Source,
			"error", err,
		)
		return handler.HandleError(fmt.Errorf(
			"⚠️ Failed to get package '%s@%s' from registry: %w",
			name,
			c.Version,
			err),
		)
	}

	entry, err := parseServerEntry(pkg, runtime.Runtime(c.Runtime), c.Tools, c.MCPDSupportedRuntimes())
	if err != nil {
		return handler.HandleError(fmt.Errorf("error parsing server entry: %w", err))
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return handler.HandleError(err)
	}

	err = cfg.AddServer(entry)
	if err != nil {
		return handler.HandleError(err)
	}

	// User-friendly output for text format.
	c.packagePrinter.SetHeader(func(w io.Writer, count int) {
		_, _ = fmt.Fprintln(w)
	})
	c.packagePrinter.SetFooter(func(w io.Writer, count int) {
		_, _ = fmt.Fprintln(w)
	})

	logger.Debug(
		"Server added",
		"name", name,
		"package", entry.Package,
		"version", entry.PackageVersion(),
		"tools", entry.Tools,
	)

	// Print the package info.
	return handler.HandleResult(entry)
}

// selectRuntime returns the most appropriate runtime from a set of installations,
// given a list of supported runtimes in priority order.
//
// It first searches for any supported runtime marked as `Recommended` and returns the first such match.
// If none are recommended, it returns the first matched runtime from the `supported` list.
//
// Returns an error if no supported runtime is found.
func selectRuntime(
	installations map[runtime.Runtime]packages.Installation,
	requestedRuntime runtime.Runtime,
	supported []runtime.Runtime,
) (runtime.Runtime, error) {
	// Try to select the recommended runtime if present.
	for _, rt := range supported {
		if requestedRuntime != "" && rt != requestedRuntime {
			continue
		}
		if inst, ok := installations[rt]; ok && inst.Recommended {
			return rt, nil
		}
	}

	// Fall back to the first supported runtime by priority.
	for _, rt := range supported {
		if requestedRuntime != "" && rt != requestedRuntime {
			continue
		}
		if _, ok := installations[rt]; ok {
			return rt, nil
		}
	}

	return "", fmt.Errorf("no supported runtimes found")
}

func parseServerEntry(
	pkg packages.Package,
	requestedRuntime runtime.Runtime,
	requestedTools []string,
	supportedRuntimes []runtime.Runtime,
) (config.ServerEntry, error) {
	requestedTools, err := filter.MatchRequestedSlice(requestedTools, pkg.Tools.Names())
	if err != nil {
		return config.ServerEntry{}, fmt.Errorf("error matching requested tools: %w", err)
	}

	selectedRuntime, runtimeErr := selectRuntime(pkg.Installations, requestedRuntime, supportedRuntimes)
	if runtimeErr != nil {
		return config.ServerEntry{}, fmt.Errorf("error selecting runtime from available installations: %w", runtimeErr)
	}

	v := "latest"
	if pkg.Version != "" {
		v = pkg.Version
	}

	runtimeSpecificName := pkg.Installations[selectedRuntime].Package
	if runtimeSpecificName == "" {
		return config.ServerEntry{}, fmt.Errorf(
			"installation package name is missing for runtime '%s'",
			selectedRuntime,
		)
	}
	runtimePackageVersion := fmt.Sprintf("%s::%s@%s", selectedRuntime, runtimeSpecificName, v)

	envs := packages.FilterArguments(pkg.Arguments, packages.EnvVar, packages.Required)
	args := packages.FilterArguments(pkg.Arguments, packages.ValueArgument, packages.Required)
	boolArgs := packages.FilterArguments(pkg.Arguments, packages.BoolArgument, packages.Required)

	return config.ServerEntry{
		Name:              pkg.ID,
		Package:           runtimePackageVersion,
		Tools:             requestedTools,
		RequiredValueArgs: args.Names(),
		RequiredBoolArgs:  boolArgs.Names(),
		RequiredEnvVars:   envs.Names(),
	}, nil
}

func (c *AddCmd) options() []regopts.ResolveOption {
	var o []regopts.ResolveOption

	if c.Version != "" && c.Version != "latest" {
		o = append(o, regopts.WithResolveVersion(c.Version))
	}
	if c.Runtime != "" {
		o = append(o, regopts.WithResolveRuntime(runtime.Runtime(c.Runtime)))
	}
	if c.Source != "" {
		o = append(o, regopts.WithResolveSource(c.Source))
	}

	return o
}
