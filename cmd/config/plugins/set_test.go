package plugins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestNewSetCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewSetCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "set", c.Use)
	require.Contains(t, c.Short, "Sets top-level config for all plugins")
	require.NotNil(t, c.RunE)
}

func TestSetCmd_SetPluginDirectory(t *testing.T) {
	t.Parallel()

	// Create a real temp directory for the plugin directory.
	pluginDir := t.TempDir()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagDir, pluginDir)
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "✓ Plugin directory set to:")
	require.Contains(t, stdout.String(), pluginDir)

	// Verify directory was set by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	require.NotNil(t, cfg.(*config.Config).Plugins)
	require.Equal(t, pluginDir, cfg.(*config.Config).Plugins.Dir)
}

func TestSetCmd_SetPluginDirectory_EmptyPath(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagDir, "   ")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin directory path cannot be empty")
}

func TestSetCmd_CreatePluginEntry(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' configured in category 'authentication' (operation: created)",
	)

	// Verify plugin was created by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows)
}

func TestSetCmd_CreatePluginEntry_WithAllFields(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "observability")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "metrics")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "response")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagRequired, "true")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagCommitHash, "abc123")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'metrics' configured in category 'observability' (operation: created)",
	)

	// Verify plugin was created with all fields by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	obsPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryObservability)
	require.Len(t, obsPlugins, 1)
	require.Equal(t, "metrics", obsPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest, config.FlowResponse}, obsPlugins[0].Flows)
	require.NotNil(t, obsPlugins[0].Required)
	require.True(t, *obsPlugins[0].Required)
	require.NotNil(t, obsPlugins[0].CommitHash)
	require.Equal(t, "abc123", *obsPlugins[0].CommitHash)
}

func TestSetCmd_CreatePluginEntry_MissingFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.ErrorContains(t, err, "flows are required when creating a new plugin entry")
}

func TestSetCmd_UpdatePluginEntry(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Load the config and add a plugin to it.
	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	// Add initial plugin.
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagRequired, "true")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' configured in category 'authentication' (operation: updated)",
	)

	// Verify plugin was updated by reloading the config.
	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows) // Unchanged
	require.NotNil(t, authPlugins[0].Required)
	require.True(t, *authPlugins[0].Required)
}

func TestSetCmd_UpdatePluginEntry_Flows(t *testing.T) {
	t.Parallel()

	requiredTrue := true

	loader := newMockLoaderFromFile(t)

	// Load the config and add a plugin to it.
	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	// Add initial plugin.
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:     "jwt-auth",
		Flows:    []config.Flow{config.FlowRequest},
		Required: &requiredTrue,
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "response")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' configured in category 'authentication' (operation: updated)",
	)

	// Verify plugin flows were updated by reloading the config.
	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowResponse}, authPlugins[0].Flows) // Updated
	require.NotNil(t, authPlugins[0].Required)
	require.True(t, *authPlugins[0].Required) // Unchanged
}

func TestSetCmd_UpdatePluginEntry_NoChanges(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Load the config and add a plugin to it.
	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	// Add initial plugin.
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' configured in category 'authentication' (operation: noop)",
	)
}

func TestSetCmd_NoFlagsProvided(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "provide either --dir or (--category and --name)")
}

func TestSetCmd_OnlyCategoryProvided(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "flags (category, name) must be provided together")
}

func TestSetCmd_OnlyNameProvided(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "flags (category, name) must be provided together")
}

func TestSetCmd_MixingModes(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagDir, "/path/to/plugins")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot use --dir with plugin entry flags")
}

func TestSetCmd_MixingModes_DirWithFlow(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagDir, "/path/to/plugins")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot use --dir with plugin entry flags")
}

func TestSetCmd_EmptyPluginName(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "   ")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin name cannot be empty")
}

func TestSetCmd_InvalidFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "invalid")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "at least one valid flow is required")
}

func TestSetCmd_DuplicateFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	// Verify duplicates were deduplicated - should only have one flow.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows)
}

func TestSetCmd_CaseInsensitiveFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "rate_limiting")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "rate-limiter")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "REQUEST")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "RESPONSE")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	// Verify plugin was created by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	rateLimitPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryRateLimiting)
	require.Len(t, rateLimitPlugins, 1)
	require.Equal(t, "rate-limiter", rateLimitPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest, config.FlowResponse}, rateLimitPlugins[0].Flows)
}

func TestSetCmd_RequiredFalseNotSet(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)

	// Verify Required field is nil (not set) by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Nil(t, authPlugins[0].Required)
}

func TestSetCmd_RequiredFalseExplicitlySet(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagRequired, "false")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)

	// Verify Required field is set to false by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.NotNil(t, authPlugins[0].Required)
	require.False(t, *authPlugins[0].Required)
}

func TestSetCmd_CommitHashEmpty(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagCommitHash, "")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)

	// Verify CommitHash field is nil (not set) by reloading the config.
	cfg, err := loader.Load("ignored-value")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Nil(t, authPlugins[0].CommitHash)
}

func TestSetCmd_UpdatePluginEntry_ClearCommitHash(t *testing.T) {
	t.Parallel()

	commitHash := "abc123"

	loader := newMockLoaderFromFile(t)

	// Load the config and add a plugin to it.
	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	// Add initial plugin with commit hash.
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:       "jwt-auth",
		Flows:      []config.Flow{config.FlowRequest},
		CommitHash: &commitHash,
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = setCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = setCmd.Flags().Set(flagCommitHash, "")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCmd.SetOut(&stdout)
	setCmd.SetErr(&stderr)

	err = executeCmd(t, setCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' configured in category 'authentication' (operation: updated)",
	)

	// Verify commit hash was cleared by reloading the config.
	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows) // Unchanged
	require.Nil(t, authPlugins[0].CommitHash)                                 // Cleared
}
