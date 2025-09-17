package mcpm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPServer_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normalize mixed case name",
			input:    `{"name": "GitHub-Server", "description": "Test server"}`,
			expected: "github-server",
		},
		{
			name:     "Normalize uppercase name",
			input:    `{"name": "TIME_SERVER", "description": "Test server"}`,
			expected: "time_server",
		},
		{
			name:     "Normalize name with spaces",
			input:    `{"name": " Mixed Case Server ", "description": "Test server"}`,
			expected: "mixed case server",
		},
		{
			name:     "Already normalized name unchanged",
			input:    `{"name": "simple-server", "description": "Test server"}`,
			expected: "simple-server",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var server MCPServer
			err := json.Unmarshal([]byte(tc.input), &server)
			require.NoError(t, err)
			require.Equal(t, tc.expected, server.Name)
			require.Equal(t, "Test server", server.Description)
		})
	}
}

func TestTool_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normalize mixed case tool name",
			input:    `{"name": "Create_Repository", "description": "Create a new repository"}`,
			expected: "create_repository",
		},
		{
			name:     "Normalize uppercase tool name",
			input:    `{"name": "LIST_ISSUES", "description": "List all issues"}`,
			expected: "list_issues",
		},
		{
			name:     "Normalize tool name with spaces",
			input:    `{"name": " Get Current Time ", "description": "Get the current time"}`,
			expected: "get current time",
		},
		{
			name:     "Already normalized tool name unchanged",
			input:    `{"name": "simple-tool", "description": "A simple tool"}`,
			expected: "simple-tool",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var tool Tool
			err := json.Unmarshal([]byte(tc.input), &tool)
			require.NoError(t, err)
			require.Equal(t, tc.expected, tool.Name)
		})
	}
}

func TestToolsSlice_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	input := `[
		{"name": "Create_Repository", "description": "Create a new repository"},
		{"name": "LIST_ISSUES", "description": "List all issues"}
	]`

	var tools []Tool
	err := json.Unmarshal([]byte(input), &tools)
	require.NoError(t, err)
	require.Len(t, tools, 2)
	require.Equal(t, "create_repository", tools[0].Name)
	require.Equal(t, "list_issues", tools[1].Name)
}
