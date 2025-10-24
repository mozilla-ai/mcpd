package plugins

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
)

func TestGetCmd_FlagConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, "category", flagCategory)
	require.Equal(t, "name", flagName)
}

func TestNewGetCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewGetCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "get", c.Use)
	require.Contains(t, c.Short, "Get top level configuration for the plugin subsystem")
	require.NotNil(t, c.RunE)
}

func TestGetCmd_PluginLevelConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupConfig    func(*config.Config)
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name: "configured directory",
			setupConfig: func(cfg *config.Config) {
				cfg.Plugins = &config.PluginConfig{
					Dir: "/path/to/plugins",
				}
			},
			expectedOutput: "Plugin Configuration:\n  Directory: /path/to/plugins\n",
		},
		{
			name: "no plugin config",
			setupConfig: func(cfg *config.Config) {
				cfg.Plugins = nil
			},
			expectError:    false,
			expectedOutput: "Plugin Configuration:\n  Directory: (not configured)\n",
		},
		{
			name: "empty directory",
			setupConfig: func(cfg *config.Config) {
				cfg.Plugins = &config.PluginConfig{
					Dir: "",
				}
			},
			expectedOutput: "Plugin Configuration:\n  Directory: (not configured)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			tc.setupConfig(cfg)

			mockLoader := &mockLoader{cfg: cfg}
			base := &cmd.BaseCmd{}
			getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			getCmd.SetOut(&stdout)
			getCmd.SetErr(&stderr)

			err = getCmd.RunE(getCmd, []string{})

			if tc.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Empty(t, stderr.String())
				require.Equal(t, tc.expectedOutput, stdout.String())
			}
		})
	}
}

func TestGetCmd_PluginLevelConfig_JSON(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Dir: "/path/to/plugins",
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set("format", "json")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	var wrapper struct {
		Result printer.PluginConfigResult `json:"result"`
	}
	err = json.Unmarshal(stdout.Bytes(), &wrapper)
	require.NoError(t, err)

	require.Equal(t, "/path/to/plugins", wrapper.Result.Dir)
}

func TestGetCmd_PluginLevelConfig_YAML(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Dir: "/path/to/plugins",
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set("format", "yaml")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	var wrapper struct {
		Result printer.PluginConfigResult `yaml:"result"`
	}
	err = yaml.Unmarshal(stdout.Bytes(), &wrapper)
	require.NoError(t, err)

	require.Equal(t, "/path/to/plugins", wrapper.Result.Dir)
}

func TestGetCmd_SpecificPluginEntry(t *testing.T) {
	t.Parallel()

	requiredTrue := true
	commitHash := "abc123"

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Authentication: []config.PluginEntry{
				{
					Name:       "jwt-auth",
					Flows:      []config.Flow{config.FlowRequest, config.FlowResponse},
					Required:   &requiredTrue,
					CommitHash: &commitHash,
				},
			},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	expectedOutput := "Plugin 'jwt-auth' in category 'authentication':\n" +
		"  Flows: request, response\n" +
		"  Required: true\n" +
		"  Commit Hash: abc123\n"

	require.Equal(t, expectedOutput, stdout.String())
}

func TestGetCmd_SpecificPluginEntry_JSON(t *testing.T) {
	t.Parallel()

	requiredFalse := false

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Observability: []config.PluginEntry{
				{
					Name:     "logger",
					Flows:    []config.Flow{config.FlowRequest},
					Required: &requiredFalse,
				},
			},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "observability")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "logger")
	require.NoError(t, err)
	err = getCmd.Flags().Set("format", "json")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	var wrapper struct {
		Result printer.PluginEntryResult `json:"result"`
	}
	err = json.Unmarshal(stdout.Bytes(), &wrapper)
	require.NoError(t, err)

	require.Equal(t, "logger", wrapper.Result.Name)
	require.Equal(t, config.CategoryObservability, wrapper.Result.Category)
	require.Equal(t, []config.Flow{config.FlowRequest}, wrapper.Result.Flows)
	require.NotNil(t, wrapper.Result.Required)
	require.False(t, *wrapper.Result.Required)
	require.Nil(t, wrapper.Result.CommitHash)
}

func TestGetCmd_SpecificPluginEntry_YAML(t *testing.T) {
	t.Parallel()

	requiredTrue := true

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Audit: []config.PluginEntry{
				{
					Name:     "compliance",
					Flows:    []config.Flow{config.FlowResponse},
					Required: &requiredTrue,
				},
			},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "audit")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "compliance")
	require.NoError(t, err)
	err = getCmd.Flags().Set("format", "yaml")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	var wrapper struct {
		Result printer.PluginEntryResult `yaml:"result"`
	}
	err = yaml.Unmarshal(stdout.Bytes(), &wrapper)
	require.NoError(t, err)

	require.Equal(t, "compliance", wrapper.Result.Name)
	require.Equal(t, config.CategoryAudit, wrapper.Result.Category)
	require.Equal(t, []config.Flow{config.FlowResponse}, wrapper.Result.Flows)
	require.NotNil(t, wrapper.Result.Required)
	require.True(t, *wrapper.Result.Required)
}

func TestGetCmd_FlagValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		category      string
		pluginName    string
		expectedError string
	}{
		{
			name:          "only category provided",
			category:      "authentication",
			pluginName:    "",
			expectedError: "must be provided together or not at all",
		},
		{
			name:          "only name provided",
			category:      "",
			pluginName:    "jwt-auth",
			expectedError: "must be provided together or not at all",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}

			mockLoader := &mockLoader{cfg: cfg}
			base := &cmd.BaseCmd{}
			getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
			require.NoError(t, err)

			if tc.category != "" {
				err = getCmd.Flags().Set(flagCategory, tc.category)
				require.NoError(t, err)
			}
			if tc.pluginName != "" {
				err = getCmd.Flags().Set(flagName, tc.pluginName)
				require.NoError(t, err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			getCmd.SetOut(&stdout)
			getCmd.SetErr(&stderr)

			err = getCmd.RunE(getCmd, []string{})
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expectedError)
		})
	}
}

func TestGetCmd_InvalidCategory(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "invalid-category")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid category 'invalid-category'")
}

func TestGetCmd_PluginNotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			Authentication: []config.PluginEntry{
				{
					Name:  "other-plugin",
					Flows: []config.Flow{config.FlowRequest},
				},
			},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin 'jwt-auth' not found in category 'authentication'")
}

func TestGetCmd_PluginNotFound_NilPluginConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: nil,
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.Error(t, err)
	require.EqualError(t, err, "plugin 'jwt-auth' not found in category 'authentication'")
}

func TestGetCmd_CaseInsensitiveCategory(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{
			RateLimiting: []config.PluginEntry{
				{
					Name:  "rate-limiter",
					Flows: []config.Flow{config.FlowRequest},
				},
			},
		},
	}

	mockLoader := &mockLoader{cfg: cfg}
	base := &cmd.BaseCmd{}
	getCmd, err := NewGetCmd(base, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	err = getCmd.Flags().Set(flagCategory, "RATE_LIMITING")
	require.NoError(t, err)
	err = getCmd.Flags().Set(flagName, "rate-limiter")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	getCmd.SetOut(&stdout)
	getCmd.SetErr(&stderr)

	err = getCmd.RunE(getCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	require.Contains(t, stdout.String(), "Plugin 'rate-limiter' in category 'rate_limiting'")
}
