package config

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mozilla-ai/mcpd/internal/context"
)

// APIConfigSection contains API server configuration settings.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type APIConfigSection struct {
	// Address to bind the API server (e.g., "0.0.0.0:8090")
	// Maps to CLI flag --addr
	Addr *string `json:"addr,omitempty" toml:"addr,omitempty" yaml:"addr,omitempty"`

	// Nested timeout configuration for API operations
	Timeout *APITimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Nested CORS configuration for cross-origin requests
	CORS *CORSConfigSection `json:"cors,omitempty" toml:"cors,omitempty" yaml:"cors,omitempty"`
}

// APITimeoutConfigSection contains timeout settings for API operations.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type APITimeoutConfigSection struct {
	// Shutdown timeout for graceful API server shutdown
	// Maps to CLI flag --timeout-api-shutdown
	Shutdown *Duration `json:"shutdown,omitempty" toml:"shutdown,omitempty" yaml:"shutdown,omitempty"`
}

// CORSConfigSection contains Cross-Origin Resource Sharing (CORS) configuration.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type CORSConfigSection struct {
	// Enable CORS support
	// Maps to CLI flag --cors-enable
	Enable *bool `json:"enable,omitempty" toml:"enable,omitempty" yaml:"enable,omitempty"`

	// Allowed origins for CORS requests
	// Maps to CLI flag --cors-origins
	Origins []string `json:"allowOrigins,omitempty" toml:"allow_origins,omitempty" yaml:"allow_origins,omitempty"`

	// Allowed HTTP methods for CORS requests
	// Maps to CLI flag --cors-methods
	Methods []string `json:"allowMethods,omitempty" toml:"allow_methods,omitempty" yaml:"allow_methods,omitempty"`

	// Allowed headers for CORS requests
	// Maps to CLI flag --cors-headers
	Headers []string `json:"allowHeaders,omitempty" toml:"allow_headers,omitempty" yaml:"allow_headers,omitempty"`

	// Headers exposed to the client
	// Maps to CLI flag --cors-expose-headers
	ExposeHeaders []string `json:"exposeHeaders,omitempty" toml:"expose_headers,omitempty" yaml:"expose_headers,omitempty"`

	// Allow credentials in CORS requests
	// Maps to CLI flag --cors-credentials
	Credentials *bool `json:"allowCredentials,omitempty" toml:"allow_credentials,omitempty" yaml:"allow_credentials,omitempty"`

	// Maximum age for CORS preflight cache
	// Maps to CLI flag --cors-max-age
	MaxAge *Duration `json:"maxAge,omitempty" toml:"max_age,omitempty" yaml:"max_age,omitempty"`
}

// DaemonConfig represents daemon-specific configuration that can be stored in .mcpd.toml.
// This extends the existing Config struct with daemon settings.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type DaemonConfig struct {
	// API configuration (includes address and nested timeout/cors)
	API *APIConfigSection `json:"api,omitempty" toml:"api,omitempty" yaml:"api,omitempty"`

	// MCP configuration (includes nested timeout and interval settings)
	MCP *MCPConfigSection `json:"mcp,omitempty" toml:"mcp,omitempty" yaml:"mcp,omitempty"`
}

// Duration is a custom time.Duration type that provides improved marshaling.
type Duration time.Duration

// MCPConfigSection contains MCP (Model Context Protocol) server configuration settings.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type MCPConfigSection struct {
	// Nested timeout configuration for MCP operations
	Timeout *MCPTimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Nested interval configuration for MCP periodic operations
	Interval *MCPIntervalConfigSection `json:"interval,omitempty" toml:"interval,omitempty" yaml:"interval,omitempty"`
}

// MCPIntervalConfigSection contains interval settings for periodic MCP operations.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type MCPIntervalConfigSection struct {
	// Health check interval for MCP servers
	// Maps to CLI flag --interval-mcp-health
	Health *Duration `json:"health,omitempty" toml:"health,omitempty" yaml:"health,omitempty"`
}

// MCPTimeoutConfigSection contains timeout settings for MCP operations.
//
// NOTE: if you add/remove fields you must review the associated Getter, Setter and Validator implementations,
// along with /docs/daemon-configuration.md.
type MCPTimeoutConfigSection struct {
	// Shutdown timeout for graceful MCP server shutdown
	// Maps to CLI flag --timeout-mcp-shutdown
	Shutdown *Duration `json:"shutdown,omitempty" toml:"shutdown,omitempty" yaml:"shutdown,omitempty"`

	// Initialization timeout for MCP server startup
	// Maps to CLI flag --timeout-mcp-init
	Init *Duration `json:"init,omitempty" toml:"init,omitempty" yaml:"init,omitempty"`

	// Health check timeout for MCP servers
	// Maps to CLI flag --timeout-mcp-health
	Health *Duration `json:"health,omitempty" toml:"health,omitempty" yaml:"health,omitempty"`
}

// AvailableKeys implements SchemaProvider for APIConfigSection.
func (a *APIConfigSection) AvailableKeys() []SchemaKey {
	keys := []SchemaKey{
		{Path: "addr", Type: "string", Description: "API server address (host:port)"},
	}

	// Always return timeout keys regardless of whether timeout section exists
	timeoutSection := &APITimeoutConfigSection{}
	for _, key := range timeoutSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "timeout." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	// Always return CORS keys regardless of whether CORS section exists
	corsSection := &CORSConfigSection{}
	for _, key := range corsSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "cors." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	return keys
}

// Get implements Getter for APIConfigSection.
// Returns all API configuration when called with no keys, or specific values when keys are provided.
func (a *APIConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return a.getAll()
	}

	key := normalizeKey(keys[0])

	if len(keys) == 1 {
		switch key {
		case "addr":
			if a.Addr == nil {
				return nil, fmt.Errorf("api.addr not set")
			}
			return *a.Addr, nil
		case "timeout":
			if a.Timeout == nil {
				return nil, fmt.Errorf("api.timeout not set")
			}
			return a.Timeout.Get()
		case "cors":
			if a.CORS == nil {
				return nil, fmt.Errorf("api.cors not set")
			}
			return a.CORS.Get()
		default:
			return nil, fmt.Errorf("unknown API config key: %s", key)
		}
	}

	// Handle multi-key access to subsections
	switch key {
	case "timeout":
		if a.Timeout == nil {
			return nil, fmt.Errorf("api.timeout not set")
		}
		return a.Timeout.Get(keys[1:]...)
	case "cors":
		if a.CORS == nil {
			return nil, fmt.Errorf("api.cors not set")
		}
		return a.CORS.Get(keys[1:]...)
	default:
		return nil, fmt.Errorf("unknown API subsection: %s", key)
	}
}

// Set implements Setter for APIConfigSection.
// Handles API configuration at the top level and routes to subsections.
func (a *APIConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("API config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	key := normalizeKey(parts[0])

	// Handle direct API config keys
	if len(parts) == 1 {
		switch key {
		case "addr":
			oldValue := a.Addr
			if value == "" {
				a.Addr = nil
			} else {
				a.Addr = &value
			}
			return determineStringPtrResult(oldValue, a.Addr), nil
		default:
			return context.Noop, fmt.Errorf("unknown API config key: %s", key)
		}
	}

	// Handle subsection routing
	switch key {
	case "timeout":
		if a.Timeout == nil {
			a.Timeout = &APITimeoutConfigSection{}
		}
		return a.Timeout.Set(strings.Join(parts[1:], "."), value)
	case "cors":
		if a.CORS == nil {
			a.CORS = &CORSConfigSection{}
		}
		return a.CORS.Set(strings.Join(parts[1:], "."), value)
	default:
		return context.Noop, fmt.Errorf("unknown API subsection: %s", key)
	}
}

// Validate implements Validator for APIConfigSection.
// Validates API configuration values.
func (a *APIConfigSection) Validate() error {
	if a == nil {
		return nil
	}

	var validationErrors []error

	// Validate address.
	if a.Addr != nil {
		if *a.Addr == "" {
			validationErrors = append(validationErrors, fmt.Errorf("API address cannot be empty"))
		} else if !isValidAddr(*a.Addr) {
			validationErrors = append(validationErrors, fmt.Errorf("API address \"%s\" appears to be invalid (expected format: host:port)", *a.Addr))
		}
	}

	// Validate subsections.
	if a.Timeout != nil {
		if err := a.Timeout.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("timeout configuration error: %w", err))
		}
	}

	if a.CORS != nil {
		if err := a.CORS.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("CORS configuration error: %w", err))
		}
	}

	return errors.Join(validationErrors...)
}

// AvailableKeys implements SchemaProvider for APITimeoutConfigSection.
func (a *APITimeoutConfigSection) AvailableKeys() []SchemaKey {
	return []SchemaKey{
		{Path: "shutdown", Type: "duration", Description: "API server shutdown timeout"},
	}
}

// Get implements Getter for APITimeoutConfigSection.
// Returns all timeout configuration when called with no keys, or specific values when keys are provided.
func (a *APITimeoutConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return a.getAll()
	}

	if len(keys) > 1 {
		// Check if the first key is a valid leaf key
		key := normalizeKey(keys[0])
		if key == "shutdown" {
			return nil, fmt.Errorf("shutdown is not a subsection")
		}
		return nil, fmt.Errorf("API timeout config does not support nested keys")
	}

	key := normalizeKey(keys[0])

	switch key {
	case "shutdown":
		if a.Shutdown == nil {
			return nil, fmt.Errorf("api.timeout.shutdown not set")
		}
		return *a.Shutdown, nil
	default:
		return nil, fmt.Errorf("API timeout %w: %s", ErrInvalidKey, key)
	}
}

// Set implements Setter for APITimeoutConfigSection.
// Handles API timeout configuration at the leaf level.
func (a *APITimeoutConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("path cannot be empty")
	}

	key := normalizeKey(path)

	switch key {
	case "shutdown":
		oldValue := a.Shutdown
		if value == "" {
			a.Shutdown = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("%w: %w", NewErrInvalidValue("shutdown", value), err)
			}
			a.Shutdown = &duration
		}
		return determineDurationPtrResult(oldValue, a.Shutdown), nil
	default:
		return context.Noop, fmt.Errorf("unknown API timeout config key: %s", key)
	}
}

// Validate implements Validator for APITimeoutConfigSection.
// Validates API timeout configuration values.
func (a *APITimeoutConfigSection) Validate() error {
	if a.Shutdown != nil {
		if *a.Shutdown <= 0 {
			return fmt.Errorf("API shutdown timeout must be positive")
		}
	}
	return nil
}

// AvailableKeys implements SchemaProvider for CORSConfigSection.
func (c *CORSConfigSection) AvailableKeys() []SchemaKey {
	return []SchemaKey{
		{Path: "enable", Type: "bool", Description: "Enable CORS support"},
		{Path: "allow_origins", Type: "[]string", Description: "Allowed origins for CORS requests"},
		{Path: "allow_methods", Type: "[]string", Description: "Allowed HTTP methods for CORS requests"},
		{Path: "allow_headers", Type: "[]string", Description: "Allowed headers for CORS requests"},
		{Path: "expose_headers", Type: "[]string", Description: "Headers exposed to the client"},
		{Path: "allow_credentials", Type: "bool", Description: "Allow credentials in CORS requests"},
		{Path: "max_age", Type: "duration", Description: "Maximum age for CORS preflight cache"},
	}
}

// EnableOrDefault returns the CORS enable setting, falling back to defaultEnable if not set.
func (c *CORSConfigSection) EnableOrDefault(defaultEnable bool) bool {
	if c == nil || c.Enable == nil {
		return defaultEnable
	}
	return *c.Enable
}

// Get implements Getter for CORSConfigSection.
// Returns all CORS configuration when called with no keys, or specific values when keys are provided.
func (c *CORSConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return c.getAll()
	}

	if err := ensureSingleKey(keys, "CORS"); err != nil {
		return nil, err
	}

	key := normalizeKey(keys[0])

	switch key {
	case "enable":
		if c.Enable == nil {
			return nil, fmt.Errorf("cors.enable not set")
		}
		return *c.Enable, nil
	case "allow_origins":
		if len(c.Origins) == 0 {
			return nil, fmt.Errorf("cors.allow_origins not set")
		}
		return c.Origins, nil
	case "allow_methods":
		if len(c.Methods) == 0 {
			return nil, fmt.Errorf("cors.allow_methods not set")
		}
		return c.Methods, nil
	case "allow_headers":
		if len(c.Headers) == 0 {
			return nil, fmt.Errorf("cors.allow_headers not set")
		}
		return c.Headers, nil
	case "expose_headers":
		if len(c.ExposeHeaders) == 0 {
			return nil, fmt.Errorf("cors.expose_headers not set")
		}
		return c.ExposeHeaders, nil
	case "allow_credentials":
		if c.Credentials == nil {
			return nil, fmt.Errorf("cors.allow_credentials not set")
		}
		return *c.Credentials, nil
	case "max_age":
		if c.MaxAge == nil {
			return nil, fmt.Errorf("cors.max_age not set")
		}
		return *c.MaxAge, nil
	default:
		return nil, fmt.Errorf("unknown CORS config key: %s", key)
	}
}

// Set implements Setter for CORSConfigSection.
// Handles CORS configuration at the leaf level.
func (c *CORSConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("CORS config path cannot be empty")
	}

	key := normalizeKey(path)

	switch key {
	case "enable":
		oldValue := c.Enable
		if value == "" {
			c.Enable = nil
		} else {
			boolValue, err := parseBool(value)
			if err != nil {
				return context.Noop, NewErrInvalidValue("enable", value)
			}
			c.Enable = &boolValue
		}
		return determineBoolPtrResult(oldValue, c.Enable), nil

	case "allow_origins":
		oldValue := c.Origins
		if value == "" {
			c.Origins = nil
		} else {
			c.Origins = parseStringArray(value)
		}
		return determineStringSliceResult(oldValue, c.Origins), nil

	case "allow_methods":
		oldValue := c.Methods
		if value == "" {
			c.Methods = nil
		} else {
			c.Methods = parseStringArray(value)
		}
		return determineStringSliceResult(oldValue, c.Methods), nil

	case "allow_headers":
		oldValue := c.Headers
		if value == "" {
			c.Headers = nil
		} else {
			c.Headers = parseStringArray(value)
		}
		return determineStringSliceResult(oldValue, c.Headers), nil

	case "expose_headers":
		oldValue := c.ExposeHeaders
		if value == "" {
			c.ExposeHeaders = nil
		} else {
			c.ExposeHeaders = parseStringArray(value)
		}
		return determineStringSliceResult(oldValue, c.ExposeHeaders), nil

	case "allow_credentials":
		oldValue := c.Credentials
		if value == "" {
			c.Credentials = nil
		} else {
			boolValue, err := parseBool(value)
			if err != nil {
				return context.Noop, fmt.Errorf("%w: %w", NewErrInvalidValue("allow_credentials", value), err)
			}
			c.Credentials = &boolValue
		}
		return determineBoolPtrResult(oldValue, c.Credentials), nil

	case "max_age":
		oldValue := c.MaxAge
		if value == "" {
			c.MaxAge = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("%w: %w", NewErrInvalidValue("max_age", value), err)
			}
			c.MaxAge = &duration
		}
		return determineDurationPtrResult(oldValue, c.MaxAge), nil

	default:
		return context.Noop, fmt.Errorf("unknown CORS config key: %s", key)
	}
}

// Validate implements Validator for CORSConfigSection.
// Validates CORS configuration values.
func (c *CORSConfigSection) Validate() error {
	var validationErrors []error

	// Validate origins.
	for _, origin := range c.Origins {
		// Wildcard origin check.
		// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Access-Control-Allow-Origin#sect
		if origin == "*" {
			continue
		}

		if origin == "" {
			validationErrors = append(validationErrors, fmt.Errorf("CORS origin cannot be empty"))
			continue
		}

		if !isValidAddr(origin) {
			validationErrors = append(validationErrors, fmt.Errorf("invalid origin address: %s", origin))
			continue
		}
	}

	// Validate methods.
	validMethods := ValidHTTPRequestMethods()
	for _, method := range c.Methods {
		// Wildcard method check.
		// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Access-Control-Allow-Methods#sect
		if method == "*" {
			continue
		}

		if method == "" {
			validationErrors = append(validationErrors, fmt.Errorf("CORS method cannot be empty"))
			continue
		}

		if _, ok := validMethods[method]; !ok {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("CORS method %s is not a valid HTTP request method", method),
			)
			continue
		}
	}

	// Validate max age.
	if c.MaxAge != nil {
		if *c.MaxAge <= 0 {
			validationErrors = append(validationErrors, fmt.Errorf("CORS max age must be positive"))
		}
	}

	return errors.Join(validationErrors...)
}

// AvailableKeys implements SchemaProvider for DaemonConfig.
func (d *DaemonConfig) AvailableKeys() []SchemaKey {
	var keys []SchemaKey

	// Always return API keys regardless of whether API section exists
	apiSection := &APIConfigSection{}
	for _, key := range apiSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "api." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	// Always return MCP keys regardless of whether MCP section exists
	mcpSection := &MCPConfigSection{}
	for _, key := range mcpSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "mcp." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	return keys
}

// Get implements Getter for DaemonConfig.
// Routes configuration retrieval to the appropriate subsection.
// When called with no keys, returns the entire daemon configuration structure.
func (d *DaemonConfig) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return d.getAll()
	}

	section := normalizeKey(keys[0])

	switch section {
	case "api":
		if d.API == nil {
			return nil, fmt.Errorf("no API configuration found")
		}
		return d.API.Get(keys[1:]...)
	case "mcp":
		if d.MCP == nil {
			return nil, fmt.Errorf("no MCP configuration found")
		}
		return d.MCP.Get(keys[1:]...)
	default:
		return nil, fmt.Errorf("unknown daemon config section: %s", section)
	}
}

// Set implements Setter for DaemonConfig.
// Routes configuration changes to the appropriate subsection.
func (d *DaemonConfig) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	section := normalizeKey(parts[0])

	switch section {
	case "api":
		if d.API == nil {
			d.API = &APIConfigSection{}
		}
		return d.API.Set(strings.Join(parts[1:], "."), value)
	case "mcp":
		if d.MCP == nil {
			d.MCP = &MCPConfigSection{}
		}
		return d.MCP.Set(strings.Join(parts[1:], "."), value)
	default:
		return context.Noop, fmt.Errorf("unknown daemon config section: %s", section)
	}
}

// Validate implements Validator for DaemonConfig.
// Validates daemon configuration by delegating to subsections.
func (d *DaemonConfig) Validate() error {
	if d == nil {
		return fmt.Errorf("no daemon configuration found")
	}

	var validationErrors []error

	// Validate subsections with error wrapping.
	if d.API != nil {
		if err := d.API.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("API configuration error: %w", err))
		}
	}

	if d.MCP != nil {
		if err := d.MCP.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("MCP configuration error: %w", err))
		}
	}

	return errors.Join(validationErrors...)
}

// MarshalText implements encoding.TextMarshaler for Duration.
func (d *Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(*d).String()), nil
}

// String returns a human-readable string representation of the duration.
func (d *Duration) String() string {
	if d == nil {
		return ""
	}

	duration := time.Duration(*d)

	// List of duration units in descending order.
	units := []struct {
		unit   time.Duration
		suffix string
	}{
		{time.Hour, "h"},
		{time.Minute, "m"},
		{time.Second, "s"},
		{time.Millisecond, "ms"},
		{time.Microsecond, "µs"},
		{time.Nanosecond, "ns"},
	}

	for _, u := range units {
		if duration%u.unit == 0 {
			return fmt.Sprintf("%d%s", duration/u.unit, u.suffix)
		}
	}

	// Fallback to nanoseconds if no exact match.
	return fmt.Sprintf("%dns", duration)
}

// UnmarshalText implements encoding.TextUnmarshaler for Duration.
func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// AvailableKeys implements SchemaProvider for MCPConfigSection.
func (m *MCPConfigSection) AvailableKeys() []SchemaKey {
	var keys []SchemaKey

	// Always return timeout keys regardless of whether timeout section exists
	timeoutSection := &MCPTimeoutConfigSection{}
	for _, key := range timeoutSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "timeout." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	// Always return interval keys regardless of whether interval section exists
	intervalSection := &MCPIntervalConfigSection{}
	for _, key := range intervalSection.AvailableKeys() {
		keys = append(keys, SchemaKey{
			Path:        "interval." + key.Path,
			Type:        key.Type,
			Description: key.Description,
		})
	}

	return keys
}

// Get implements Getter for MCPConfigSection.
// Returns all MCP configuration when called with no keys, or routes to subsections when keys are provided.
func (m *MCPConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return m.getAll()
	}

	subsection := normalizeKey(keys[0])

	if len(keys) == 1 {
		switch subsection {
		case "timeout":
			if m.Timeout == nil {
				return nil, fmt.Errorf("mcp.timeout not set")
			}
			return m.Timeout.Get()
		case "interval":
			if m.Interval == nil {
				return nil, fmt.Errorf("mcp.interval not set")
			}
			return m.Interval.Get()
		default:
			return nil, fmt.Errorf("unknown MCP config key: %s", subsection)
		}
	}

	// Handle multi-key access to subsections
	switch subsection {
	case "timeout":
		if m.Timeout == nil {
			return nil, fmt.Errorf("mcp.timeout not set")
		}
		return m.Timeout.Get(keys[1:]...)
	case "interval":
		if m.Interval == nil {
			return nil, fmt.Errorf("mcp.interval not set")
		}
		return m.Interval.Get(keys[1:]...)
	default:
		return nil, fmt.Errorf("unknown MCP subsection: %s", subsection)
	}
}

// Set implements Setter for MCPConfigSection.
// Routes MCP configuration changes to the appropriate subsection.
func (m *MCPConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("MCP config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	subsection := normalizeKey(parts[0])

	switch subsection {
	case "timeout":
		if m.Timeout == nil {
			m.Timeout = &MCPTimeoutConfigSection{}
		}
		return m.Timeout.Set(strings.Join(parts[1:], "."), value)
	case "interval":
		if m.Interval == nil {
			m.Interval = &MCPIntervalConfigSection{}
		}
		return m.Interval.Set(strings.Join(parts[1:], "."), value)
	default:
		return context.Noop, fmt.Errorf("invalid MCP path, expected subsection.key: %s", path)
	}
}

// Validate implements Validator for MCPConfigSection.
// Validates MCP configuration by delegating to subsections.
func (m *MCPConfigSection) Validate() error {
	if m == nil {
		return nil
	}

	var validationErrors []error

	// Validate subsections with error wrapping.
	if m.Timeout != nil {
		if err := m.Timeout.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("timeout configuration error: %w", err))
		}
	}

	if m.Interval != nil {
		if err := m.Interval.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("interval configuration error: %w", err))
		}
	}

	return errors.Join(validationErrors...)
}

// AvailableKeys implements SchemaProvider for MCPIntervalConfigSection.
func (m *MCPIntervalConfigSection) AvailableKeys() []SchemaKey {
	return []SchemaKey{
		{Path: "health", Type: "duration", Description: "Health check interval for MCP servers"},
	}
}

// Get implements Getter for MCPIntervalConfigSection.
// Returns all interval configuration when called with no keys, or specific values when keys are provided.
func (m *MCPIntervalConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return m.getAll()
	}

	if err := ensureSingleKey(keys, "MCP interval"); err != nil {
		return nil, err
	}

	key := normalizeKey(keys[0])

	switch key {
	case "health":
		if m.Health == nil {
			return nil, fmt.Errorf("mcp.interval.health not set")
		}
		return *m.Health, nil
	default:
		return nil, fmt.Errorf("unknown MCP interval config key: %s", key)
	}
}

// Set implements Setter for MCPIntervalConfigSection.
// Handles MCP interval configuration at the leaf level.
func (m *MCPIntervalConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("path cannot be empty")
	}

	key := normalizeKey(path)

	switch key {
	case "health":
		oldValue := m.Health
		if value == "" {
			m.Health = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("invalid duration for health: %w", err)
			}
			m.Health = &duration
		}
		return determineDurationPtrResult(oldValue, m.Health), nil
	default:
		return context.Noop, fmt.Errorf("unknown MCP interval config key: %s", key)
	}
}

// Validate implements Validator for MCPIntervalConfigSection.
// Validates MCP interval configuration values.
func (m *MCPIntervalConfigSection) Validate() error {
	if m == nil {
		return nil
	}

	if m.Health != nil {
		if *m.Health <= 0 {
			return fmt.Errorf("MCP health interval must be positive")
		}
	}

	return nil
}

// AvailableKeys implements SchemaProvider for MCPTimeoutConfigSection.
func (m *MCPTimeoutConfigSection) AvailableKeys() []SchemaKey {
	return []SchemaKey{
		{Path: "shutdown", Type: "duration", Description: "MCP server shutdown timeout"},
		{Path: "init", Type: "duration", Description: "MCP server initialization timeout"},
		{Path: "health", Type: "duration", Description: "Health check timeout for MCP servers"},
	}
}

// Get implements Getter for MCPTimeoutConfigSection.
// Returns all timeout configuration when called with no keys, or specific values when keys are provided.
func (m *MCPTimeoutConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		return m.getAll()
	}

	if err := ensureSingleKey(keys, "MCP timeout"); err != nil {
		return nil, err
	}

	key := normalizeKey(keys[0])

	switch key {
	case "shutdown":
		if m.Shutdown == nil {
			return nil, fmt.Errorf("mcp.timeout.shutdown not set")
		}
		return *m.Shutdown, nil
	case "init":
		if m.Init == nil {
			return nil, fmt.Errorf("mcp.timeout.init not set")
		}
		return *m.Init, nil
	case "health":
		if m.Health == nil {
			return nil, fmt.Errorf("mcp.timeout.health not set")
		}
		return *m.Health, nil
	default:
		return nil, fmt.Errorf("unknown MCP timeout config key: %s", key)
	}
}

// Set implements Setter for MCPTimeoutConfigSection.
// Handles MCP timeout configuration at the leaf level.
func (m *MCPTimeoutConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	if strings.TrimSpace(path) == "" {
		return context.Noop, fmt.Errorf("path cannot be empty")
	}

	key := normalizeKey(path)

	switch key {
	case "shutdown":
		oldValue := m.Shutdown
		if value == "" {
			m.Shutdown = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("%w: %w", NewErrInvalidValue("shutdown", value), err)
			}
			m.Shutdown = &duration
		}
		return determineDurationPtrResult(oldValue, m.Shutdown), nil
	case "init":
		oldValue := m.Init
		if value == "" {
			m.Init = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("invalid duration for init: %w", err)
			}
			m.Init = &duration
		}
		return determineDurationPtrResult(oldValue, m.Init), nil
	case "health":
		oldValue := m.Health
		if value == "" {
			m.Health = nil
		} else {
			duration, err := parseDuration(value)
			if err != nil {
				return context.Noop, fmt.Errorf("invalid duration for health: %w", err)
			}
			m.Health = &duration
		}
		return determineDurationPtrResult(oldValue, m.Health), nil
	default:
		return context.Noop, fmt.Errorf("unknown MCP timeout config key: %s", key)
	}
}

// Validate implements Validator for MCPTimeoutConfigSection.
// Validates MCP timeout configuration values.
func (m *MCPTimeoutConfigSection) Validate() error {
	if m == nil {
		return nil
	}

	var validationErrors []error

	if m.Shutdown != nil {
		if *m.Shutdown <= 0 {
			validationErrors = append(validationErrors, fmt.Errorf("MCP shutdown timeout must be positive"))
		}
	}

	if m.Init != nil {
		if *m.Init <= 0 {
			validationErrors = append(validationErrors, fmt.Errorf("MCP init timeout must be positive"))
		}
	}

	if m.Health != nil {
		if *m.Health <= 0 {
			validationErrors = append(validationErrors, fmt.Errorf("MCP health timeout must be positive"))
		}
	}

	return errors.Join(validationErrors...)
}

// ValidHTTPRequestMethods returns a map of all valid HTTP request methods.
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods
func ValidHTTPRequestMethods() map[string]struct{} {
	return map[string]struct{}{
		http.MethodGet:     {},
		http.MethodHead:    {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodDelete:  {},
		http.MethodConnect: {},
		http.MethodOptions: {},
		http.MethodTrace:   {},
		http.MethodPatch:   {},
	}
}

// getAll returns all configured values for the APIConfigSection (and subsections).
func (a *APIConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if a.Addr != nil {
		result["addr"] = *a.Addr
	}

	if a.Timeout != nil {
		timeoutResult, _ := a.Timeout.Get()
		if timeoutResult != nil {
			if timeoutMap, ok := timeoutResult.(map[string]any); ok && len(timeoutMap) > 0 {
				result["timeout"] = timeoutResult
			}
		}
	}

	if a.CORS != nil {
		corsResult, _ := a.CORS.Get()
		if corsResult != nil {
			if corsMap, ok := corsResult.(map[string]any); ok && len(corsMap) > 0 {
				result["cors"] = corsResult
			}
		}
	}

	return result, nil
}

// getAll returns all configured values for the APITimeoutConfigSection.
func (a *APITimeoutConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if a.Shutdown != nil {
		result["shutdown"] = *a.Shutdown
	}

	return result, nil
}

// getAll returns all configured values for the CORSConfigSection.
func (c *CORSConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if c.Enable != nil {
		result["enable"] = *c.Enable
	}
	if len(c.Origins) > 0 {
		result["allow_origins"] = c.Origins
	}
	if len(c.Methods) > 0 {
		result["allow_methods"] = c.Methods
	}
	if len(c.Headers) > 0 {
		result["allow_headers"] = c.Headers
	}
	if len(c.ExposeHeaders) > 0 {
		result["expose_headers"] = c.ExposeHeaders
	}
	if c.Credentials != nil {
		result["allow_credentials"] = *c.Credentials
	}
	if c.MaxAge != nil {
		result["max_age"] = *c.MaxAge
	}

	return result, nil
}

// getAll returns all configured values for the DaemonConfig (and subsections).
func (d *DaemonConfig) getAll() (any, error) {
	result := make(map[string]any)

	if d.API != nil {
		apiResult, _ := d.API.Get()
		if apiResult != nil {
			if apiMap, ok := apiResult.(map[string]any); ok && len(apiMap) > 0 {
				result["api"] = apiResult
			}
		}
	}

	if d.MCP != nil {
		mcpResult, _ := d.MCP.Get()
		if mcpResult != nil {
			if mcpMap, ok := mcpResult.(map[string]any); ok && len(mcpMap) > 0 {
				result["mcp"] = mcpResult
			}
		}
	}

	return result, nil
}

// getAll returns all configured values for the MCPConfigSection (and subsections).
func (m *MCPConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if m.Timeout != nil {
		timeoutResult, _ := m.Timeout.Get()
		if timeoutResult != nil {
			if timeoutMap, ok := timeoutResult.(map[string]any); ok && len(timeoutMap) > 0 {
				result["timeout"] = timeoutResult
			}
		}
	}

	if m.Interval != nil {
		intervalResult, _ := m.Interval.Get()
		if intervalResult != nil {
			if intervalMap, ok := intervalResult.(map[string]any); ok && len(intervalMap) > 0 {
				result["interval"] = intervalResult
			}
		}
	}

	return result, nil
}

// getAll returns all configured values for the MCPIntervalConfigSection.
func (m *MCPIntervalConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if m.Health != nil {
		result["health"] = *m.Health
	}

	return result, nil
}

// getAll returns all configured values for the MCPTimeoutConfigSection.
func (m *MCPTimeoutConfigSection) getAll() (any, error) {
	result := make(map[string]any)

	if m.Shutdown != nil {
		result["shutdown"] = *m.Shutdown
	}
	if m.Init != nil {
		result["init"] = *m.Init
	}
	if m.Health != nil {
		result["health"] = *m.Health
	}

	return result, nil
}

// isValidAddr performs basic validation for host:port format using stdlib.
func isValidAddr(addr string) bool {
	// Use net.SplitHostPort for proper parsing
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	// Special case: ":" (empty host, empty port) is valid for bind-all-interfaces
	if host == "" && port == "" {
		return true
	}

	// Port must not be empty (except for the special case above)
	if port == "" {
		return false
	}

	// Host validation: either empty (wildcard), valid IP, or valid hostname
	if host != "" {
		// Check for invalid characters in hostname (spaces, etc.)
		if strings.ContainsAny(host, " \t\n\r") {
			return false
		}

		// Try parsing as IP first
		if net.ParseIP(host) == nil {
			// If not an IP, should be a valid hostname (basic check)
			if len(host) == 0 || len(host) > 253 {
				return false
			}
		}
	}

	return true
}

// normalizeKey normalizes a key by trimming whitespace and converting to lowercase.
func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// parseBool parses a string into a boolean value, supporting multiple formats.
func parseBool(value string) (bool, error) {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid value: '%s'", value)
	}

	return v, nil
}

// parseDuration parses a string into a Duration value.
func parseDuration(value string) (Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}
	return Duration(duration), nil
}

// parseStringArray parses a comma-separated string into a slice of trimmed strings.
func parseStringArray(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}
	return result
}

// ensureSingleKey ensures that only a single key is provided for leaf-level config.
func ensureSingleKey(keys []string, configType string) error {
	if len(keys) > 1 {
		return fmt.Errorf("%s %w: %s", configType, ErrInvalidKey, strings.Join(keys, "."))
	}
	return nil
}

// determineBoolPtrResult determines the UpsertResult for boolean pointer changes.
func determineBoolPtrResult(old *bool, new *bool) context.UpsertResult {
	switch {
	case old == nil && new == nil:
		return context.Noop
	case old == nil:
		return context.Created
	case new == nil:
		return context.Deleted
	case *old != *new:
		return context.Updated
	default:
		return context.Noop
	}
}

// determineDurationPtrResult determines the UpsertResult for Duration pointer changes.
func determineDurationPtrResult(old *Duration, new *Duration) context.UpsertResult {
	switch {
	case old == nil && new == nil:
		return context.Noop
	case old == nil:
		return context.Created
	case new == nil:
		return context.Deleted
	case *old != *new:
		return context.Updated
	default:
		return context.Noop
	}
}

// determineStringPtrResult determines the UpsertResult for string pointer changes.
func determineStringPtrResult(old *string, new *string) context.UpsertResult {
	switch {
	case old == nil && new == nil:
		return context.Noop
	case old == nil:
		return context.Created
	case new == nil:
		return context.Deleted
	case *old != *new:
		return context.Updated
	default:
		return context.Noop
	}
}

// determineStringSliceResult determines the UpsertResult for string slice changes.
func determineStringSliceResult(old []string, new []string) context.UpsertResult {
	switch {
	case len(old) == 0 && len(new) == 0:
		return context.Noop
	case len(old) == 0:
		return context.Created
	case len(new) == 0:
		return context.Deleted
	default:
		if len(old) != len(new) {
			return context.Updated
		}
		for i := range old {
			if old[i] != new[i] {
				return context.Updated
			}
		}
		return context.Noop
	}
}
