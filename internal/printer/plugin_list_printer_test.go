package printer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

func TestPluginListPrinter_Item_SingleCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   PluginListResult
		expected string
	}{
		{
			name: "category with multiple plugins",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryAuthentication: {
					{
						Name:     "auth-plugin-1",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
					{
						Name:     "auth-plugin-2",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(false),
					},
				},
			}),
			expected: "Configured plugins in 'authentication' (2 total):\n" +
				"  auth-plugin-1\n" +
				"    Flows: request\n" +
				"    Required: true\n" +
				"  auth-plugin-2\n" +
				"    Flows: request, response\n" +
				"    Required: false\n",
		},
		{
			name: "category with single plugin",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryValidation: {
					{
						Name:     "validator",
						Flows:    []config.Flow{config.FlowRequest},
						Required: nil,
					},
				},
			}),
			expected: "Configured plugins in 'validation' (1 total):\n" +
				"  validator\n" +
				"    Flows: request\n" +
				"    Required: false\n",
		},
		{
			name: "category with no plugins",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryObservability: {},
			}),
			expected: "Configured plugins in 'observability' (0 total):\n" +
				"  (No plugins configured)\n",
		},
		{
			name: "category with nil plugins",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryAudit: nil,
			}),
			expected: "Configured plugins in 'audit' (0 total):\n" +
				"  (No plugins configured)\n",
		},
		{
			name: "plugin with commit hash",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryContent: {
					{
						Name:       "content-filter",
						Flows:      []config.Flow{config.FlowResponse},
						Required:   ptrBool(true),
						CommitHash: ptrString("abc123def456"),
					},
				},
			}),
			expected: "Configured plugins in 'content' (1 total):\n" +
				"  content-filter\n" +
				"    Flows: response\n" +
				"    Required: true\n" +
				"    Commit Hash: abc123def456\n",
		},
		{
			name: "plugin with multiple flows",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryObservability: {
					{
						Name:     "logger",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(false),
					},
				},
			}),
			expected: "Configured plugins in 'observability' (1 total):\n" +
				"  logger\n" +
				"    Flows: request, response\n" +
				"    Required: false\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &PluginListPrinter{}

			err := printer.Item(&buf, tc.result)
			require.NoError(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestPluginListPrinter_Item_AllCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   PluginListResult
		expected string
	}{
		{
			name: "multiple categories with plugins in execution order",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
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
				config.CategoryObservability: {
					{
						Name:     "logger",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(false),
					},
				},
			}),
			expected: "Configured plugins (4 total):\n" +
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
			name: "single category with one plugin",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryAudit: {
					{
						Name:     "audit-logger",
						Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
						Required: ptrBool(true),
					},
				},
			}),
			expected: "Configured plugins in 'audit' (1 total):\n" +
				"  audit-logger\n" +
				"    Flows: request, response\n" +
				"    Required: true\n",
		},
		{
			name:     "empty categories map",
			result:   NewPluginListResult(map[config.Category][]config.PluginEntry{}),
			expected: "No plugins configured in any category\n",
		},
		{
			name:     "nil categories map",
			result:   NewPluginListResult(nil),
			expected: "No plugins configured in any category\n",
		},
		{
			name: "all categories with distinct count",
			result: NewPluginListResult(map[config.Category][]config.PluginEntry{
				config.CategoryObservability: {
					{
						Name:     "shared-plugin",
						Flows:    []config.Flow{config.FlowRequest},
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
				config.CategoryAuthorization: {
					{
						Name:     "authz-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(true),
					},
					{
						Name:     "shared-plugin",
						Flows:    []config.Flow{config.FlowRequest},
						Required: ptrBool(false),
					},
				},
			}),
			expected: "Configured plugins (3 total):\n" +
				"\nobservability (1 total):\n" +
				"  shared-plugin\n" +
				"    Flows: request\n" +
				"    Required: false\n" +
				"\nauthentication (1 total):\n" +
				"  auth-plugin\n" +
				"    Flows: request\n" +
				"    Required: true\n" +
				"\nauthorization (2 total):\n" +
				"  authz-plugin\n" +
				"    Flows: request\n" +
				"    Required: true\n" +
				"  shared-plugin\n" +
				"    Flows: request\n" +
				"    Required: false\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &PluginListPrinter{}

			err := printer.Item(&buf, tc.result)
			require.NoError(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestPluginListPrinter_HeaderFooter(t *testing.T) {
	t.Parallel()

	t.Run("custom header", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginListPrinter{}

		printer.SetHeader(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== HEADER ===\n"))
		})

		printer.Header(&buf, 1)
		require.Equal(t, "=== HEADER ===\n", buf.String())
	})

	t.Run("custom footer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginListPrinter{}

		printer.SetFooter(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== FOOTER ===\n"))
		})

		printer.Footer(&buf, 1)
		require.Equal(t, "=== FOOTER ===\n", buf.String())
	})

	t.Run("no header when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginListPrinter{}

		printer.Header(&buf, 1)
		require.Empty(t, buf.String())
	})

	t.Run("no footer when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginListPrinter{}

		printer.Footer(&buf, 1)
		require.Empty(t, buf.String())
	})
}

func TestFormatFlows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flows    []config.Flow
		expected string
	}{
		{
			name:     "single flow",
			flows:    []config.Flow{config.FlowRequest},
			expected: "request",
		},
		{
			name:     "multiple flows",
			flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
			expected: "request, response",
		},
		{
			name:     "empty flows",
			flows:    []config.Flow{},
			expected: "",
		},
		{
			name:     "nil flows",
			flows:    nil,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := formatFlows(tc.flows)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		required *bool
		expected string
	}{
		{
			name:     "required true",
			required: ptrBool(true),
			expected: "true",
		},
		{
			name:     "required false",
			required: ptrBool(false),
			expected: "false",
		},
		{
			name:     "required nil defaults to false",
			required: nil,
			expected: "false",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := formatRequired(tc.required)
			require.Equal(t, tc.expected, result)
		})
	}
}

// Helper functions for test data.

func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
