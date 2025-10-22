package options

import (
	"path/filepath"
	"time"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/files"
)

// BuildOption defines a functional option for configuring registry builds.
type BuildOption func(*BuildOptions) error

// BuildOptions contains configuration for building a registry.
// NewBuildOptions should be used to create instances of BuildOptions.
type BuildOptions struct {
	// UseCache determines if registry manifest caching should be used.
	UseCache bool

	// RefreshCache forces refresh of cached registry manifests.
	RefreshCache bool

	// CacheDir specifies the cache directory (empty uses default).
	CacheDir string

	// CacheTTL specifies the cache time-to-live (zero uses default).
	CacheTTL time.Duration
}

// NewBuildOptions creates BuildOptions with optional configurations applied.
// Starts with default values, then applies options in order with later options overriding earlier ones.
func NewBuildOptions(opts ...BuildOption) (BuildOptions, error) {
	options := BuildOptions{
		UseCache:     true,
		RefreshCache: false,
		CacheDir:     "", // Empty uses default
		CacheTTL:     0,  // Zero uses default
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&options); err != nil {
			return BuildOptions{}, err
		}
	}

	return options, nil
}

// WithCaching configures whether caching is enabled.
func WithCaching(enabled bool) BuildOption {
	return func(o *BuildOptions) error {
		o.UseCache = enabled
		return nil
	}
}

// WithRefreshCache configures whether to force cache refresh.
func WithRefreshCache(refreshCache bool) BuildOption {
	return func(o *BuildOptions) error {
		o.RefreshCache = refreshCache
		return nil
	}
}

// WithCacheDir configures the cache directory.
func WithCacheDir(dir string) BuildOption {
	return func(o *BuildOptions) error {
		o.CacheDir = dir
		return nil
	}
}

// WithCacheTTL configures the cache time-to-live.
func WithCacheTTL(ttl time.Duration) BuildOption {
	return func(o *BuildOptions) error {
		o.CacheTTL = ttl
		return nil
	}
}

// DefaultCacheDir returns the default cache directory for registry manifests.
// It is the user-specific cache directory with "registries" appended.
// Returns the path or an error if the user-specific cache directory cannot be determined.
func DefaultCacheDir() (string, error) {
	cacheDir, err := files.UserSpecificCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "registries"), nil
}

// DefaultCacheTTL returns the default cache time-to-live.
func DefaultCacheTTL() *config.Duration {
	d := config.Duration(24 * time.Hour)
	return &d
}