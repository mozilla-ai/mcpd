package printer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

func TestServerEntryPrinter_Item(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		entry          config.ServerEntry
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "basic server with tools only",
			entry: config.ServerEntry{
				Name:    "test-server",
				Package: "uvx::test-package@1.0.0",
				Tools:   []string{"tool1", "tool2", "tool3"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server' (version: 1.0.0)",
				"tools: tool1, tool2, tool3",
			},
			notExpected: []string{
				"environment variables",
				"positional arguments",
				"command line arguments",
				"boolean flags",
			},
		},
		{
			name: "server with environment variables",
			entry: config.ServerEntry{
				Name:            "test-server",
				Package:         "uvx::test-package@1.0.0",
				Tools:           []string{"tool1"},
				RequiredEnvVars: []string{"API_KEY", "SECRET_TOKEN"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server'",
				"! The following environment variables are required",
				"API_KEY",
				"SECRET_TOKEN",
				"see: mcpd config env set --help",
			},
		},
		{
			name: "server with positional arguments",
			entry: config.ServerEntry{
				Name:                   "test-server",
				Package:                "uvx::test-package@1.0.0",
				Tools:                  []string{"tool1"},
				RequiredPositionalArgs: []string{"input_file", "output_file"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server'",
				"❗ The following positional arguments are required for this server:",
				"📍 (1) input_file",
				"📍 (2) output_file",
				"see: mcpd config args set --help",
			},
		},
		{
			name: "server with value arguments",
			entry: config.ServerEntry{
				Name:              "test-server",
				Package:           "uvx::test-package@1.0.0",
				Tools:             []string{"tool1"},
				RequiredValueArgs: []string{"--host", "--port"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server'",
				"❗ The following command line arguments are required (along with values) for this server:",
				"🚩 --host",
				"🚩 --port",
				"see: mcpd config args set --help",
			},
		},
		{
			name: "server with boolean arguments",
			entry: config.ServerEntry{
				Name:             "test-server",
				Package:          "uvx::test-package@1.0.0",
				Tools:            []string{"tool1"},
				RequiredBoolArgs: []string{"--verbose", "--debug"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server'",
				"❗ The following command line arguments are required (as boolean flags) for this server:",
				"✅ --verbose",
				"✅ --debug",
				"see: mcpd config args set --help",
			},
		},
		{
			name: "server with all argument types",
			entry: config.ServerEntry{
				Name:                   "test-server",
				Package:                "uvx::test-package@1.0.0",
				Tools:                  []string{"tool1", "tool2"},
				RequiredEnvVars:        []string{"API_KEY"},
				RequiredPositionalArgs: []string{"input_file"},
				RequiredValueArgs:      []string{"--host"},
				RequiredBoolArgs:       []string{"--verbose"},
			},
			expectedOutput: []string{
				"✓ Added server 'test-server'",
				"tools: tool1, tool2",
				"! The following environment variables are required",
				"API_KEY",
				"❗ The following positional arguments are required for this server:",
				"📍 (1) input_file",
				"❗ The following command line arguments are required (along with values) for this server:",
				"🚩 --host",
				"❗ The following command line arguments are required (as boolean flags) for this server:",
				"✅ --verbose",
				"see: mcpd config env set --help",
				"see: mcpd config args set --help",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			printer := &ServerEntryPrinter{}
			var buf bytes.Buffer

			err := printer.Item(&buf, tc.entry)
			require.NoError(t, err)

			output := buf.String()

			// Check that all expected strings are present
			for _, expected := range tc.expectedOutput {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tc.notExpected {
				assert.NotContains(t, output, notExpected, "Output should not contain: %s", notExpected)
			}
		})
	}
}

func TestServerEntryPrinter_HeaderFooter(t *testing.T) {
	t.Parallel()

	t.Run("with header and footer", func(t *testing.T) {
		t.Parallel()

		printer := &ServerEntryPrinter{}

		headerCalled := false
		footerCalled := false

		printer.SetHeader(func(w io.Writer, count int) {
			headerCalled = true
			_, _ = fmt.Fprintf(w, "Header: %d items\n", count)
		})

		printer.SetFooter(func(w io.Writer, count int) {
			footerCalled = true
			_, _ = fmt.Fprintf(w, "Footer: %d items\n", count)
		})

		var buf bytes.Buffer

		printer.Header(&buf, 5)
		assert.True(t, headerCalled)
		assert.Contains(t, buf.String(), "Header: 5 items")

		buf.Reset()
		printer.Footer(&buf, 3)
		assert.True(t, footerCalled)
		assert.Contains(t, buf.String(), "Footer: 3 items")
	})

	t.Run("without header and footer", func(t *testing.T) {
		t.Parallel()

		printer := &ServerEntryPrinter{}
		var buf bytes.Buffer

		// Should not panic or write anything
		printer.Header(&buf, 5)
		printer.Footer(&buf, 3)

		assert.Empty(t, buf.String())
	})
}

func TestServerEntryPrinter_ArgumentOrder(t *testing.T) {
	t.Parallel()

	// Test that positional arguments are displayed before value args and bool args
	entry := config.ServerEntry{
		Name:                   "test-server",
		Package:                "uvx::test-package@1.0.0",
		Tools:                  []string{"tool1"},
		RequiredPositionalArgs: []string{"input", "output"},
		RequiredValueArgs:      []string{"--config"},
		RequiredBoolArgs:       []string{"--verbose"},
	}

	printer := &ServerEntryPrinter{}
	var buf bytes.Buffer

	err := printer.Item(&buf, entry)
	require.NoError(t, err)

	output := buf.String()

	// Find the positions of each argument type in the output
	posPos := strings.Index(output, "positional arguments")
	valPos := strings.Index(output, "along with values")
	boolPos := strings.Index(output, "boolean flags")

	// Verify that positional args appear before value args
	if posPos != -1 && valPos != -1 {
		assert.Less(t, posPos, valPos, "Positional arguments should appear before value arguments")
	}

	// Verify that value args appear before bool args
	if valPos != -1 && boolPos != -1 {
		assert.Less(t, valPos, boolPos, "Value arguments should appear before boolean arguments")
	}
}
