package runtime

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

func TestLoadFromURL_HTTP(t *testing.T) {
	t.Parallel()

	testData := testStruct{
		Message: "hello",
		Count:   42,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"message": "hello", "count": 42}`)
	}))
	defer server.Close()

	result, err := LoadFromURL[testStruct](server.URL, "test-registry")
	require.NoError(t, err)
	require.Equal(t, testData, result)
}

func TestLoadFromURL_File(t *testing.T) {
	t.Parallel()

	testData := testStruct{
		Message: "hello from file",
		Count:   123,
	}

	// Create a temporary file with test data.
	tempFile := filepath.Join(t.TempDir(), "test.json")
	content := `{"message": "hello from file", "count": 123}`
	err := os.WriteFile(tempFile, []byte(content), 0o644)
	require.NoError(t, err)

	// Test with file:// URL.
	fileURL := "file://" + tempFile
	result, err := LoadFromURL[testStruct](fileURL, "test-registry")
	require.NoError(t, err)
	require.Equal(t, testData, result)
}

func TestLoadFromURL_InvalidURL(t *testing.T) {
	t.Parallel()

	_, err := LoadFromURL[testStruct]("://invalid-url", "test-registry")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid URL")
}

func TestLoadFromURL_UnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := LoadFromURL[testStruct]("ftp://example.com/test.json", "test-registry")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported URL scheme")
}

func TestLoadFromURL_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadFromURL[testStruct]("file:///nonexistent/file.json", "test-registry")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read file")
}

func TestLoadFromURL_InvalidJSON(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), "invalid.json")
	content := `{"invalid": json}`
	err := os.WriteFile(tempFile, []byte(content), 0o644)
	require.NoError(t, err)

	fileURL := "file://" + tempFile
	_, err = LoadFromURL[testStruct](fileURL, "test-registry")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal")
}

func TestLoadFromURL_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "Not found")
	}))
	defer server.Close()

	_, err := LoadFromURL[testStruct](server.URL, "test-registry")
	require.Error(t, err)
	require.Contains(t, err.Error(), "received non-OK HTTP status")
}
