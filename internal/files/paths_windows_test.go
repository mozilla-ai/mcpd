//go:build windows

package files

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathToFileURL_Windows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			// Note explicit platform difference - Linux escapes the '\' but Windows does not.
			name:     "native Windows path with backslashes",
			path:     `C:\Users\test\registry.json`,
			expected: "file:///C:/Users/test/registry.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, PathToFileURL(tc.path))
		})
	}
}

func TestPathToFileURL_FileURLToPath_RoundTrip_Windows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			// Round trip normalises backslashes to forward slashes via filepath.ToSlash.
			name:     "native Windows path with backslashes",
			path:     `C:\Users\test\registry.json`,
			expected: "C:/Users/test/registry.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, err := url.Parse(PathToFileURL(tc.path))
			require.NoError(t, err)
			require.Equal(t, tc.expected, FileURLToPath(u))
		})
	}
}
