package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnyIntersection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []Runtime
		b        []Runtime
		expected bool
	}{
		{
			name:     "common element",
			a:        []Runtime{NPX, UVX},
			b:        []Runtime{Python, UVX},
			expected: true,
		},
		{
			name:     "no common elements",
			a:        []Runtime{NPX},
			b:        []Runtime{Docker},
			expected: false,
		},
		{
			name:     "empty a slice",
			a:        []Runtime{},
			b:        []Runtime{UVX},
			expected: false,
		},
		{
			name:     "empty b slice",
			a:        []Runtime{UVX},
			b:        []Runtime{},
			expected: false,
		},
		{
			name:     "both empty",
			a:        []Runtime{},
			b:        []Runtime{},
			expected: false,
		},
		{
			name:     "identical sets",
			a:        []Runtime{NPX, UVX},
			b:        []Runtime{NPX, UVX},
			expected: true,
		},
		{
			name:     "nil and empty",
			a:        nil,
			b:        []Runtime{},
			expected: false,
		},
		{
			name:     "empty and nil",
			a:        []Runtime{},
			b:        nil,
			expected: false,
		},
		{
			name:     "nil nil draw",
			a:        nil,
			b:        nil,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := AnyIntersection(tc.a, tc.b)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []Runtime
		sep      string
		expected string
	}{
		{
			name:     "multiple runtimes",
			input:    []Runtime{NPX, UVX, Docker},
			sep:      ",",
			expected: "npx,uvx,docker",
		},
		{
			name:     "single runtime",
			input:    []Runtime{Python},
			sep:      ",",
			expected: "python",
		},
		{
			name:     "empty slice",
			input:    []Runtime{},
			sep:      ",",
			expected: "",
		},
		{
			name:     "custom separator",
			input:    []Runtime{NPX, UVX},
			sep:      " | ",
			expected: "npx | uvx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := Join(tc.input, tc.sep)
			require.Equal(t, tc.expected, result)
		})
	}
}
