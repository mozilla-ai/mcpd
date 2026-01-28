package cache

import (
	"fmt"
	"strings"
	"time"

	"github.com/mozilla-ai/mcpd/internal/registry/options"
)

// Option defines a functional option for configuring Cache.
type Option func(*Options) error

// Options contains optional configuration for the cache.
type Options struct {
	// dir is the directory where cache files are stored.
	dir string

	// ttl is the time-to-live for cached entries.
	ttl time.Duration

	// enabled determines if caching is enabled.
	enabled bool

	// refreshCache forces cache refresh when true.
	refreshCache bool
}

func NewOptions(opts ...Option) (Options, error) {
	dir, err := options.DefaultCacheDir()
	if err != nil {
		return Options{}, err
	}

	// Default options.
	o := Options{
		dir:          dir,
		ttl:          time.Duration(*options.DefaultCacheTTL()),
		enabled:      true,
		refreshCache: false,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&o); err != nil {
			return Options{}, err
		}
	}

	return o, nil
}

// WithDirectory sets the cache directory.
func WithDirectory(dir string) Option {
	return func(o *Options) error {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return fmt.Errorf("cache directory cannot be empty")
		}
		o.dir = dir
		return nil
	}
}

// WithTTL sets the cache entry time-to-live.
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) error {
		if ttl <= 0 {
			return fmt.Errorf("TTL must be positive, got %v", ttl)
		}
		o.ttl = ttl
		return nil
	}
}

// WithCaching configures whether caching is enabled.
func WithCaching(enabled bool) Option {
	return func(o *Options) error {
		o.enabled = enabled
		return nil
	}
}

// WithRefreshCache forces cache refresh.
func WithRefreshCache(refreshCache bool) Option {
	return func(o *Options) error {
		o.refreshCache = refreshCache
		return nil
	}
}
