package cache

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/internal/files"
)

// Cache manages cached registry manifests.
// NewCache should be used to create instances of Cache.
type Cache struct {
	// dir is the directory where cache files are stored.
	dir string

	// ttl is the time-to-live for cached entries.
	ttl time.Duration

	// enabled determines if caching is enabled.
	enabled bool

	// refresh forces cache refresh when true.
	refresh bool

	// logger is used for logging cache operations.
	logger hclog.Logger
}

// NewCache creates a new cache instance for registry manifests.
func NewCache(logger hclog.Logger, opts ...Option) (*Cache, error) {
	options, err := NewOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Only create cache directory if caching is enabled.
	if options.enabled {
		if err := files.EnsureAtLeastRegularDir(options.dir); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &Cache{
		dir:     options.dir,
		logger:  logger.Named("cache"),
		enabled: options.enabled,
		refresh: options.refreshCache,
		ttl:     options.ttl,
	}, nil
}

// URL returns the appropriate URL (cached file:// or original HTTP).
func (c *Cache) URL(remoteURL string) (string, error) {
	if !c.enabled {
		c.logger.Debug("Cache disabled, using remote URL", "url", remoteURL)

		return remoteURL, nil
	}

	// Generate cache file path from URL hash.
	hash := sha256.Sum256([]byte(remoteURL))
	filename := fmt.Sprintf("%x.json", hash)
	cachePath := filepath.Join(c.dir, filename)

	// Check if refresh requested or cache expired.
	if c.refresh {
		c.logger.Debug("Cache refresh requested", "url", remoteURL)
		if err := c.downloadToCache(remoteURL, cachePath); err != nil {
			c.logger.Warn(
				"Failed to refresh cache, using remote URL",
				"url", remoteURL,
				"path", cachePath,
				"error", err,
			)

			return remoteURL, nil
		}
	} else if c.isExpired(cachePath) {
		c.logger.Debug("Cache expired or missing", "url", remoteURL, "path", cachePath)

		if err := c.downloadToCache(remoteURL, cachePath); err != nil {
			c.logger.Warn(
				"Failed to update cache, using remote URL",
				"url", remoteURL,
				"path", cachePath,
				"error", err,
			)

			return remoteURL, nil
		}
	}

	// Return file:// URL if cache exists.
	if _, err := os.Stat(cachePath); err == nil {
		fileURL := "file://" + cachePath
		c.logger.Debug("Using cached file", "url", fileURL, "remote", remoteURL)

		return fileURL, nil
	}

	// Fall back to original URL.
	c.logger.Debug("Cache file not found, using remote URL", "url", remoteURL, "path", cachePath)

	return remoteURL, nil
}

// downloadToCache downloads content from URL and saves to cache.
func (c *Cache) downloadToCache(url, cachePath string) error {
	c.logger.Debug("Downloading to cache", "url", url, "path", cachePath)

	// Download the content.
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch URL '%s': %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status from URL '%s': %d", url, resp.StatusCode)
	}

	// Create temporary file first.
	tmpFile, err := os.CreateTemp(c.dir, "tmp-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath) // Clean up on any error.
	}()

	// Copy content to temporary file.
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	_ = tmpFile.Close()

	// Atomically rename to final location.
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	c.logger.Debug("Successfully cached file", "url", url, "path", cachePath)
	return nil
}

// isExpired checks if a cache file is expired based on modification time.
func (c *Cache) isExpired(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true // Treat missing as expired.
	}
	return time.Since(info.ModTime()) > time.Duration(c.ttl)
}
