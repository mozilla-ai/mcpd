package daemon

import (
	"fmt"
	"time"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// Options contains optional configuration for the daemon.
// NewOptions should be used to create instances of Options.
type Options struct {
	// APIOptions contains functional options for the API server.
	APIOptions []APIOption

	// ClientInitTimeout specifies how long to wait for MCP server initialization.
	ClientInitTimeout time.Duration

	// ClientHealthCheckInterval specifies how often to ping MCP servers for health checks.
	ClientHealthCheckInterval time.Duration

	// ClientHealthCheckTimeout specifies maximum time to wait for health check responses.
	ClientHealthCheckTimeout time.Duration

	// ClientShutdownTimeout specifies how long to wait for MCP clients to close.
	ClientShutdownTimeout time.Duration

	// PluginConfig specifies the configuration for plugins.
	PluginConfig *config.PluginConfig
}

// Option defines a functional option for configuring Options.
// Options are applied in order, with later options overriding earlier ones.
type Option func(*Options) error

// NewOptions creates Options with optional configurations applied.
// Starts with default values, then applies options in order with later options overriding earlier ones.
func NewOptions(opts ...Option) (Options, error) {
	options := defaultOptions()

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&options); err != nil {
			return Options{}, err
		}
	}

	return options, nil
}

// WithAPIOptions configures API server options.
// Replaces all previous API configuration including CORS settings.
func WithAPIOptions(apiOpts ...APIOption) Option {
	return func(o *Options) error {
		o.APIOptions = apiOpts
		return nil
	}
}

// WithMCPServerInitTimeout configures how long to wait for MCP servers to initialize.
func WithMCPServerInitTimeout(timeout time.Duration) Option {
	return func(o *Options) error {
		if timeout <= 0 {
			return fmt.Errorf("init timeout must be positive, got %v", timeout)
		}
		o.ClientInitTimeout = timeout
		return nil
	}
}

// WithMCPServerHealthCheckInterval configures how often to ping MCP servers for health checks.
func WithMCPServerHealthCheckInterval(interval time.Duration) Option {
	return func(o *Options) error {
		if interval <= 0 {
			return fmt.Errorf("health check interval must be positive, got %v", interval)
		}
		o.ClientHealthCheckInterval = interval
		return nil
	}
}

// WithMCPServerHealthCheckTimeout configures maximum time to wait for MCP server health check responses.
func WithMCPServerHealthCheckTimeout(timeout time.Duration) Option {
	return func(o *Options) error {
		if timeout <= 0 {
			return fmt.Errorf("health check timeout must be positive, got %v", timeout)
		}
		o.ClientHealthCheckTimeout = timeout
		return nil
	}
}

// WithMCPServerShutdownTimeout configures how long to wait for MCP servers to shut down.
func WithMCPServerShutdownTimeout(timeout time.Duration) Option {
	return func(o *Options) error {
		if timeout <= 0 {
			return fmt.Errorf("server shutdown timeout must be positive, got %v", timeout)
		}
		o.ClientShutdownTimeout = timeout
		return nil
	}
}

// WithPluginConfig configures the plugin system with the specified configuration.
func WithPluginConfig(cfg *config.PluginConfig) Option {
	return func(o *Options) error {
		o.PluginConfig = cfg
		return nil
	}
}

// DefaultClientInitTimeout is the default time to wait for MCP server initialization.
func DefaultClientInitTimeout() time.Duration {
	return 30 * time.Second
}

// DefaultHealthCheckInterval is the default interval for health checks.
func DefaultHealthCheckInterval() time.Duration {
	return 10 * time.Second
}

// DefaultHealthCheckTimeout is the default timeout for health check responses.
func DefaultHealthCheckTimeout() time.Duration {
	return 3 * time.Second
}

// DefaultClientShutdownTimeout is the default time to wait for MCP clients to close.
func DefaultClientShutdownTimeout() time.Duration {
	return 5 * time.Second
}

// defaultOptions returns Options with default values.
func defaultOptions() Options {
	return Options{
		ClientInitTimeout:         DefaultClientInitTimeout(),
		ClientHealthCheckInterval: DefaultHealthCheckInterval(),
		ClientHealthCheckTimeout:  DefaultHealthCheckTimeout(),
		ClientShutdownTimeout:     DefaultClientShutdownTimeout(),
	}
}
