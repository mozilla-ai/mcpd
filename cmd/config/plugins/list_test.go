package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
)

// newTestConfig creates a config.Config with the given plugin map.
func newTestConfig(t *testing.T, plugins map[config.Category][]config.PluginEntry) *config.Config {
	t.Helper()

	cfg := &config.Config{
		Plugins: &config.PluginConfig{},
	}

	for cat, entries := range plugins {
		switch cat {
		case config.CategoryAuthentication:
			cfg.Plugins.Authentication = entries
		case config.CategoryAuthorization:
			cfg.Plugins.Authorization = entries
		case config.CategoryRateLimiting:
			cfg.Plugins.RateLimiting = entries
		case config.CategoryValidation:
			cfg.Plugins.Validation = entries
		case config.CategoryContent:
			cfg.Plugins.Content = entries
		case config.CategoryObservability:
			cfg.Plugins.Observability = entries
		case config.CategoryAudit:
			cfg.Plugins.Audit = entries
		}
	}

	return cfg
}

func TestNewListCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewListCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "list", c.Use)
	require.Contains(t, c.Short, "List configured plugin entries")
	require.NotNil(t, c.RunE)
}

func TestListCmd_SingleCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		category       string
		plugins        map[config.Category][]config.PluginEntry
		expectedOutput string
		expectedError  string
	}{
		{
			name:     "list plugins in authentication category",
			category: "authentication",
			plugins: map[config.Category][]config.PluginEntry{
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
						Name:     "validator",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
			},
			expectedOutput: "Configured plugins in 'authentication' (2 total):\n" +
				"  auth-plugin-1\n" +
				"    Flows: request\n" +
				"    Required: true\n" +
				"  auth-plugin-2\n" +
				"    Flows: request\n" +
				"    Required: false\n",
		},
		{
			name:     "list plugins in empty category",
			category: "validation",
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryAuthentication: {
					{
						Name:     "auth-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
			},
			expectedOutput: "Configured plugins in 'validation' (0 total):\n" +
				"  (No plugins configured)\n",
		},
		{
			name:     "list plugins with commit hash",
			category: "content",
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryContent: {
					{
						Name:       "content-filter",
						Flows:      []config.Flow{config.FlowResponse},
						Required:   ptrBool(true),
						CommitHash: ptrString("abc123"),
					},
				},
			},
			expectedOutput: "Configured plugins in 'content' (1 total):\n" +
				"  content-filter\n" +
				"    Flows: response\n" +
				"    Required: true\n" +
				"    Commit Hash: abc123\n",
		},
		{
			name:          "invalid category",
			category:      "foo",
			plugins:       map[config.Category][]config.PluginEntry{},
			expectedError: "invalid category 'foo'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, tc.plugins)
			loader := &mockLoader{cfg: cfg}

			base := &cmd.BaseCmd{}
			listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			// Set category flag.
			err = listCmd.Flags().Set("category", tc.category)
			if tc.expectedError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedError)
				return
			} else {
				require.NoError(t, err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			listCmd.SetOut(&stdout)
			listCmd.SetErr(&stderr)

			err = listCmd.RunE(listCmd, []string{})
			require.NoError(t, err)
			require.Empty(t, stderr.String())
			require.Equal(t, tc.expectedOutput, stdout.String())
		})
	}
}

func TestListCmd_AllCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plugins        map[config.Category][]config.PluginEntry
		expectedOutput string
	}{
		{
			name: "multiple categories with plugins",
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryObservability: {
					{
						Name:     "logger",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(false),
					},
				},
				config.CategoryAuthentication: {
					{
						Name:     "auth-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
				config.CategoryValidation: {
					{
						Name:     "validator-1",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(false),
					},
					{
						Name:     "validator-2",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
			},
			expectedOutput: "Configured plugins (4 total):\n" +
				"\nobservability (1 total):\n" +
				"  logger\n" +
				"    Flows: request, response\n" +
				"    Required: false\n" +
				"\nauthentication (1 total):\n" +
				"  auth-plugin\n" +
				"    Flows: request\n" +
				"    Required: true\n" +
				"\nvalidation (2 total):\n" +
				"  validator-1\n" +
				"    Flows: request\n" +
				"    Required: false\n" +
				"  validator-2\n" +
				"    Flows: request\n" +
				"    Required: true\n",
		},
		{
			name:           "no plugins in any category",
			plugins:        map[config.Category][]config.PluginEntry{},
			expectedOutput: "No plugins configured in any category\n",
		},
		{
			name:           "nil plugins map",
			plugins:        nil,
			expectedOutput: "No plugins configured in any category\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, tc.plugins)
			loader := &mockLoader{cfg: cfg}

			base := &cmd.BaseCmd{}
			listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			listCmd.SetOut(&stdout)
			listCmd.SetErr(&stderr)

			err = listCmd.RunE(listCmd, []string{})

			require.NoError(t, err)
			require.Empty(t, stderr.String())
			require.Equal(t, tc.expectedOutput, stdout.String())
		})
	}
}

func TestListCmd_JSONOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category config.Category
		plugins  map[config.Category][]config.PluginEntry
		validate func(t *testing.T, result printer.PluginListResult)
	}{
		{
			name:     "single category json output",
			category: config.CategoryAuthentication,
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryAuthentication: {
					{
						Name:     "auth-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
			},
			validate: func(t *testing.T, result printer.PluginListResult) {
				t.Helper()
				require.Len(t, result.Categories, 1)
				require.Contains(t, result.Categories, config.CategoryAuthentication)
				require.Len(t, result.Categories[config.CategoryAuthentication], 1)
				require.Equal(t, "auth-plugin", result.Categories[config.CategoryAuthentication][0].Name)
				require.Equal(t, 1, result.TotalPlugins)
			},
		},
		{
			name:     "all categories json output",
			category: "",
			plugins: map[config.Category][]config.PluginEntry{
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
			},
			validate: func(t *testing.T, result printer.PluginListResult) {
				t.Helper()
				require.Len(t, result.Categories, 2)
				require.Equal(t, 2, result.TotalPlugins)
				require.Contains(t, result.Categories, config.CategoryAuthentication)
				require.Contains(t, result.Categories, config.CategoryValidation)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, tc.plugins)
			loader := &mockLoader{cfg: cfg}

			base := &cmd.BaseCmd{}
			listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			err = listCmd.Flags().Set("format", "json")
			require.NoError(t, err)

			if tc.category != "" {
				err = listCmd.Flags().Set("category", string(tc.category))
				require.NoError(t, err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			listCmd.SetOut(&stdout)
			listCmd.SetErr(&stderr)

			err = listCmd.RunE(listCmd, []string{})
			require.NoError(t, err)

			require.Empty(t, stderr.String())
			var wrapper struct {
				Result printer.PluginListResult `json:"result"`
			}
			err = json.Unmarshal(stdout.Bytes(), &wrapper)
			require.NoError(t, err)

			tc.validate(t, wrapper.Result)
		})
	}
}

func TestListCmd_YAMLOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category config.Category
		plugins  map[config.Category][]config.PluginEntry
		validate func(t *testing.T, result printer.PluginListResult)
	}{
		{
			name:     "single category yaml output",
			category: config.CategoryAuthorization,
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryAuthorization: {
					{
						Name:     "authz-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(false),
					},
				},
			},
			validate: func(t *testing.T, result printer.PluginListResult) {
				t.Helper()
				require.Len(t, result.Categories, 1)
				require.Contains(t, result.Categories, config.CategoryAuthorization)
				require.Len(t, result.Categories[config.CategoryAuthorization], 1)
				require.Equal(t, "authz-plugin", result.Categories[config.CategoryAuthorization][0].Name)
				require.Equal(t, 1, result.TotalPlugins)
			},
		},
		{
			name:     "all categories yaml output",
			category: "",
			plugins: map[config.Category][]config.PluginEntry{
				config.CategoryObservability: {
					{
						Name:     "logger",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(false),
					},
				},
				config.CategoryAudit: {
					{
						Name:     "auditor",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
				},
			},
			validate: func(t *testing.T, result printer.PluginListResult) {
				t.Helper()
				require.Len(t, result.Categories, 2)
				require.Equal(t, 2, result.TotalPlugins)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, tc.plugins)
			loader := &mockLoader{cfg: cfg}

			base := &cmd.BaseCmd{}
			listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			err = listCmd.Flags().Set("format", "yaml")
			require.NoError(t, err)

			if tc.category != "" {
				err = listCmd.Flags().Set("category", string(tc.category))
				require.NoError(t, err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			listCmd.SetOut(&stdout)
			listCmd.SetErr(&stderr)

			err = listCmd.RunE(listCmd, []string{})
			require.NoError(t, err)

			require.Empty(t, stderr.String())
			var wrapper struct {
				Result printer.PluginListResult `yaml:"result"`
			}
			err = yaml.Unmarshal(stdout.Bytes(), &wrapper)
			require.NoError(t, err)

			tc.validate(t, wrapper.Result)
		})
	}
}

func TestListCmd_DistinctPluginCount(t *testing.T) {
	t.Parallel()

	t.Run("same plugin in multiple categories counted once", func(t *testing.T) {
		t.Parallel()

		// Same plugin name in different categories.
		plugins := map[config.Category][]config.PluginEntry{
			config.CategoryAuthentication: {
				{
					Name:     "shared-plugin",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(true),
				},
			},
			config.CategoryValidation: {
				{
					Name:     "shared-plugin",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(false),
				},
			},
		}

		cfg := newTestConfig(t, plugins)
		loader := &mockLoader{cfg: cfg}

		base := &cmd.BaseCmd{}
		listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)

		err = listCmd.Flags().Set("format", "json")
		require.NoError(t, err)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		listCmd.SetOut(&stdout)
		listCmd.SetErr(&stderr)

		err = listCmd.RunE(listCmd, []string{})
		require.NoError(t, err)

		require.Empty(t, stderr.String())
		var wrapper struct {
			Result printer.PluginListResult `json:"result"`
		}
		err = json.Unmarshal(stdout.Bytes(), &wrapper)
		require.NoError(t, err)

		// Should count distinct plugin names, so 1 not 2.
		require.Equal(t, 1, wrapper.Result.TotalPlugins)
	})

	t.Run("text output shows distinct count", func(t *testing.T) {
		t.Parallel()

		plugins := map[config.Category][]config.PluginEntry{
			config.CategoryAuthentication: {
				{
					Name:     "plugin-a",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(true),
				},
			},
			config.CategoryValidation: {
				{
					Name:     "plugin-a",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(false),
				},
				{
					Name:     "plugin-b",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(true),
				},
			},
		}

		cfg := newTestConfig(t, plugins)
		loader := &mockLoader{cfg: cfg}

		base := &cmd.BaseCmd{}
		listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		listCmd.SetOut(&stdout)
		listCmd.SetErr(&stderr)

		err = listCmd.RunE(listCmd, []string{})
		require.NoError(t, err)
		require.Empty(t, stderr.String())

		output := stdout.String()
		// Should show 2 distinct plugins in header (plugin-a and plugin-b).
		require.Contains(t, output, "Configured plugins (2 total):")
	})
}

func TestListCmd_ConfigLoadError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		err: fmt.Errorf("failed to load config"),
	}

	base := &cmd.BaseCmd{}
	listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	listCmd.SetOut(&stdout)
	listCmd.SetErr(&stderr)

	err = listCmd.RunE(listCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to load config")
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestListCmd_AllValidCategories(t *testing.T) {
	t.Parallel()

	// Test that all valid categories can be used.
	validCategories := []string{
		"authentication",
		"authorization",
		"rate_limiting",
		"validation",
		"content",
		"observability",
		"audit",
	}

	for _, category := range validCategories {
		t.Run(fmt.Sprintf("category_%s", category), func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig(t, map[config.Category][]config.PluginEntry{})
			loader := &mockLoader{cfg: cfg}

			base := &cmd.BaseCmd{}
			listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			err = listCmd.Flags().Set("category", category)
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			listCmd.SetOut(&stdout)
			listCmd.SetErr(&stderr)

			err = listCmd.RunE(listCmd, []string{})
			require.NoError(t, err)
			require.Empty(t, stderr.String())

			output := stdout.String()
			require.Contains(t, output, fmt.Sprintf("Configured plugins in '%s'", category))
		})
	}

	t.Run("category_case_insensitive", func(t *testing.T) {
		t.Parallel()
		cfg := newTestConfig(t, map[config.Category][]config.PluginEntry{})
		loader := &mockLoader{cfg: cfg}
		base := &cmd.BaseCmd{}
		listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)
		// Mixed case should be accepted.
		err = listCmd.Flags().Set("category", "AuThEnTiCaTiOn")
		require.NoError(t, err)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		listCmd.SetOut(&stdout)
		listCmd.SetErr(&stderr)
		err = listCmd.RunE(listCmd, []string{})
		require.NoError(t, err)
		require.Empty(t, stderr.String())
		require.Contains(t, stdout.String(), "Configured plugins in 'authentication'")
	})
}

func TestListCmd_CategoryExecutionOrder(t *testing.T) {
	t.Parallel()

	t.Run("categories displayed in execution order", func(t *testing.T) {
		t.Parallel()

		// Add plugins in all categories to verify order.
		plugins := map[config.Category][]config.PluginEntry{
			config.CategoryAudit: {
				{Name: "audit-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
			config.CategoryContent: {
				{Name: "content-plugin", Flows: []config.Flow{config.FlowResponse}, Required: ptrBool(false)},
			},
			config.CategoryValidation: {
				{Name: "validation-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
			config.CategoryRateLimiting: {
				{Name: "rate-limit-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
			config.CategoryAuthorization: {
				{Name: "authz-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
			config.CategoryAuthentication: {
				{Name: "auth-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
			config.CategoryObservability: {
				{Name: "observability-plugin", Flows: []config.Flow{config.FlowRequest}, Required: ptrBool(false)},
			},
		}

		cfg := newTestConfig(t, plugins)
		loader := &mockLoader{cfg: cfg}

		base := &cmd.BaseCmd{}
		listCmd, err := NewListCmd(base, cmdopts.WithConfigLoader(loader))
		require.NoError(t, err)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		listCmd.SetOut(&stdout)
		listCmd.SetErr(&stderr)

		err = listCmd.RunE(listCmd, []string{})
		require.NoError(t, err)
		require.Empty(t, stderr.String())

		output := stdout.String()

		// Verify categories appear in the expected execution order.
		expectedOrder := config.OrderedCategories()

		lastIndex := -1
		for _, category := range expectedOrder {
			needle := fmt.Sprintf("\n%s (", category)
			index := strings.Index(output, needle)
			require.Greater(t,
				index,
				lastIndex,
				"category %s should appear after previous category in output", category,
			)
			lastIndex = index
		}
	})
}

// Helper functions for test data.

func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
