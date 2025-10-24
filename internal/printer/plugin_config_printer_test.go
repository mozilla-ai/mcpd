package printer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginConfigPrinter_Item(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   PluginConfigResult
		expected string
	}{
		{
			name: "configured directory",
			result: PluginConfigResult{
				Dir: "/path/to/plugins",
			},
			expected: "Plugin Configuration:\n" +
				"  Directory: /path/to/plugins\n",
		},
		{
			name: "empty directory shows not configured",
			result: PluginConfigResult{
				Dir: "",
			},
			expected: "Plugin Configuration:\n" +
				"  Directory: (not configured)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &PluginConfigPrinter{}

			err := printer.Item(&buf, tc.result)
			require.NoError(t, err)

			require.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestPluginConfigPrinter_HeaderFooter(t *testing.T) {
	t.Parallel()

	t.Run("custom header", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginConfigPrinter{}

		printer.SetHeader(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== HEADER ===\n"))
		})

		printer.Header(&buf, 1)
		require.Equal(t, "=== HEADER ===\n", buf.String())
	})

	t.Run("custom footer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginConfigPrinter{}

		printer.SetFooter(func(w io.Writer, count int) {
			_, _ = w.Write([]byte("=== FOOTER ===\n"))
		})

		printer.Footer(&buf, 1)
		require.Equal(t, "=== FOOTER ===\n", buf.String())
	})

	t.Run("no header when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginConfigPrinter{}

		printer.Header(&buf, 1)
		require.Empty(t, buf.String())
	})

	t.Run("no footer when not set", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		printer := &PluginConfigPrinter{}

		printer.Footer(&buf, 1)
		require.Empty(t, buf.String())
	})
}
