package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestCache_New(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logger := hclog.NewNullLogger()

	tc := []struct {
		name         string
		dir          string
		ttl          time.Duration
		enabled      bool
		refreshCache bool
		expectNil    bool
	}{
		{
			name:      "creates cache successfully with caching enabled",
			dir:       filepath.Join(tempDir, "test-cache"),
			ttl:       time.Hour,
			enabled:   true,
			expectNil: false,
		},
		{
			name:      "creates cache instance even when caching is disabled",
			dir:       filepath.Join(tempDir, "test-cache-2"),
			ttl:       time.Hour,
			enabled:   false,
			expectNil: false,
		},
		{
			name:         "creates cache when refresh is true even with caching disabled",
			dir:          filepath.Join(tempDir, "test-cache-3"),
			ttl:          time.Hour,
			enabled:      false,
			refreshCache: true,
			expectNil:    false,
		},
	}

	for _, testCase := range tc {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{
				WithDirectory(testCase.dir),
				WithTTL(testCase.ttl),
				WithCaching(testCase.enabled),
				WithRefreshCache(testCase.refreshCache),
			}

			cache, err := NewCache(logger, opts...)
			require.NoError(t, err)

			if testCase.expectNil {
				require.Nil(t, cache)
			} else {
				require.NotNil(t, cache)
				require.Equal(t, testCase.dir, cache.dir)
				require.Equal(t, testCase.ttl, cache.ttl)
				require.Equal(t, testCase.enabled, cache.enabled)
				require.Equal(t, testCase.refreshCache, cache.refresh)
			}
		})
	}
}

func TestCache_URL_CachingDisabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logger := hclog.NewNullLogger()

	// When caching is disabled, cache instance is still created.
	cache, err := NewCache(logger, WithDirectory(tempDir), WithTTL(time.Hour), WithCaching(false))
	require.NoError(t, err)
	require.NotNil(t, cache)

	// With caching disabled, should always return original URL.
	originalURL := "https://example.com/test.json"
	url, err := cache.URL(originalURL)
	require.NoError(t, err)
	require.Equal(t, originalURL, url)
}

func TestCache_NoCacheDirectoryCreatedWhenDisabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cacheSubDir := filepath.Join(tempDir, "cache-subdir")
	logger := hclog.NewNullLogger()

	// Verify the cache directory doesn't exist initially.
	_, err := os.Stat(cacheSubDir)
	require.True(t, os.IsNotExist(err), "Cache directory should not exist initially")

	// Create cache with caching disabled.
	cache, err := NewCache(logger, WithDirectory(cacheSubDir), WithTTL(time.Hour), WithCaching(false))
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Verify the cache directory still doesn't exist after NewCache.
	_, err = os.Stat(cacheSubDir)
	require.True(t, os.IsNotExist(err), "Cache directory should not be created when caching is disabled")

	// Call URL method (which should not create directory since caching is disabled).
	originalURL := "https://example.com/test.json"
	url, err := cache.URL(originalURL)
	require.NoError(t, err)
	require.Equal(t, originalURL, url)

	// Verify the cache directory still doesn't exist after URL call.
	_, err = os.Stat(cacheSubDir)
	require.True(
		t,
		os.IsNotExist(err),
		"Cache directory should not be created even after URL call when caching is disabled",
	)
}

func TestCache_URL_WithValidCache(t *testing.T) {
	t.Parallel()

	// Create a test HTTP server.
	testData := `{"test": "data"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	logger := hclog.NewNullLogger()

	cache, err := NewCache(logger, WithDirectory(tempDir), WithTTL(time.Hour))
	require.NoError(t, err)

	// First call should cache the data.
	url1, err := cache.URL(server.URL)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(url1, "file://"))

	// Second call should return cached URL.
	url2, err := cache.URL(server.URL)
	require.NoError(t, err)
	require.Equal(t, url1, url2)

	// Verify cached file exists and contains expected data.
	filePath := strings.TrimPrefix(url1, "file://")
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, testData, string(content))
}

func TestCache_URL_ExpiredCache(t *testing.T) {
	t.Parallel()

	// Create a test HTTP server that returns different data each time.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"call": %d}`, callCount)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	logger := hclog.NewNullLogger()

	// Use very short TTL for testing.
	cache, err := NewCache(logger, WithDirectory(tempDir), WithTTL(10*time.Millisecond))
	require.NoError(t, err)

	// First call should cache the data.
	url1, err := cache.URL(server.URL)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(url1, "file://"))

	// Read first cached content.
	filePath := strings.TrimPrefix(url1, "file://")
	content1, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, `{"call": 1}`, string(content1))

	// Wait for cache to expire.
	time.Sleep(20 * time.Millisecond)

	// Second call should refresh cache.
	url2, err := cache.URL(server.URL)
	require.NoError(t, err)
	require.Equal(t, url1, url2) // Same file path.

	// Read updated cached content.
	content2, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, `{"call": 2}`, string(content2))
}

func TestCache_URL_RefreshCache(t *testing.T) {
	t.Parallel()

	// Create a test HTTP server that returns different data each time.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"call": %d}`, callCount)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	logger := hclog.NewNullLogger()

	// First, create cache with normal settings.
	cache1, err := NewCache(logger, WithDirectory(tempDir), WithTTL(time.Hour))
	require.NoError(t, err)

	url1, err := cache1.URL(server.URL)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(url1, "file://"))

	// Read first cached content.
	filePath := strings.TrimPrefix(url1, "file://")
	content1, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, `{"call": 1}`, string(content1))

	// Create cache with refresh flag.
	cache2, err := NewCache(logger, WithDirectory(tempDir), WithTTL(time.Hour), WithRefreshCache(true))
	require.NoError(t, err)

	url2, err := cache2.URL(server.URL)
	require.NoError(t, err)
	require.Equal(t, url1, url2) // Same file path.

	// Read updated cached content.
	content2, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, `{"call": 2}`, string(content2))
}
