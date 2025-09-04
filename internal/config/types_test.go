package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerEntry_Equal(t *testing.T) {
	t.Parallel()

	baseEntry := func() *ServerEntry {
		return &ServerEntry{
			Name:                   "test-server",
			Package:                "uvx::test-server@1.0.0",
			Tools:                  []string{"tool1", "tool2"},
			RequiredEnvVars:        []string{"API_KEY", "SECRET"},
			RequiredPositionalArgs: []string{"pos1", "pos2"},
			RequiredValueArgs:      []string{"--arg1", "--arg2"},
			RequiredBoolArgs:       []string{"--flag1", "--flag2"},
		}
	}

	bse := baseEntry()

	testCases := []struct {
		name     string
		entry1   *ServerEntry
		entry2   *ServerEntry
		expected bool
	}{
		{
			name:     "identical entries",
			entry1:   bse,
			entry2:   bse,
			expected: true,
		},
		{
			name:     "identical content different instances",
			entry1:   baseEntry(),
			entry2:   baseEntry(),
			expected: true,
		},
		{
			name:     "nil comparison",
			entry1:   baseEntry(),
			entry2:   nil,
			expected: false,
		},
		{
			name:   "different names",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.Name = "different-server"
				return srv
			}(),
			expected: false,
		},
		{
			name:   "different packages",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.Package = "uvx::different-server@1.0.0"
				return srv
			}(),
			expected: false,
		},
		{
			name:   "tools order independent - same content",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.Tools = []string{"tool2", "tool1"} // Different order
				return srv
			}(),
			expected: true,
		},
		{
			name:   "env vars order independent - same content",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.RequiredEnvVars = []string{"SECRET", "API_KEY"} // Different order
				return srv
			}(),
			expected: true,
		},
		{
			name:   "positional args order matters",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.RequiredPositionalArgs = []string{"pos2", "pos1"} // Different order
				return srv
			}(),
			expected: false,
		},
		{
			name:   "value args order independent - same content",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.RequiredValueArgs = []string{"--arg2", "--arg1"} // Different order
				return srv
			}(),
			expected: true,
		},
		{
			name:   "bool args order independent - same content",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.RequiredBoolArgs = []string{"--flag2", "--flag1"} // Different order
				return srv
			}(),
			expected: true,
		},
		{
			name:   "different tools content",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.Tools = []string{"tool1", "tool3"} // Different content
				return srv
			}(),
			expected: false,
		},
		{
			name:   "empty slices vs nil slices are equal",
			entry1: baseEntry(),
			entry2: func() *ServerEntry {
				srv := baseEntry()
				srv.Tools = nil
				return srv
			}(),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.entry1.Equals(tc.entry2)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualStringSlicesUnordered(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "identical slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "same elements different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "a", "b"},
			expected: true,
		},
		{
			name:     "different elements",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "empty vs nil",
			a:        []string{},
			b:        nil,
			expected: true,
		},
		{
			name:     "duplicate elements same count",
			a:        []string{"a", "b", "a"},
			b:        []string{"b", "a", "a"},
			expected: true,
		},
		{
			name:     "duplicate elements different count",
			a:        []string{"a", "b", "a"},
			b:        []string{"a", "b", "b"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := equalStringSlicesUnordered(tc.a, tc.b)
			require.Equal(t, tc.expected, result)
		})
	}
}
