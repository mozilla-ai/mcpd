package files

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathToFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "POSIX-style absolute path",
			path:     "/tmp/registry.json",
			expected: "file:///tmp/registry.json",
		},
		{
			name:     "Windows-style absolute path with drive letter",
			path:     "C:/Users/test/registry.json",
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

func TestFileURLToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileURL  string
		expected string
	}{
		{
			name:     "POSIX-style absolute path",
			fileURL:  "file:///tmp/registry.json",
			expected: "/tmp/registry.json",
		},
		{
			name:     "Windows-style absolute path with drive letter",
			fileURL:  "file:///C:/Users/test/registry.json",
			expected: "C:/Users/test/registry.json",
		},
		{
			name:     "Windows file URL with drive letter as host (file://C:/...)",
			fileURL:  "file://C:/Users/test/registry.json",
			expected: "C:/Users/test/registry.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, err := url.Parse(tc.fileURL)
			require.NoError(t, err)
			require.Equal(t, tc.expected, FileURLToPath(u))
		})
	}
}

func TestPathToFileURL_FileURLToPath_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "POSIX-style absolute path",
			path: "/tmp/registry.json",
		},
		{
			name: "Windows-style absolute path with drive letter",
			path: "C:/Users/test/registry.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, err := url.Parse(PathToFileURL(tc.path))
			require.NoError(t, err)
			require.Equal(t, tc.path, FileURLToPath(u))
		})
	}
}
