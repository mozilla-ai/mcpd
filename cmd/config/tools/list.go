package tools

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// ListCmd represents the command for listing tools for MCP servers.
// Use NewListCmd to create instances of ListCmd.
type ListCmd struct {
	*internalcmd.BaseCmd
	cfgLoader       config.Loader
	toolsPrinter    output.Printer[printer.ToolsListResult]
	registryBuilder registry.Builder
	Format          internalcmd.OutputFormat
	All             bool
	CacheDisabled   bool
	CacheRefresh    bool
	CacheDir        string
	CacheTTL        string
}

// NewListCmd creates a new list command for displaying MCP server tools.
func NewListCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:         baseCmd,
		cfgLoader:       opts.ConfigLoader,
		registryBuilder: opts.RegistryBuilder,
		toolsPrinter:    &printer.ToolsListPrinter{},
		Format:          internalcmd.FormatText, // Default to plain text
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "Lists the configured (allowed) tools for a specific MCP server",
		Long:  "Lists the configured (allowed) tools for a specific MCP server from the .mcpd.toml configuration file",
		RunE:  c.run,
		Args:  cobra.ExactArgs(1),
	}

	allowed := internalcmd.AllowedOutputFormats()
	cobraCmd.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowed.String()),
	)

	cobraCmd.Flags().BoolVar(
		&c.All,
		"all",
		false,
		"List all available tools from the registry instead of only allowed tools in config file "+
			"(supports caching flags)",
	)

	// Cache configuration flags (not used for standard configuration listing)
	cobraCmd.Flags().BoolVar(
		&c.CacheDisabled,
		"no-cache",
		false,
		"Disable registry manifest caching",
	)

	cobraCmd.Flags().BoolVar(
		&c.CacheRefresh,
		"refresh-cache",
		false,
		"Force refresh of cached registry manifests",
	)

	defaultCacheDir, err := options.DefaultCacheDir()
	if err != nil {
		return nil, fmt.Errorf("error getting default cache directory: %w", err)
	}

	cobraCmd.Flags().StringVar(
		&c.CacheDir,
		"cache-dir",
		defaultCacheDir,
		"Directory for caching registry manifests",
	)

	cobraCmd.Flags().StringVar(
		&c.CacheTTL,
		"cache-ttl",
		options.DefaultCacheTTL().String(),
		"Time-to-live for cached registry manifests (e.g. 1h, 30m, 24h)",
	)

	return cobraCmd, nil
}

// listAll queries the registry for all available tools for the given server.
func (c *ListCmd) listAll(h output.Handler[printer.ToolsListResult], s *config.ServerEntry) error {
	serverRuntime := s.Runtime()
	if serverRuntime == "" {
		return h.HandleError(fmt.Errorf("invalid package format in configuration: %s", s.Package))
	}

	version := s.PackageVersion()

	// Parse cache TTL.
	cacheTTL, err := time.ParseDuration(c.CacheTTL)
	if err != nil {
		return h.HandleError(fmt.Errorf("invalid cache TTL: %w", err))
	}

	// Build registry with caching options.
	reg, err := c.registryBuilder.Build(
		options.WithCaching(!c.CacheDisabled),
		options.WithRefreshCache(c.CacheRefresh),
		options.WithCacheDir(c.CacheDir),
		options.WithCacheTTL(cacheTTL),
	)
	if err != nil {
		return h.HandleError(fmt.Errorf("failed to build registry: %w", err))
	}

	// Build resolve options with runtime and version.
	resolveOpts := []options.ResolveOption{
		options.WithResolveRuntime(runtime.Runtime(serverRuntime)),
	}
	if version != "" && version != "latest" {
		resolveOpts = append(resolveOpts, options.WithResolveVersion(version))
	}

	// Resolve the specific server from the registry.
	serverResult, err := reg.Resolve(s.Name, resolveOpts...)
	if err != nil {
		return h.HandleError(fmt.Errorf("failed to resolve server '%s': %w", s.Name, err))
	}

	// Extract and normalize all available tools.
	allTools := make([]string, len(serverResult.Tools))
	for i, tool := range serverResult.Tools {
		allTools[i] = filter.NormalizeString(tool.Name)
	}

	// Sort tools alphabetically for consistent output.
	sort.Strings(allTools)

	result := printer.ToolsListResult{
		Server: s.Name,
		Tools:  allTools,
		Count:  len(allTools),
	}

	return h.HandleResult(result)
}

// list outputs the configured tools for the given server.
func (c *ListCmd) list(h output.Handler[printer.ToolsListResult], s *config.ServerEntry) error {
	// Sort tools alphabetically for consistent output.
	tools := slices.Clone(s.Tools)
	sort.Strings(tools)

	result := printer.ToolsListResult{
		Server: s.Name,
		Tools:  tools,
		Count:  len(tools),
	}

	return h.HandleResult(result)
}

func (c *ListCmd) run(cmd *cobra.Command, args []string) error {
	handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.Format, c.toolsPrinter)
	if err != nil {
		return err
	}

	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return handler.HandleError(fmt.Errorf("server-name is required"))
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return handler.HandleError(err)
	}

	// Find the server in the configuration.
	var foundServer *config.ServerEntry
	for _, srv := range cfg.ListServers() {
		if srv.Name == serverName {
			foundServer = &srv
			break
		}
	}

	if foundServer == nil {
		return handler.HandleError(fmt.Errorf("server '%s' not found in configuration", serverName))
	}

	// If --all flag is set, query the registry for all available tools.
	if c.All {
		return c.listAll(handler, foundServer)
	}

	// Otherwise, list the configured tools.
	return c.list(handler, foundServer)
}
