package plugins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestNewRemoveCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewRemoveCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "remove", c.Use)
	require.Contains(t, c.Short, "Remove a plugin entry from a category")
	require.NotNil(t, c.RunE)
}

func TestRemoveCmd_RemoveExistingPlugin(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' removed from category 'authentication' (operation: deleted)",
	)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 0)
}

func TestRemoveCmd_RemoveNonExistentPlugin(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "other-plugin",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "nonexistent")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin nonexistent not found in category 'authentication'")
}

func TestRemoveCmd_InvalidCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "invalid")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid category 'invalid'")
}

func TestRemoveCmd_NoPluginConfig(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "test")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "no plugins configured")
}

func TestRemoveCmd_RemoveLastPluginInCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryObservability, config.PluginEntry{
		Name:  "metrics",
		Flows: []config.Flow{config.FlowRequest, config.FlowResponse},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "observability")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "metrics")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'metrics' removed from category 'observability' (operation: deleted)",
	)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	require.NotNil(t, cfg.Plugins)
	obsPlugins := cfg.Plugins.ListPlugins(config.CategoryObservability)
	require.Len(t, obsPlugins, 0)
}

func TestRemoveCmd_RemoveFromMultipleCategories(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	_, err = cfg.UpsertPlugin(config.CategoryObservability, config.PluginEntry{
		Name:  "metrics",
		Flows: []config.Flow{config.FlowRequest, config.FlowResponse},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd1, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd1.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd1.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout1 bytes.Buffer
	var stderr1 bytes.Buffer
	removeCmd1.SetOut(&stdout1)
	removeCmd1.SetErr(&stderr1)

	err = executeCmd(t, removeCmd1, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr1.String())
	require.Contains(
		t,
		stdout1.String(),
		"✓ Plugin 'jwt-auth' removed from category 'authentication' (operation: deleted)",
	)

	removeCmd2, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd2.Flags().Set(flagCategory, "observability")
	require.NoError(t, err)
	err = removeCmd2.Flags().Set(flagName, "metrics")
	require.NoError(t, err)

	var stdout2 bytes.Buffer
	var stderr2 bytes.Buffer
	removeCmd2.SetOut(&stdout2)
	removeCmd2.SetErr(&stderr2)

	err = executeCmd(t, removeCmd2, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr2.String())
	require.Contains(
		t,
		stdout2.String(),
		"✓ Plugin 'metrics' removed from category 'observability' (operation: deleted)",
	)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	require.NotNil(t, cfg.Plugins)
	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 0)
	obsPlugins := cfg.Plugins.ListPlugins(config.CategoryObservability)
	require.Len(t, obsPlugins, 0)
}

func TestRemoveCmd_CaseInsensitiveCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryRateLimiting, config.PluginEntry{
		Name:  "rate-limiter",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "RATE_LIMITING")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "rate-limiter")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'rate-limiter' removed from category 'rate_limiting' (operation: deleted)",
	)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	rateLimitPlugins := cfg.Plugins.ListPlugins(config.CategoryRateLimiting)
	require.Len(t, rateLimitPlugins, 0)
}

func TestRemoveCmd_RemoveOneOfMultiplePluginsInCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "oauth2",
		Flows: []config.Flow{config.FlowRequest, config.FlowResponse},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(
		t,
		stdout.String(),
		"✓ Plugin 'jwt-auth' removed from category 'authentication' (operation: deleted)",
	)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "oauth2", authPlugins[0].Name)
}

func TestRemoveCmd_ConfigFileValidAfterRemove(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	cfgModifier, err := loader.Load("ignored-value")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	requiredTrue := true
	commitHash := "abc123"
	_, err = cfg.UpsertPlugin(config.CategoryObservability, config.PluginEntry{
		Name:       "metrics",
		Flows:      []config.Flow{config.FlowRequest, config.FlowResponse},
		Required:   &requiredTrue,
		CommitHash: &commitHash,
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = removeCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = removeCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	err = executeCmd(t, removeCmd, []string{})
	require.NoError(t, err)

	cfgModifier, err = loader.Load("ignored-value")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	require.NotNil(t, cfg.Plugins)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 0)

	obsPlugins := cfg.Plugins.ListPlugins(config.CategoryObservability)
	require.Len(t, obsPlugins, 1)
	require.Equal(t, "metrics", obsPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest, config.FlowResponse}, obsPlugins[0].Flows)
	require.NotNil(t, obsPlugins[0].Required)
	require.True(t, *obsPlugins[0].Required)
	require.NotNil(t, obsPlugins[0].CommitHash)
	require.Equal(t, "abc123", *obsPlugins[0].CommitHash)
}
