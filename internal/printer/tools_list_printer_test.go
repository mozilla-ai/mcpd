package printer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToolsListPrinter_Item(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   ToolsListResult
		expected string
	}{
		{
			name: "server with multiple tools",
			result: ToolsListResult{
				Server: "test-server",
				Tools:  []string{"create_file", "read_file", "update_file"},
				Count:  3,
			},
			expected: "Tools for 'test-server' (3 total):\n  create_file\n  read_file\n  update_file\n",
		},
		{
			name: "server with single tool",
			result: ToolsListResult{
				Server: "single-tool-server",
				Tools:  []string{"list_items"},
				Count:  1,
			},
			expected: "Tools for 'single-tool-server' (1 total):\n  list_items\n",
		},
		{
			name: "server with no tools",
			result: ToolsListResult{
				Server: "empty-server",
				Tools:  []string{},
				Count:  0,
			},
			expected: "Tools for 'empty-server' (0 total):\n  (No tools configured)\n",
		},
		{
			name: "server with nil tools",
			result: ToolsListResult{
				Server: "nil-tools-server",
				Tools:  nil,
				Count:  0,
			},
			expected: "Tools for 'nil-tools-server' (0 total):\n  (No tools configured)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &ToolsListPrinter{}

			err := printer.Item(&buf, tc.result)
			require.NoError(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestToolsListPrinter_HeaderFooter(t *testing.T) {
	t.Parallel()

	t.Run("custom header", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &ToolsListPrinter{}

		printer.SetHeader(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== HEADER ===\n"))
		})

		printer.Header(&buf, 1)
		require.Equal(t, "=== HEADER ===\n", buf.String())
	})

	t.Run("custom footer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &ToolsListPrinter{}

		printer.SetFooter(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== FOOTER ===\n"))
		})

		printer.Footer(&buf, 1)
		require.Equal(t, "=== FOOTER ===\n", buf.String())
	})

	t.Run("no header when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &ToolsListPrinter{}

		printer.Header(&buf, 1)
		require.Empty(t, buf.String())
	})

	t.Run("no footer when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &ToolsListPrinter{}

		printer.Footer(&buf, 1)
		require.Empty(t, buf.String())
	})
}
