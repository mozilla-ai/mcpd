package printer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestPluginEntryPrinter_Item(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   PluginEntryResult
		expected string
	}{
		{
			name: "plugin with all fields",
			result: PluginEntryResult{
				PluginEntry: config.PluginEntry{
					Name:       "jwt-auth",
					Flows:      []config.Flow{config.FlowRequest, config.FlowResponse},
					Required:   ptrBool(true),
					CommitHash: ptrString("abc123"),
				},
				Category: "authentication",
			},
			expected: "Plugin 'jwt-auth' in category 'authentication':\n" +
				"  Flows: request, response\n" +
				"  Required: true\n" +
				"  Commit Hash: abc123\n",
		},
		{
			name: "plugin without commit hash",
			result: PluginEntryResult{
				PluginEntry: config.PluginEntry{
					Name:     "rate-limiter",
					Flows:    []config.Flow{config.FlowRequest},
					Required: ptrBool(false),
				},
				Category: "rate_limiting",
			},
			expected: "Plugin 'rate-limiter' in category 'rate_limiting':\n" +
				"  Flows: request\n" +
				"  Required: false\n",
		},
		{
			name: "plugin with nil required defaults to false",
			result: PluginEntryResult{
				PluginEntry: config.PluginEntry{
					Name:     "logger",
					Flows:    []config.Flow{config.FlowRequest, config.FlowResponse},
					Required: nil,
				},
				Category: "observability",
			},
			expected: "Plugin 'logger' in category 'observability':\n" +
				"  Flows: request, response\n" +
				"  Required: false\n",
		},
		{
			name: "plugin with single flow",
			result: PluginEntryResult{
				PluginEntry: config.PluginEntry{
					Name:     "validator",
					Flows:    []config.Flow{config.FlowResponse},
					Required: ptrBool(true),
				},
				Category: "validation",
			},
			expected: "Plugin 'validator' in category 'validation':\n" +
				"  Flows: response\n" +
				"  Required: true\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &PluginEntryPrinter{}

			err := printer.Item(&buf, tc.result)
			require.NoError(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestPluginEntryPrinter_HeaderFooter(t *testing.T) {
	t.Parallel()

	t.Run("custom header", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginEntryPrinter{}

		printer.SetHeader(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== HEADER ===\n"))
		})

		printer.Header(&buf, 1)
		require.Equal(t, "=== HEADER ===\n", buf.String())
	})

	t.Run("custom footer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginEntryPrinter{}

		printer.SetFooter(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== FOOTER ===\n"))
		})

		printer.Footer(&buf, 1)
		require.Equal(t, "=== FOOTER ===\n", buf.String())
	})

	t.Run("no header when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginEntryPrinter{}

		printer.Header(&buf, 1)
		require.Empty(t, buf.String())
	})

	t.Run("no footer when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginEntryPrinter{}

		printer.Footer(&buf, 1)
		require.Empty(t, buf.String())
	})
}
