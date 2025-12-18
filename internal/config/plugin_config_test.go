package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

func testPluginStringPtr(t *testing.T, s string) *string {
	t.Helper()
	return &s
}

func testPluginBoolPtr(t *testing.T, b bool) *bool {
	t.Helper()
	return &b
}

func TestPluginEntry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   PluginEntry
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid entry with single flow",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			wantErr: false,
		},
		{
			name: "valid entry with both flows",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest, FlowResponse},
			},
			wantErr: false,
		},
		{
			name: "valid entry with optional fields",
			entry: PluginEntry{
				Name:       "test-plugin",
				CommitHash: testPluginStringPtr(t, "abc123"),
				Required:   testPluginBoolPtr(t, true),
				Flows:      []Flow{FlowRequest},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			entry: PluginEntry{
				Name:  "",
				Flows: []Flow{FlowRequest},
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "whitespace name",
			entry: PluginEntry{
				Name:  "   ",
				Flows: []Flow{FlowRequest},
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "empty flows",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{},
			},
			wantErr: true,
			errMsg:  "at least one flow is required",
		},
		{
			name: "invalid flow value",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{"invalid"},
			},
			wantErr: true,
			errMsg:  "invalid flow",
		},
		{
			name: "duplicate flows",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest, FlowRequest},
			},
			wantErr: true,
			errMsg:  "duplicate flow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.entry.Validate()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPluginEntry_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry PluginEntry
		other *PluginEntry
		want  bool
	}{
		{
			name: "equal entries",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			other: &PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			want: true,
		},
		{
			name: "equal entries - different flow order",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest, FlowResponse},
			},
			other: &PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowResponse, FlowRequest},
			},
			want: true,
		},
		{
			name: "equal entries - duplicate flows",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest, FlowRequest},
			},
			other: &PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			want: true,
		},
		{
			name: "equal entries with all fields",
			entry: PluginEntry{
				Name:       "test-plugin",
				CommitHash: testPluginStringPtr(t, "abc123"),
				Required:   testPluginBoolPtr(t, true),
				Flows:      []Flow{FlowRequest, FlowResponse},
			},
			other: &PluginEntry{
				Name:       "test-plugin",
				CommitHash: testPluginStringPtr(t, "abc123"),
				Required:   testPluginBoolPtr(t, true),
				Flows:      []Flow{FlowRequest, FlowResponse},
			},
			want: true,
		},
		{
			name: "different names",
			entry: PluginEntry{
				Name:  "test-plugin-1",
				Flows: []Flow{FlowRequest},
			},
			other: &PluginEntry{
				Name:  "test-plugin-2",
				Flows: []Flow{FlowRequest},
			},
			want: false,
		},
		{
			name: "different commit hashes",
			entry: PluginEntry{
				Name:       "test-plugin",
				CommitHash: testPluginStringPtr(t, "abc123"),
				Flows:      []Flow{FlowRequest},
			},
			other: &PluginEntry{
				Name:       "test-plugin",
				CommitHash: testPluginStringPtr(t, "def456"),
				Flows:      []Flow{FlowRequest},
			},
			want: false,
		},
		{
			name: "different required values",
			entry: PluginEntry{
				Name:     "test-plugin",
				Required: testPluginBoolPtr(t, true),
				Flows:    []Flow{FlowRequest},
			},
			other: &PluginEntry{
				Name:     "test-plugin",
				Required: testPluginBoolPtr(t, false),
				Flows:    []Flow{FlowRequest},
			},
			want: false,
		},
		{
			name: "different flows",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			other: &PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowResponse},
			},
			want: false,
		},
		{
			name: "nil other",
			entry: PluginEntry{
				Name:  "test-plugin",
				Flows: []Flow{FlowRequest},
			},
			other: nil,
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.entry.Equals(tc.other)
			require.Equal(t, tc.want, result)
		})
	}
}

func TestPluginEntry_HasFlow(t *testing.T) {
	t.Parallel()

	entry := PluginEntry{
		Name:  "test-plugin",
		Flows: []Flow{FlowRequest, FlowResponse},
	}

	require.True(t, entry.HasFlow(FlowRequest))
	require.True(t, entry.HasFlow(FlowResponse))
	require.False(t, entry.HasFlow("invalid"))
}

func TestPluginEntry_FlowsDistinct(t *testing.T) {
	t.Parallel()

	entry := PluginEntry{
		Name:  "test-plugin",
		Flows: []Flow{FlowRequest, FlowResponse},
	}

	distinct := entry.FlowsDistinct()

	require.Len(t, distinct, 2)
	_, hasRequest := distinct[FlowRequest]
	require.True(t, hasRequest)
	_, hasResponse := distinct[FlowResponse]
	require.True(t, hasResponse)
}

func TestPluginConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *PluginConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config with single plugin",
			config: &PluginConfig{
				Authentication: []PluginEntry{
					{
						Name:  "jwt-auth",
						Flows: []Flow{FlowRequest},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple categories",
			config: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
				Observability: []PluginEntry{
					{Name: "metrics", Flows: []Flow{FlowRequest, FlowResponse}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid plugin in authentication",
			config: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "", Flows: []Flow{FlowRequest}},
				},
			},
			wantErr: true,
			errMsg:  "authentication",
		},
		{
			name: "invalid plugin in audit",
			config: &PluginConfig{
				Audit: []PluginEntry{
					{Name: "audit-log", Flows: []Flow{}},
				},
			},
			wantErr: true,
			errMsg:  "audit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePluginBinaries(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil plugins", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{Plugins: nil}
		err := ValidatePluginBinaries(cfg)

		require.NoError(t, err)
	})

	t.Run("returns nil for empty plugin dir", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Plugins: &PluginConfig{Dir: ""},
		}
		err := ValidatePluginBinaries(cfg)

		require.NoError(t, err)
	})

	t.Run("returns nil for valid plugin directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		cfg := &Config{
			Plugins: &PluginConfig{Dir: dir},
		}
		err := ValidatePluginBinaries(cfg)

		require.NoError(t, err)
	})

	t.Run("returns error for missing plugin binary", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: dir,
				Authentication: []PluginEntry{
					{Name: "missing-plugin", Flows: []Flow{FlowRequest}},
				},
			},
		}
		err := ValidatePluginBinaries(cfg)

		require.Error(t, err)
		require.Contains(t, err.Error(), "missing-plugin")
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: "/non/existent/directory",
			},
		}
		err := ValidatePluginBinaries(cfg)

		require.Error(t, err)
		require.Contains(t, err.Error(), "plugin directory")
	})

	t.Run("passes when configured plugins exist as executables", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		// Create executable file.
		pluginPath := filepath.Join(dir, "my-plugin")
		require.NoError(t, os.WriteFile(pluginPath, []byte("#!/bin/sh\n"), 0o755))

		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: dir,
				Authentication: []PluginEntry{
					{Name: "my-plugin", Flows: []Flow{FlowRequest}},
				},
			},
		}
		err := ValidatePluginBinaries(cfg)

		require.NoError(t, err)
	})

	t.Run("passes with multiple plugins across categories", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		// Create executable files.
		for _, name := range []string{"auth-plugin", "rate-plugin"} {
			pluginPath := filepath.Join(dir, name)
			require.NoError(t, os.WriteFile(pluginPath, []byte("#!/bin/sh\n"), 0o755))
		}

		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: dir,
				Authentication: []PluginEntry{
					{Name: "auth-plugin", Flows: []Flow{FlowRequest}},
				},
				RateLimiting: []PluginEntry{
					{Name: "rate-plugin", Flows: []Flow{FlowRequest}},
				},
			},
		}
		err := ValidatePluginBinaries(cfg)

		require.NoError(t, err)
	})
}

func TestPluginConfig_categorySlice(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		Authentication: []PluginEntry{{Name: "auth", Flows: []Flow{FlowRequest}}},
	}

	tests := []struct {
		name     string
		category Category
		wantErr  bool
	}{
		{name: "authentication", category: CategoryAuthentication, wantErr: false},
		{name: "authorization", category: CategoryAuthorization, wantErr: false},
		{name: "rate_limiting", category: CategoryRateLimiting, wantErr: false},
		{name: "validation", category: CategoryValidation, wantErr: false},
		{name: "content", category: CategoryContent, wantErr: false},
		{name: "observability", category: CategoryObservability, wantErr: false},
		{name: "audit", category: CategoryAudit, wantErr: false},
		{name: "unknown category", category: "unknown", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			slice, err := config.categorySlice(tc.category)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, slice)
			} else {
				require.NoError(t, err)
				require.NotNil(t, slice)
			}
		})
	}
}

func TestPluginConfig_upsertPlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		entry      PluginEntry
		wantResult context.UpsertResult
		wantErr    bool
		wantName   string
	}{
		{
			name:     "create new plugin",
			initial:  &PluginConfig{},
			category: CategoryAuthentication,
			entry: PluginEntry{
				Name:  "jwt-auth",
				Flows: []Flow{FlowRequest},
			},
			wantName:   "jwt-auth",
			wantResult: context.Created,
			wantErr:    false,
		},
		{
			name: "update existing plugin",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category: CategoryAuthentication,
			entry: PluginEntry{
				Name:  "jwt-auth",
				Flows: []Flow{FlowRequest, FlowResponse},
			},
			wantName:   "jwt-auth",
			wantResult: context.Updated,
			wantErr:    false,
		},
		{
			name: "noop when no changes",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category: CategoryAuthentication,
			entry: PluginEntry{
				Name:  "jwt-auth",
				Flows: []Flow{FlowRequest},
			},
			wantName:   "jwt-auth",
			wantResult: context.Noop,
			wantErr:    false,
		},
		{
			name:     "invalid entry",
			initial:  &PluginConfig{},
			category: CategoryAuthentication,
			entry: PluginEntry{
				Name:  "",
				Flows: []Flow{FlowRequest},
			},
			wantResult: context.Noop,
			wantErr:    true,
		},
		{
			name:     "invalid category",
			initial:  &PluginConfig{},
			category: "invalid",
			entry: PluginEntry{
				Name:  "test",
				Flows: []Flow{FlowRequest},
			},
			wantResult: context.Noop,
			wantErr:    true,
		},
		{
			name:     "trim whitespace",
			initial:  &PluginConfig{},
			category: CategoryAuthentication,
			entry: PluginEntry{
				Name:  " jwt-auth  ",
				Flows: []Flow{FlowRequest},
			},
			wantName:   "jwt-auth",
			wantResult: context.Created,
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.upsertPlugin(tc.category, tc.entry)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				updated, found := tc.initial.plugin(tc.category, tc.wantName)
				require.True(t, found)
				require.Equal(t, tc.wantName, updated.Name)
			}

			require.Equal(t, tc.wantResult, result)
		})
	}
}

func TestPluginConfig_deletePlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		pluginName string
		wantResult context.UpsertResult
		wantErr    bool
	}{
		{
			name: "delete existing plugin",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			wantResult: context.Deleted,
			wantErr:    false,
		},
		{
			name: "delete non-existing plugin",
			initial: &PluginConfig{
				Authentication: []PluginEntry{},
			},
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			wantResult: context.Noop,
			wantErr:    true,
		},
		{
			name:       "nil config",
			initial:    nil,
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			wantResult: context.Noop,
			wantErr:    true,
		},
		{
			name: "empty plugin name",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "",
			wantResult: context.Noop,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.deletePlugin(tc.category, tc.pluginName)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.wantResult, result)
		})
	}
}

func TestPluginConfig_listPlugins(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		Authentication: []PluginEntry{
			{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			{Name: "oauth2", Flows: []Flow{FlowRequest}},
		},
	}

	plugins := config.ListPlugins(CategoryAuthentication)
	require.Len(t, plugins, 2)

	// Verify it returns a copy.
	plugins[0].Name = "modified"
	require.Equal(t, "jwt-auth", config.Authentication[0].Name)
}

func TestPluginConfig_plugin(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		Authentication: []PluginEntry{
			{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
		},
	}

	// Find existing plugin.
	plugin, found := config.plugin(CategoryAuthentication, "jwt-auth")
	require.True(t, found)
	require.Equal(t, "jwt-auth", plugin.Name)

	// Not found.
	_, found = config.plugin(CategoryAuthentication, "nonexistent")
	require.False(t, found)

	// Nil config.
	var nilConfig *PluginConfig
	_, found = nilConfig.plugin(CategoryAuthentication, "jwt-auth")
	require.False(t, found)
}

func TestConfig_PluginMethods(t *testing.T) {
	t.Parallel()

	t.Run("UpsertPlugin creates plugin config if nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			configFilePath: t.TempDir() + "/test.toml",
		}

		entry := PluginEntry{
			Name:  "jwt-auth",
			Flows: []Flow{FlowRequest},
		}

		result, err := config.UpsertPlugin(CategoryAuthentication, entry)
		require.NoError(t, err)
		require.Equal(t, context.Created, result)
		require.NotNil(t, config.Plugins)
	})

	t.Run("DeletePlugin on nil config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			configFilePath: t.TempDir() + "/test.toml",
		}

		result, err := config.DeletePlugin(CategoryAuthentication, "jwt-auth")
		require.Error(t, err)
		require.Equal(t, context.Noop, result)
		require.Contains(t, err.Error(), "no plugins configured")
	})

	t.Run("ListPlugins on nil plugin config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			// When Plugins is nil, we expect nil return.
			Plugins: nil,
		}

		require.Nil(t, config.Plugins)
		require.Nil(t, config.Plugins.ListPlugins(CategoryAuthentication))
	})

	t.Run("Plugin on nil config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		_, found := config.Plugin(CategoryAuthentication, "jwt-auth")
		require.False(t, found)
	})
}

func TestConfig_validate_withPlugins(t *testing.T) {
	t.Parallel()

	t.Run("valid plugins pass validation", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Servers: []ServerEntry{
				{Name: "test", Package: "uvx::test@1.0.0"},
			},
			Plugins: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
		}

		err := config.validate()
		require.NoError(t, err)
	})

	t.Run("invalid plugins fail validation", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Servers: []ServerEntry{
				{Name: "test", Package: "uvx::test@1.0.0"},
			},
			Plugins: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "", Flows: []Flow{FlowRequest}},
				},
			},
		}

		err := config.validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "plugin configuration error")
	})
}

func TestPluginConfig_Validate_errorMessages(t *testing.T) {
	t.Parallel()

	t.Run("error message includes plugin name", func(t *testing.T) {
		t.Parallel()

		config := &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{}}, // Invalid: no flows.
			},
		}

		err := config.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "jwt-auth")
		require.Contains(t, err.Error(), "authentication")
		require.NotContains(t, err.Error(), "[0]")
	})

	t.Run("error message for unnamed plugin", func(t *testing.T) {
		t.Parallel()

		config := &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "", Flows: []Flow{FlowRequest}}, // Invalid: no name.
			},
		}

		err := config.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown")
		require.Contains(t, err.Error(), "authentication")
	})
}

func TestFlows(t *testing.T) {
	t.Parallel()

	flows := Flows()

	// Should contain exactly request and response.
	require.Len(t, flows, 2)
	require.Contains(t, flows, FlowRequest)
	require.Contains(t, flows, FlowResponse)

	// Verify that modifications don't affect subsequent calls (clone behavior).
	delete(flows, FlowRequest)
	require.Len(t, flows, 1)

	// Get a fresh copy - should still have both flows.
	freshFlows := Flows()
	require.Len(t, freshFlows, 2)
	require.Contains(t, freshFlows, FlowRequest)
	require.Contains(t, freshFlows, FlowResponse)
}

func TestFlow_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flow  Flow
		valid bool
	}{
		{
			name:  "valid request flow",
			flow:  FlowRequest,
			valid: true,
		},
		{
			name:  "valid response flow",
			flow:  FlowResponse,
			valid: true,
		},
		{
			name:  "invalid empty flow",
			flow:  Flow(""),
			valid: false,
		},
		{
			name:  "invalid unknown flow",
			flow:  Flow("unknown"),
			valid: false,
		},
		{
			name:  "invalid uppercase",
			flow:  Flow("REQUEST"),
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.flow.IsValid()
			require.Equal(t, tc.valid, result)
		})
	}
}

func TestParseFlowsDistinct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected map[Flow]struct{}
	}{
		{
			name:  "single valid flow",
			input: []string{"request"},
			expected: map[Flow]struct{}{
				FlowRequest: {},
			},
		},
		{
			name:  "two valid flows",
			input: []string{"request", "response"},
			expected: map[Flow]struct{}{
				FlowRequest:  {},
				FlowResponse: {},
			},
		},
		{
			name:  "duplicates are deduplicated",
			input: []string{"request", "request", "response", "response"},
			expected: map[Flow]struct{}{
				FlowRequest:  {},
				FlowResponse: {},
			},
		},
		{
			name:     "invalid flows are ignored",
			input:    []string{"invalid", "foo", "bar"},
			expected: map[Flow]struct{}{},
		},
		{
			name:  "mixed valid and invalid",
			input: []string{"request", "invalid", "response", "foo"},
			expected: map[Flow]struct{}{
				FlowRequest:  {},
				FlowResponse: {},
			},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[Flow]struct{}{},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: map[Flow]struct{}{},
		},
		{
			name:  "case insensitive",
			input: []string{"REQUEST", "Response", "REQUEST"},
			expected: map[Flow]struct{}{
				FlowRequest:  {},
				FlowResponse: {},
			},
		},
		{
			name:  "with whitespace",
			input: []string{" request ", "  response  "},
			expected: map[Flow]struct{}{
				FlowRequest:  {},
				FlowResponse: {},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ParseFlowsDistinct(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestAddCmd_OrderedFlowNames(t *testing.T) {
	flows := OrderedFlowNames()
	require.Len(t, flows, 2)
	require.Equal(t, "request", flows[0])
	require.Equal(t, "response", flows[1])
}

func TestPluginConfig_moveToCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		from       Category
		pluginName string
		to         Category
		force      bool
		wantResult context.UpsertResult
		wantErr    bool
		errMsg     string
	}{
		{
			name: "move to different category",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			from:       CategoryAuthentication,
			pluginName: "jwt-auth",
			to:         CategoryAudit,
			wantResult: context.Updated,
		},
		{
			name: "plugin not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "other", Flows: []Flow{FlowRequest}},
				},
			},
			from:       CategoryAuthentication,
			pluginName: "nonexistent",
			to:         CategoryAudit,
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "not found",
		},
		{
			name: "duplicate without force",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "shared", Flows: []Flow{FlowRequest}},
				},
				Audit: []PluginEntry{
					{Name: "shared", Flows: []Flow{FlowResponse}},
				},
			},
			from:       CategoryAuthentication,
			pluginName: "shared",
			to:         CategoryAudit,
			force:      false,
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "already exists",
		},
		{
			name: "duplicate with force",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "shared", Flows: []Flow{FlowRequest}},
				},
				Audit: []PluginEntry{
					{Name: "shared", Flows: []Flow{FlowResponse}},
				},
			},
			from:       CategoryAuthentication,
			pluginName: "shared",
			to:         CategoryAudit,
			force:      true,
			wantResult: context.Updated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.moveToCategory(tc.from, tc.pluginName, tc.to, tc.force)

			require.Equal(t, tc.wantResult, result)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				// Verify plugin moved.
				_, found := tc.initial.plugin(tc.from, tc.pluginName)
				require.False(t, found)
				_, found = tc.initial.plugin(tc.to, tc.pluginName)
				require.True(t, found)
			}
		})
	}
}

func TestPluginConfig_moveBefore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		pluginName string
		targetName string
		wantResult context.UpsertResult
		wantErr    bool
		errMsg     string
		wantOrder  []string
	}{
		{
			name: "move before target",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
					{Name: "c", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "c",
			targetName: "a",
			wantResult: context.Updated,
			wantOrder:  []string{"c", "a", "b"},
		},
		{
			name: "already in position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			targetName: "b",
			wantResult: context.Noop,
			wantOrder:  []string{"a", "b"},
		},
		{
			name: "plugin not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "nonexistent",
			targetName: "a",
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "not found",
		},
		{
			name: "target not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			targetName: "nonexistent",
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "target plugin",
		},
		{
			name: "single plugin",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			targetName: "a",
			wantResult: context.Noop,
			wantOrder:  []string{"a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.moveBefore(tc.category, tc.pluginName, tc.targetName)

			require.Equal(t, tc.wantResult, result)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				plugins := tc.initial.ListPlugins(tc.category)
				for i, want := range tc.wantOrder {
					require.Equal(t, want, plugins[i].Name)
				}
			}
		})
	}
}

func TestPluginConfig_moveAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		pluginName string
		targetName string
		wantResult context.UpsertResult
		wantErr    bool
		errMsg     string
		wantOrder  []string
	}{
		{
			name: "move after target",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
					{Name: "c", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			targetName: "b",
			wantResult: context.Updated,
			wantOrder:  []string{"b", "a", "c"},
		},
		{
			name: "already in position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "b",
			targetName: "a",
			wantResult: context.Noop,
			wantOrder:  []string{"a", "b"},
		},
		{
			name: "plugin not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "nonexistent",
			targetName: "a",
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "not found",
		},
		{
			name: "target not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			targetName: "nonexistent",
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "target plugin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.moveAfter(tc.category, tc.pluginName, tc.targetName)

			require.Equal(t, tc.wantResult, result)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				plugins := tc.initial.ListPlugins(tc.category)
				for i, want := range tc.wantOrder {
					require.Equal(t, want, plugins[i].Name)
				}
			}
		})
	}
}

func TestPluginConfig_moveToPosition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		pluginName string
		position   int
		wantResult context.UpsertResult
		wantErr    bool
		errMsg     string
		wantOrder  []string
	}{
		{
			name: "move to first position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
					{Name: "c", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "c",
			position:   1,
			wantResult: context.Updated,
			wantOrder:  []string{"c", "a", "b"},
		},
		{
			name: "move to last position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
					{Name: "c", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			position:   3,
			wantResult: context.Updated,
			wantOrder:  []string{"b", "c", "a"},
		},
		{
			name: "move to end with -1",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
					{Name: "c", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			position:   -1,
			wantResult: context.Updated,
			wantOrder:  []string{"b", "c", "a"},
		},
		{
			name: "already at position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			position:   1,
			wantResult: context.Noop,
			wantOrder:  []string{"a", "b"},
		},
		{
			name: "position clamped to max",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			position:   100,
			wantResult: context.Updated,
			wantOrder:  []string{"b", "a"},
		},
		{
			name: "plugin not found",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "nonexistent",
			position:   1,
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "not found",
		},
		{
			name: "single plugin",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			position:   1,
			wantResult: context.Noop,
			wantOrder:  []string{"a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.moveToPosition(tc.category, tc.pluginName, tc.position)

			require.Equal(t, tc.wantResult, result)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				plugins := tc.initial.ListPlugins(tc.category)
				for i, want := range tc.wantOrder {
					require.Equal(t, want, plugins[i].Name)
				}
			}
		})
	}
}

func TestPluginConfig_movePlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initial    *PluginConfig
		category   Category
		pluginName string
		opts       []MoveOption
		wantResult context.UpsertResult
		wantErr    bool
		errMsg     string
	}{
		{
			name: "move to category",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			opts:       []MoveOption{WithToCategory(CategoryAudit)},
			wantResult: context.Updated,
		},
		{
			name: "move before",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "b",
			opts:       []MoveOption{WithBefore("a")},
			wantResult: context.Updated,
		},
		{
			name: "move after",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "a",
			opts:       []MoveOption{WithAfter("b")},
			wantResult: context.Updated,
		},
		{
			name: "move to position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "a", Flows: []Flow{FlowRequest}},
					{Name: "b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "b",
			opts:       []MoveOption{WithPosition(1)},
			wantResult: context.Updated,
		},
		{
			name: "move to category and position",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
				Audit: []PluginEntry{
					{Name: "audit-a", Flows: []Flow{FlowRequest}},
					{Name: "audit-b", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			opts:       []MoveOption{WithToCategory(CategoryAudit), WithPosition(1)},
			wantResult: context.Updated,
		},
		{
			name: "no operation specified",
			initial: &PluginConfig{
				Authentication: []PluginEntry{
					{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				},
			},
			category:   CategoryAuthentication,
			pluginName: "jwt-auth",
			opts:       []MoveOption{},
			wantResult: context.Noop,
			wantErr:    true,
			errMsg:     "no move operation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.initial.movePlugin(tc.category, tc.pluginName, tc.opts...)

			require.Equal(t, tc.wantResult, result)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMoveOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithToCategory", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(WithToCategory(CategoryAudit))
		require.NoError(t, err)
		require.NotNil(t, opts.toCategory)
		require.Equal(t, CategoryAudit, *opts.toCategory)
	})

	t.Run("WithBefore", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(WithBefore("target"))
		require.NoError(t, err)
		require.NotNil(t, opts.before)
		require.Equal(t, "target", *opts.before)
	})

	t.Run("WithAfter", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(WithAfter("target"))
		require.NoError(t, err)
		require.NotNil(t, opts.after)
		require.Equal(t, "target", *opts.after)
	})

	t.Run("WithPosition", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(WithPosition(3))
		require.NoError(t, err)
		require.NotNil(t, opts.position)
		require.Equal(t, 3, *opts.position)
	})

	t.Run("WithForce", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(WithForce(true))
		require.NoError(t, err)
		require.True(t, opts.force)
	})

	t.Run("multiple options", func(t *testing.T) {
		t.Parallel()

		opts, err := newMoveOptions(
			WithToCategory(CategoryAudit),
			WithPosition(1),
			WithForce(true),
		)
		require.NoError(t, err)
		require.NotNil(t, opts.toCategory)
		require.Equal(t, CategoryAudit, *opts.toCategory)
		require.NotNil(t, opts.position)
		require.Equal(t, 1, *opts.position)
		require.True(t, opts.force)
	})
}
