package packages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTools_Names(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Tools
		expected []string
	}{
		{
			name:     "empty list",
			input:    Tools{},
			expected: []string{},
		},
		{
			name: "single tool",
			input: Tools{
				{Name: "tool-1"},
			},
			expected: []string{"tool-1"},
		},
		{
			name: "multiple tools",
			input: Tools{
				{Name: "alpha"},
				{Name: "beta"},
				{Name: "gamma"},
			},
			expected: []string{"alpha", "beta", "gamma"},
		},
		{
			name: "tools with empty names",
			input: Tools{
				{Name: ""},
				{Name: "    "},
				{Name: "non-empty"},
			},
			expected: []string{"", "", "non-empty"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.input.Names()
			require.Equal(t, tc.expected, result)
		})
	}
}
