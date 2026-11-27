package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		given    ServerName
		key      string
		reported string
	}{
		{
			name:     "pascal case",
			given:    "GitHub",
			key:      "github",
			reported: "GitHub",
		},
		{
			name:     "camel case",
			given:    "gitHub",
			key:      "github",
			reported: "gitHub",
		},
		{
			name:     "surrounding whitespace",
			given:    "  GitHub-Server  ",
			key:      "github-server",
			reported: "GitHub-Server",
		},
		{
			name:     "already lowercase",
			given:    "github",
			key:      "github",
			reported: "github",
		},
		{
			name:     "empty",
			given:    "",
			key:      "",
			reported: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.key, tc.given.Key())
			require.Equal(t, tc.reported, tc.given.String())
		})
	}
}

// TestServerName_KeyIsTheLowercasedReportedName pins the one relationship the two forms have to keep.
// Key indexes a name and String reports it, so a key that stopped being the lowercased reported name would
// let a stored name drift out of reach of the entry it was stored under.
func TestServerName_KeyIsTheLowercasedReportedName(t *testing.T) {
	t.Parallel()

	names := []ServerName{
		"",
		"  padded  ",
		"GitHub",
		"MiXeD-Case_Name.9",
		"github",
	}

	for _, name := range names {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()

			require.Equal(t, strings.ToLower(name.String()), name.Key())
		})
	}
}

func TestToolName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		given    ToolName
		key      string
		reported string
	}{
		{
			name:     "camel case",
			given:    "getCurrentTime",
			key:      "getcurrenttime",
			reported: "getCurrentTime",
		},
		{
			name:     "screaming snake case",
			given:    "GET_CURRENT_TIME",
			key:      "get_current_time",
			reported: "GET_CURRENT_TIME",
		},
		{
			name:     "surrounding whitespace",
			given:    "  getCurrentTime  ",
			key:      "getcurrenttime",
			reported: "getCurrentTime",
		},
		{
			name:     "already lowercase",
			given:    "get_current_time",
			key:      "get_current_time",
			reported: "get_current_time",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.key, tc.given.Key())
			require.Equal(t, tc.reported, tc.given.String())
		})
	}
}

// TestToolName_KeyIsTheLowercasedReportedName pins the one relationship the two forms have to keep.
func TestToolName_KeyIsTheLowercasedReportedName(t *testing.T) {
	t.Parallel()

	names := []ToolName{
		"",
		"  padded  ",
		"getCurrentTime",
		"GET_CURRENT_TIME",
		"get_current_time",
	}

	for _, name := range names {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()

			require.Equal(t, strings.ToLower(name.String()), name.Key())
		})
	}
}
