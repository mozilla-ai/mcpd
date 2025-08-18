package daemon

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// APIOptions contains optional configuration for the API server.
// NewAPIOptions should be used to create instances of APIOptions.
type APIOptions struct {
	// CORS configuration for cross-origin requests.
	CORS CORSConfig

	// ShutdownTimeout specifies how long to wait for graceful shutdown.
	ShutdownTimeout time.Duration
}

// CORSConfig defines Cross-Origin Resource Sharing settings for the API server.
type CORSConfig struct {
	// Enabled determines whether CORS headers are added to responses.
	Enabled bool

	// AllowCredentials indicates whether the request can include credentials.
	// Must be false when AllowOrigins contains "*"
	AllowCredentials bool

	// AllowedHeaders specifies which headers the client can include in requests.
	AllowedHeaders []string

	// AllowMethods specifies which HTTP methods are permitted.
	// Using strings to match the go-chi/cors library API.
	AllowMethods []string

	// AllowOrigins specifies which origins can access the API.
	// Use ["*"] to allow all origins (not recommended for production).
	AllowOrigins []string

	// ExposedHeaders specifies which response headers are accessible to the client.
	ExposedHeaders []string

	// MaxAge specifies how long browsers can cache preflight responses.
	MaxAge time.Duration
}

// APIOption defines a functional option for configuring APIOptions.
// Options are applied in order, with later options overriding earlier ones.
type APIOption func(*APIOptions) error

// NewAPIOptions creates APIOptions with optional configurations applied.
// Starts with default values, then applies options in order with later options overriding earlier ones.
func NewAPIOptions(opts ...APIOption) (APIOptions, error) {
	options := APIOptions{
		CORS: CORSConfig{
			// Explicitly setting some values for clarity.
			Enabled:          false,
			AllowOrigins:     nil,
			AllowMethods:     DefaultCORSAllowMethods(),
			AllowedHeaders:   DefaultCORSAllowHeaders(),
			AllowCredentials: DefaultCORSAllowCredentials(),
			ExposedHeaders:   nil,
			MaxAge:           DefaultCORSMaxAge(),
		},
		ShutdownTimeout: DefaultAPIShutdownTimeout(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&options); err != nil {
			return APIOptions{}, err
		}
	}

	return options, nil
}

// WithCORSEnabled enables or disables CORS support.
func WithCORSEnabled(enabled bool) APIOption {
	return func(o *APIOptions) error {
		o.CORS.Enabled = enabled
		return nil
	}
}

// WithCORSAllowHeaders sets which additional request headers are safe for the client to send.
// See: https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_request_header for information on
// safelisted-headers that don't required explicit configuration.
func WithCORSAllowHeaders(headers []string) APIOption {
	return func(o *APIOptions) error {
		o.CORS.AllowedHeaders = headers
		return nil
	}
}

// WithCORSAllowOrigins sets the allowed origins for CORS requests.
func WithCORSAllowOrigins(origins []string) APIOption {
	return func(o *APIOptions) error {
		o.CORS.AllowOrigins = origins
		return nil
	}
}

// WithCORSAllowMethods sets the allowed HTTP methods for CORS requests.
func WithCORSAllowMethods(methods []string) APIOption {
	return func(o *APIOptions) error {
		o.CORS.AllowMethods = methods
		return nil
	}
}

// WithCORSAllowCredentials sets whether credentials are allowed in CORS requests.
func WithCORSAllowCredentials(allowed bool) APIOption {
	return func(o *APIOptions) error {
		o.CORS.AllowCredentials = allowed
		return nil
	}
}

// WithCORSExposeHeaders sets which additional response headers are safe for the client to read.
// See: https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_response_header for information on
// safelisted-headers that don't required explicit configuration.
func WithCORSExposeHeaders(headers []string) APIOption {
	return func(o *APIOptions) error {
		o.CORS.ExposedHeaders = headers
		return nil
	}
}

// WithCORSMaxAge sets how long browsers can cache CORS preflight responses.
func WithCORSMaxAge(maxAge time.Duration) APIOption {
	return func(o *APIOptions) error {
		o.CORS.MaxAge = maxAge
		return nil
	}
}

// WithShutdownTimeout configures how long to wait for graceful shutdown.
func WithShutdownTimeout(timeout time.Duration) APIOption {
	return func(o *APIOptions) error {
		if timeout <= 0 {
			return fmt.Errorf("shutdown timeout must be positive, got %v", timeout)
		}
		o.ShutdownTimeout = timeout
		return nil
	}
}

// DefaultCORSAllowHeaders returns standard headers required for API interaction.
func DefaultCORSAllowHeaders() []string {
	// Headers that are safe-listed regardless of configuration.
	return []string{
		"Accept",
		"Accept-Language",
		"Content-Language",
		"Content-Type",
		"Range",
	}
}

// DefaultCORSAllowMethods returns standard HTTP methods for CORS.
func DefaultCORSAllowMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
	}
}

// DefaultCORSAllowCredentials returns the default CORS 'allow credentials' setting.
func DefaultCORSAllowCredentials() bool {
	return false
}

// DefaultCORSMaxAge returns the default CORS max age duration.
// Max age is the default time browsers can cache preflight responses.
func DefaultCORSMaxAge() time.Duration {
	return 5 * time.Minute
}

// DefaultAPIShutdownTimeout is the default time allowed for API server graceful shutdown.
func DefaultAPIShutdownTimeout() time.Duration {
	return 5 * time.Second
}

// validateAddr checks if the address is a valid "host:port" string.
func validateAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	if port == "" {
		return fmt.Errorf("address missing port")
	}

	if _, err := strconv.Atoi(port); err != nil {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return fmt.Errorf("invalid address port: %s", port)
		}
	}

	_ = host
	return nil
}
