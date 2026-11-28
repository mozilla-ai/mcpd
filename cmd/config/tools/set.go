package tools

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/flags"
	"github.com/mozilla-ai/mcpd/internal/registry"
	"github.com/mozilla-ai/mcpd/internal/registry/options"
	"github.com/mozilla-ai/mcpd/internal/runtime"
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

	defaultCacheTTL := options.DefaultCacheTTL()
	cobraCmd.Flags().StringVar(
		&c.cacheTTL,
		"cache-ttl",
		defaultCacheTTL.String(),
		"Time-to-live for cached registry manifests (e.g. 1h, 30m, 24h)",
	)

	return cobraCmd, nil
}

// resolveAvailableTools queries the registry for all available tools for the given server.
// Returns the set of tool names as the registry presents them.
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
		available[domain.ToolName(tool.Name).String()] = struct{}{}
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

	// Keep each supplied tool name as given, dropping any that is only whitespace.
	requestedTools := make([]domain.ToolName, 0, len(c.tools))
	for _, tool := range c.tools {
		name := domain.ToolName(tool)
		if name.String() != "" {
			requestedTools = append(requestedTools, name)
		}
	}

	if len(requestedTools) == 0 {
		return fmt.Errorf("at least one valid tool name is required")
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return err
	}

	// Find the server in the configuration, matching without regard to case so the name works here as it
	// does in a daemon URL.
	var foundServer *config.ServerEntry

	wanted := domain.ServerName(serverName).Key()
	for _, srv := range cfg.ListServers() {
		if domain.ServerName(srv.Name).Key() == wanted {
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

	// MCP tool names are case-sensitive, so a requested name has to match the registry's own spelling.
	var invalidTools []string

	matchedTools := make([]domain.ToolName, 0, len(requestedTools))
	for _, tool := range requestedTools {
		if _, exists := availableTools[tool.String()]; !exists {
			invalidTools = append(invalidTools, tool.String())

			continue
		}

		matchedTools = append(matchedTools, tool)
	}

	if len(invalidTools) > 0 {
		return fmt.Errorf("the following tools are not available for server '%s': %v", serverName, invalidTools)
	}

	// Deduplicate, so one tool is recorded once.
	toolSet := make(map[string]struct{}, len(foundServer.Tools)+len(matchedTools))
	for _, tool := range foundServer.Tools {
		toolSet[domain.ToolName(tool).String()] = struct{}{}
	}

	// Track which tools are actually new.
	newTools := make([]string, 0, len(matchedTools))
	for _, tool := range matchedTools {
		if _, exists := toolSet[tool.String()]; exists {
			continue
		}

		toolSet[tool.String()] = struct{}{}
		newTools = append(newTools, tool.String())
	}

	allTools := slices.Sorted(maps.Keys(toolSet))

	// Update the server's tools.
	foundServer.Tools = allTools

	// The stored name is used rather than the supplied one, because the lookup above matched without
	// regard to case and the configuration holds only the spelling it was written with.
	if err := cfg.UpdateServer(*foundServer); err != nil {
		return fmt.Errorf("error updating server configuration: %w", err)
	}

	// Provide feedback to the user, naming the server as the configuration spells it.
	var msg string
	if len(newTools) == 0 {
		msg = fmt.Sprintf(
			"✓ No new tools added for server '%s' (all specified tools already exist)\n",
			foundServer.Name,
		)
	} else {
		msg = fmt.Sprintf("✓ Tools added for server '%s': %v\n", foundServer.Name, newTools)
	}

	if _, err := fmt.Fprint(cmd.OutOrStdout(), msg); err != nil {
		return err
	}

	return nil
}
