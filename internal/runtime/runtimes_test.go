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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := AnyIntersection(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Join(tt.input, tt.sep)
			require.Equal(t, tt.expected, result)
		})
	}
}
