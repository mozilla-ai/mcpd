package mozilla_ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServer_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedID   string
		expectedName string
	}{
		{
			name:         "Normalize mixed case ID, keep original name",
			input:        `{"id": "GitHub-Server", "name": "GitHub Server", "description": "Test server"}`,
			expectedID:   "github-server",
			expectedName: "GitHub Server",
		},
		{
			name:         "Normalize uppercase ID, keep original name",
			input:        `{"id": "TIME_SERVER", "name": "Time Server", "description": "Test server"}`,
			expectedID:   "time_server",
			expectedName: "Time Server",
		},
		{
			name:         "Normalize ID with spaces, keep original name",
			input:        `{"id": " Mixed Case Server ", "name": "Mixed Case Server", "description": "Test server"}`,
			expectedID:   "mixed case server",
			expectedName: "Mixed Case Server",
		},
		{
			name:         "Already normalized ID unchanged, keep original name",
			input:        `{"id": "simple-server", "name": "Simple Server", "description": "Test server"}`,
			expectedID:   "simple-server",
			expectedName: "Simple Server",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var server Server
			err := json.Unmarshal([]byte(tc.input), &server)
			require.NoError(t, err)
			require.Equal(t, tc.expectedID, server.ID)
			require.Equal(t, tc.expectedName, server.Name)
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
