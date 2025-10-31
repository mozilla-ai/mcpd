package plugins

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// createTempConfigFile creates a temporary config file for testing.
func createTempConfigFile(t *testing.T) string {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	// Write minimal valid config.
	content := "servers = []\n"
	require.NoError(t, os.WriteFile(tempFile.Name(), []byte(content), 0o644))

	return tempFile.Name()
}

// mockLoaderFromFile creates a mock loader that loads from a real temp file.
// This allows the config to be saved properly during tests.
type mockLoaderFromFile struct {
	filePath string
	loader   *config.DefaultLoader
}

func newMockLoaderFromFile(t *testing.T) *mockLoaderFromFile {
	t.Helper()

	return &mockLoaderFromFile{
		filePath: createTempConfigFile(t),
		loader:   &config.DefaultLoader{},
	}
}

func (m *mockLoaderFromFile) Load(_ string) (config.Modifier, error) {
	return m.loader.Load(m.filePath)
}

func TestNewAddCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewAddCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "add <plugin-name>", c.Use)
	require.Contains(t, c.Short, "Add a new plugin entry")
	require.NotNil(t, c.RunE)
}

func TestAddCmd_Success(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt-auth"})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "✓ Plugin 'jwt-auth' added to category 'authentication'")

	// Verify plugin was added by reloading the config.
	cfg, err := loader.Load("")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows)
}

func TestAddCmd_WithAllFields(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "observability")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "response")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagRequired, "true")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagCommitHash, "abc123")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"metrics"})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "✓ Plugin 'metrics' added to category 'observability'")

	// Verify plugin was added with all fields by reloading the config.
	cfg, err := loader.Load("")
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

func TestAddCmd_PluginAlreadyExists(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Dir:            "/path/to/plugins",
			Authentication: []config.PluginEntry{{Name: "jwt-auth", Flows: []config.Flow{config.FlowRequest}}},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt-auth"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin 'jwt-auth' already exists in category 'authentication'")
	require.Contains(t, err.Error(), "To update an existing plugin, use: mcpd config plugins set")
}

func TestAddCmd_InvalidFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "invalid")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one valid flow is required")
}

func TestAddCmd_DuplicateFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt"})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	// Verify duplicates were deduplicated - should only have one flow.
	cfg, err := loader.Load("")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt", authPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, authPlugins[0].Flows)
}

func TestAddCmd_EmptyName(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"   "})
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin name cannot be empty")
}

func TestAddCmd_InvalidCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "invalid-category")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid category 'invalid-category'")
}

func TestAddCmd_CaseInsensitiveFlows(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "rate_limiting")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "REQUEST")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "RESPONSE")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"rate-limiter"})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	// Verify plugin was added by reloading the config.
	cfg, err := loader.Load("")
	require.NoError(t, err)

	rateLimitPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryRateLimiting)
	require.Len(t, rateLimitPlugins, 1)
	require.Equal(t, "rate-limiter", rateLimitPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest, config.FlowResponse}, rateLimitPlugins[0].Flows)
}

func TestAddCmd_RequiredFalseNotSet(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt-auth"})
	require.NoError(t, err)

	// Verify Required field is nil (not set) by reloading the config.
	cfg, err := loader.Load("")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Nil(t, authPlugins[0].Required)
}

func TestAddCmd_RequiredFalseExplicitlySet(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)
	base := &cmd.BaseCmd{}
	addCmd, err := NewAddCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = addCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagFlow, "request")
	require.NoError(t, err)
	err = addCmd.Flags().Set(flagRequired, "false")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	addCmd.SetOut(&stdout)
	addCmd.SetErr(&stderr)

	err = addCmd.RunE(addCmd, []string{"jwt-auth"})
	require.NoError(t, err)

	// Verify Required field is set to false by reloading the config.
	cfg, err := loader.Load("")
	require.NoError(t, err)

	authPlugins := cfg.(*config.Config).Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.NotNil(t, authPlugins[0].Required)
	require.False(t, *authPlugins[0].Required)
}
