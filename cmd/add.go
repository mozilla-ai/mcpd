package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
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
	*cmd.BaseCmd
	Version         string
	Tools           []string
	Runtime         string
	Source          string
	cfgLoader       config.Loader
	packagePrinter  printer.Printer
	registryBuilder registry.Builder
}

// NewAddCmd creates a newly configured (Cobra) command.
func NewAddCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &AddCmd{
		BaseCmd:         baseCmd,
		cfgLoader:       opts.ConfigLoader,
		packagePrinter:  opts.Printer,
		registryBuilder: opts.RegistryBuilder,
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

	cobraCommand.Flags().StringVar(
		&c.Runtime,
		"runtime",
		"",
		"Optional, specify the runtime of the server package (e.g. uvx, npx)",
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

	reg, err := c.registryBuilder.Build()
	if err != nil {
		return err
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
		return fmt.Errorf("⚠️ Failed to get package '%s@%s' from registry: %w", name, c.Version, err)
	}

	requestedTools, err := filter.MatchRequestedSlice(c.Tools, pkg.Tools)
	if err != nil {
		return fmt.Errorf("error matching requested tools: %w", err)
	}

	selectedRuntime, runtimeErr := c.runtime(pkg)
	if runtimeErr != nil {
		return runtimeErr
	}

	version := "latest"
	if pkg.Version != "" {
		version = pkg.Version
	}

	runtimePackageVersion := fmt.Sprintf("%s::%s@%s", selectedRuntime, pkg.Name, version)

	entry := config.ServerEntry{
		Name:    pkg.ID,
		Package: runtimePackageVersion,
		Tools:   requestedTools,
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return err
	}

	err = cfg.AddServer(entry)
	if err != nil {
		return err
	}

	// User-friendly output + logging
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "✓ Added server '%s' (version: %s)%s\n", name, version, requestedTools)
	if err != nil {
		return err
	}
	logger.Debug("Server added", "name", name, "version", version, "tools", requestedTools)

	// Print the package info.
	if err = c.packagePrinter.PrintPackage(pkg); err != nil {
		return err
	}

	return nil
}

func (c *AddCmd) runtime(pkg packages.Package) (runtime.Runtime, error) {
	// TODO: Sort out preference for runtimes
	var selectedRuntime runtime.Runtime
	supportedRuntimes := runtime.DefaultSupportedRuntimes()
	for k, v := range pkg.InstallationDetails {
		if _, ok := supportedRuntimes[k]; ok {
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
		return "", fmt.Errorf("no supported runtimes found for '%s'", pkg.Name)
	}
	return selectedRuntime, nil
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
