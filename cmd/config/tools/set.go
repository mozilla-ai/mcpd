package tools

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// SetCmd represents the command for setting (adding) tools to an MCP server configuration.
// Use NewSetCmd to create instances of SetCmd.
type SetCmd struct {
	*cmd.BaseCmd
	cfgLoader       config.Loader
	registryBuilder registry.Builder
	tools           []string
	cacheDisabled   bool
	cacheRefresh    bool
	cacheDir        string
	cacheTTL        string
}

// NewSetCmd creates a new set command for adding tools to an MCP server configuration.
func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:         baseCmd,
		cfgLoader:       opts.ConfigLoader,
		registryBuilder: opts.RegistryBuilder,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> --tool <tool1> [--tool <tool2> ...]",
		Short: "Add allowed tools to an MCP server configuration",
		Long: "Add allowed tools to an MCP server configuration. " +
			"Tools are added to the existing set of tools (append behavior). " +
			"Duplicate tools are automatically deduplicated. " +
			"Only tools that are available for the server (as defined in the registry) can be added.",
		RunE: c.run,
		Args: cobra.ExactArgs(1), // server-name
	}

	cobraCmd.Flags().StringArrayVar(
		&c.tools,
		"tool",
		nil,
		"Tool to add to the server's allowed list (can be repeated)",
	)
	_ = cobraCmd.MarkFlagRequired("tool")

	// Cache configuration flags (for validating tools against registry)
	cobraCmd.Flags().BoolVar(
		&c.cacheDisabled,
		"no-cache",
		false,
		"Disable registry manifest caching",
	)

	cobraCmd.Flags().BoolVar(
		&c.cacheRefresh,
		"refresh-cache",
		false,
		"Force refresh of cached registry manifests",
	)

	defaultCacheDir, err := options.DefaultCacheDir()
	if err != nil {
		return nil, fmt.Errorf("error getting default cache directory: %w", err)
	}

	cobraCmd.Flags().StringVar(
		&c.cacheDir,
		"cache-dir",
		defaultCacheDir,
		"Directory for caching registry manifests",
	)

	cobraCmd.Flags().StringVar(
		&c.cacheTTL,
		"cache-ttl",
		options.DefaultCacheTTL().String(),
		"Time-to-live for cached registry manifests (e.g. 1h, 30m, 24h)",
	)

	return cobraCmd, nil
}

// resolveAvailableTools queries the registry for all available tools for the given server.
// Returns a map of normalized tool names.
func (c *SetCmd) resolveAvailableTools(s *config.ServerEntry) (map[string]struct{}, error) {
	serverRuntime := s.Runtime()
	if serverRuntime == "" {
		return nil, fmt.Errorf("invalid package format in configuration: %s", s.Package)
	}

	version := s.PackageVersion()

	// Parse cache TTL.
	cacheTTL, err := time.ParseDuration(c.cacheTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache TTL: %w", err)
	}

	// Build registry with caching options.
	reg, err := c.registryBuilder.Build(
		options.WithCaching(!c.cacheDisabled),
		options.WithRefreshCache(c.cacheRefresh),
		options.WithCacheDir(c.cacheDir),
		options.WithCacheTTL(cacheTTL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build registry: %w", err)
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
		return nil, fmt.Errorf("failed to resolve server '%s': %w", s.Name, err)
	}

	available := make(map[string]struct{}, len(serverResult.Tools))
	for _, tool := range serverResult.Tools {
		available[filter.NormalizeString(tool.Name)] = struct{}{}
	}

	return available, nil
}

func (c *SetCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	if len(c.tools) == 0 {
		return fmt.Errorf("at least one --tool flag is required")
	}

	// Normalize all supplied tool names.
	normalizedTools := make([]string, 0, len(c.tools))
	for _, tool := range c.tools {
		normalized := filter.NormalizeString(tool)
		if normalized != "" {
			normalizedTools = append(normalizedTools, normalized)
		}
	}

	if len(normalizedTools) == 0 {
		return fmt.Errorf("at least one valid tool name is required")
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return err
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
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	// Get all available tools from the registry for this server (runtime, version).
	availableTools, err := c.resolveAvailableTools(foundServer)
	if err != nil {
		return fmt.Errorf("failed to get available tools for server '%s': %w", serverName, err)
	}

	// Validate that all requested tools are available.
	var invalidTools []string
	for _, tool := range normalizedTools {
		if _, exists := availableTools[tool]; !exists {
			invalidTools = append(invalidTools, tool)
		}
	}

	if len(invalidTools) > 0 {
		return fmt.Errorf("the following tools are not available for server '%s': %v", serverName, invalidTools)
	}

	// Create a map for efficient deduplication.
	toolSet := make(map[string]struct{}, len(foundServer.Tools)+len(normalizedTools))
	for _, tool := range foundServer.Tools {
		toolSet[tool] = struct{}{}
	}

	// Track which tools are actually new.
	newTools := make([]string, 0, len(normalizedTools))
	for _, tool := range normalizedTools {
		if _, exists := toolSet[tool]; !exists {
			toolSet[tool] = struct{}{}
			newTools = append(newTools, tool)
		}
	}

	allTools := slices.Collect(maps.Keys(toolSet))
	slices.Sort(allTools)

	// Update the server's tools.
	foundServer.Tools = allTools

	// Update server in config by removing and re-adding (following existing pattern).
	if err := cfg.RemoveServer(serverName); err != nil {
		return fmt.Errorf("error updating server configuration: %w", err)
	}

	if err := cfg.AddServer(*foundServer); err != nil {
		return fmt.Errorf("error updating server configuration: %w", err)
	}

	// Save the configuration.
	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	// Provide feedback to the user.
	var msg string
	if len(newTools) == 0 {
		msg = fmt.Sprintf("✓ No new tools added for server '%s' (all specified tools already exist)\n", serverName)
	} else {
		msg = fmt.Sprintf("✓ Tools added for server '%s': %v\n", serverName, newTools)
	}

	if _, err := fmt.Fprint(cmd.OutOrStdout(), msg); err != nil {
		return err
	}

	return nil
}
