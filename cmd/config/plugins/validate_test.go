package plugins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

func TestNewValidateCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewValidateCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "validate", c.Use)
	require.Contains(t, c.Short, "Validate plugin configuration")
	require.NotNil(t, c.RunE)
}

func TestValidateCmd_AllValidPlugins(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "auth-plugin",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
		},
		config.CategoryValidation: {
			{
				Name:     "validator",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(false),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "All plugins validated successfully!")
	require.Contains(t, output, "Plugins: 2")
	require.Contains(t, output, "Issues: 0")
}

func TestValidateCmd_InvalidPluginEntry(t *testing.T) {
	t.Parallel()

	// Plugin with missing required fields.
	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:  "invalid-plugin",
				Flows: nil, // Missing required flows.
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed with 1 error(s)")
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "Plugin 'invalid-plugin'")
	require.Contains(t, output, "Issues: 1")
}

func TestValidateCmd_SpecificCategory(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "auth-plugin",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
		},
		config.CategoryValidation: {
			{
				Name:  "invalid-validator",
				Flows: nil, // Invalid.
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Only validate authentication category.
	err = validateCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.NoError(t, err) // Should pass because we only validated authentication.
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "All plugins validated successfully!")
	require.Contains(t, output, "Categories: 1")
	require.Contains(t, output, "Plugins: 1")
}

func TestValidateCmd_NoPluginConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: nil,
	}
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "No plugin configuration found")
}

func TestValidateCmd_CheckBinaries(t *testing.T) {
	t.Parallel()

	t.Run("binary exists", func(t *testing.T) {
		t.Parallel()

		// Create temp directory with a plugin binary.
		pluginDir := t.TempDir()
		pluginPath := filepath.Join(pluginDir, "test-plugin")
		err := os.WriteFile(pluginPath, []byte("binary"), 0o755)
		require.NoError(t, err)

		plugins := map[config.Category][]config.PluginEntry{
			config.CategoryAuthentication: {
				{
					Name:     "test-plugin",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(true),
				},
			},
		}

		cfg := newTestConfig(t, plugins)
		cfg.Plugins.Dir = pluginDir
		loader := &mockLoader{cfg: cfg}

		base := &cmd.BaseCmd{}
		validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)

		err = validateCmd.Flags().Set("check-binaries", "true")
		require.NoError(t, err)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		validateCmd.SetOut(&stdout)
		validateCmd.SetErr(&stderr)

		err = executeCmd(t, validateCmd, []string{})
		require.NoError(t, err)
		require.Empty(t, stderr.String())

		output := stdout.String()
		require.Contains(t, output, "All plugins validated successfully!")
		require.Contains(t, output, "Binary checks: enabled")
	})

	t.Run("binary missing", func(t *testing.T) {
		t.Parallel()

		pluginDir := t.TempDir()

		plugins := map[config.Category][]config.PluginEntry{
			config.CategoryAuthentication: {
				{
					Name:     "missing-plugin",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(true),
				},
			},
		}

		cfg := newTestConfig(t, plugins)
		cfg.Plugins.Dir = pluginDir
		loader := &mockLoader{cfg: cfg}

		base := &cmd.BaseCmd{}
		validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)

		err = validateCmd.Flags().Set("check-binaries", "true")
		require.NoError(t, err)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		validateCmd.SetOut(&stdout)
		validateCmd.SetErr(&stderr)

		err = executeCmd(t, validateCmd, []string{})
		require.Error(t, err)
		// Validating loader catches missing binaries at load time.
		require.ErrorContains(t, err, "missing-plugin")
		require.ErrorContains(t, err, "not found")
		require.Empty(t, stderr.String())
	})
}

func TestValidateCmd_PluginDirectoryNotConfigured(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "test-plugin",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	cfg.Plugins.Dir = "" // No directory configured.
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = validateCmd.Flags().Set("check-binaries", "true")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.Error(t, err)
	// Validating loader catches missing directory config at load time.
	require.ErrorContains(t, err, "plugin directory not configured")
	require.Empty(t, stderr.String())
}

func TestValidateCmd_PluginDirectoryDoesNotExist(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "test-plugin",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	cfg.Plugins.Dir = "/nonexistent/plugin/directory"
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = validateCmd.Flags().Set("check-binaries", "true")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.Error(t, err)
	// Validating loader catches non-existent directory at load time.
	require.ErrorContains(t, err, "plugin directory")
	require.ErrorContains(t, err, "/nonexistent/plugin/directory")
	require.Empty(t, stderr.String())
}

func TestValidateCmd_VerboseOutput(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "auth-plugin",
				Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
				Required: ptrBool(true),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = validateCmd.Flags().Set("verbose", "true")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "Config structure valid")
	require.Contains(t, output, "Flows: request, response")
	require.Contains(t, output, "Required: true")
}

func TestValidateCmd_ConfigLoadError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		err: os.ErrNotExist,
	}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.Error(t, err)
	require.Empty(t, stdout.String())
}

func TestValidateCmd_InvalidCategory(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t, map[config.Category][]config.PluginEntry{})
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = validateCmd.Flags().Set(flagCategory, "invalid-category")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid category")
}

func TestValidateCmd_MultiplePluginsMultipleCategories(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "auth-plugin-1",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
			{
				Name:     "auth-plugin-2",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(false),
			},
		},
		config.CategoryValidation: {
			{
				Name:     "validator-1",
				Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
				Required: ptrBool(true),
			},
		},
		config.CategoryObservability: {
			{
				Name:     "logger",
				Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
				Required: ptrBool(false),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "All plugins validated successfully!")
	require.Contains(t, output, "Categories: 3")
	require.Contains(t, output, "Plugins: 4")
}

func TestValidateCmd_MixedValidAndInvalidPlugins(t *testing.T) {
	t.Parallel()

	plugins := map[config.Category][]config.PluginEntry{
		config.CategoryAuthentication: {
			{
				Name:     "valid-auth-plugin",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(true),
			},
			{
				Name:  "invalid-auth-plugin",
				Flows: nil, // Invalid.
			},
		},
		config.CategoryValidation: {
			{
				Name:     "valid-validator",
				Flows:    []config.Flow{config.FlowRequest},
				Required: ptrBool(false),
			},
		},
	}

	cfg := newTestConfig(t, plugins)
	loader := &mockLoader{cfg: cfg}

	base := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	err = executeCmd(t, validateCmd, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed with 1 error(s)")
	require.Empty(t, stderr.String())

	output := stdout.String()
	require.Contains(t, output, "Plugin 'valid-auth-plugin'")
	require.Contains(t, output, "✓ Valid")
	require.Contains(t, output, "Plugin 'invalid-auth-plugin'")
	require.Contains(t, output, "✗")
	require.Contains(t, output, "Plugins: 3")
	require.Contains(t, output, "Issues: 1")
}
