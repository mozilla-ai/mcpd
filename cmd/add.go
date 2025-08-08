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
	AllowDeprecated bool
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
		"",
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
		"Optional, specify the runtime of the server (e.g. uvx, npx)",
	)

	cobraCommand.Flags().StringVar(
		&c.Source,
		"source",
		"",
		"Optional, specify the source registry of the server (e.g. mozilla-ai)",
	)

	allowed := internalcmd.AllowedOutputFormats()
	cobraCommand.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowed.String()),
	)

	cobraCommand.Flags().BoolVar(
		&c.AllowDeprecated,
		"allow-deprecated",
		false,
		"Optional, allows server installations marked as deprecated to be added",
	)

	return cobraCommand, nil
}

// serverEntryOptions contains configuration for parsing a server entry
type serverEntryOptions struct {
	Runtime           runtime.Runtime
	Tools             []string
	SupportedRuntimes []runtime.Runtime
	AllowDeprecated   bool
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
			"server retrieval from registry failed",
			"name", name,
			"version", c.Version,
			"tools", strings.Join(c.Tools, ","),
			"runtime", c.Runtime,
			"source", c.Source,
			"error", err,
		)
		return handler.HandleError(fmt.Errorf(
			"⚠️ Failed to get server '%s@%s' from registry: %w",
			name,
			c.Version,
			err),
		)
	}

	opts := serverEntryOptions{
		Runtime:           runtime.Runtime(c.Runtime),
		Tools:             c.Tools,
		SupportedRuntimes: c.MCPDSupportedRuntimes(),
		AllowDeprecated:   c.AllowDeprecated,
	}
	entry, err := parseServerEntry(pkg, opts)
	if err != nil {
		return handler.HandleError(err)
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

func parseServerEntry(pkg packages.Server, opts serverEntryOptions) (config.ServerEntry, error) {
	requestedTools, err := filter.MatchRequestedSlice(opts.Tools, pkg.Tools.Names())
	if err != nil {
		return config.ServerEntry{}, fmt.Errorf("error matching requested tools: %w", err)
	}

	selectedRuntime, err := selectRuntime(pkg.Installations, opts.Runtime, opts.SupportedRuntimes)
	if err != nil {
		return config.ServerEntry{}, fmt.Errorf("error selecting runtime from available installations: %w", err)
	}

	installation, ok := pkg.Installations[selectedRuntime]
	if !ok {
		return config.ServerEntry{}, fmt.Errorf(
			"installation not found for runtime '%s'",
			selectedRuntime,
		)
	}

	if installation.Deprecated && !opts.AllowDeprecated {
		return config.ServerEntry{}, fmt.Errorf(
			"server '%s' with runtime '%s' is deprecated, use --allow-deprecated flag to proceed",
			pkg.ID,
			selectedRuntime,
		)
	}

	if installation.Package == "" {
		return config.ServerEntry{}, fmt.Errorf(
			"installation server name is missing for runtime '%s'",
			selectedRuntime,
		)
	}

	version := "latest"
	if installation.Version != "" {
		version = installation.Version
	}

	runtimePackageVersion := fmt.Sprintf("%s::%s@%s", selectedRuntime, installation.Package, version)
	envs := pkg.Arguments.FilterBy(packages.Required, packages.EnvVar).Names()
	positionalArgs := pkg.Arguments.FilterBy(packages.Required, packages.PositionalArgument).Ordered().Names()
	valueArgs := pkg.Arguments.FilterBy(packages.Required, packages.ValueArgument).Ordered().Names()
	boolArgs := pkg.Arguments.FilterBy(packages.Required, packages.BoolArgument).Names()

	return config.ServerEntry{
		Name:                   pkg.ID,
		Package:                runtimePackageVersion,
		Tools:                  requestedTools,
		RequiredPositionalArgs: positionalArgs,
		RequiredValueArgs:      valueArgs,
		RequiredBoolArgs:       boolArgs,
		RequiredEnvVars:        envs,
	}, nil
}

func (c *AddCmd) options() []regopts.ResolveOption {
	var o []regopts.ResolveOption

	if c.Version != "" {
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
