package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

// Duration is a custom time.Duration type that provides improved marshaling.
type Duration time.Duration

// DaemonConfig represents daemon-specific configuration that can be stored in .mcpd.toml.
// This extends the existing Config struct with daemon settings.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type DaemonConfig struct {
	// API configuration (includes address and nested timeout/cors)
	API *APIConfigSection `json:"api,omitempty" toml:"api,omitempty" yaml:"api,omitempty"`

	// MCP configuration (includes nested timeout and interval settings)
	MCP *MCPConfigSection `json:"mcp,omitempty" toml:"mcp,omitempty" yaml:"mcp,omitempty"`
}

// APIConfigSection contains API server configuration settings.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type APIConfigSection struct {
	// Address to bind the API server (e.g., "0.0.0.0:8090")
	// Maps to CLI flag --addr
	Addr *string `json:"addr,omitempty" toml:"addr,omitempty" yaml:"addr,omitempty"`

	// Nested timeout configuration for API operations
	Timeout *APITimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// CORS configuration for API server
	CORS *CORSConfigSection `json:"cors,omitempty" toml:"cors,omitempty" yaml:"cors,omitempty"`
}

// APITimeoutConfigSection contains API-specific timeout configuration.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type APITimeoutConfigSection struct {
	// API shutdown timeout for graceful shutdown
	// Maps to CLI flag --timeout-api-shutdown
	Shutdown *Duration `json:"shutdown,omitempty" toml:"shutdown,omitempty" yaml:"shutdown,omitempty"`
}

// CORSConfigSection contains CORS configuration.
// Maps to CLI flags --cors-*
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type CORSConfigSection struct {
	// Enable CORS support
	// Maps to CLI flag --cors-enable
	Enable *bool `json:"enable" toml:"enable" yaml:"enable"`

	// Allowed origins (e.g., ["http://localhost:3000", "https://example.com"])
	// Use ["*"] to allow all origins
	// Maps to CLI flag --cors-origin (repeated)
	Origins []string `json:"allowOrigins,omitempty" toml:"allow_origins,omitempty" yaml:"allow_origins,omitempty"`

	// Allowed HTTP methods
	// Maps to CLI flag --cors-method (repeated)
	Methods []string `json:"allowMethods,omitempty" toml:"allow_methods,omitempty" yaml:"allow_methods,omitempty"`

	// Allowed headers.
	Headers []string `json:"allowHeaders,omitempty" toml:"allow_headers,omitempty" yaml:"allow_headers,omitempty"`

	// Headers to expose.
	ExposeHeaders []string `json:"exposeHeaders,omitempty" toml:"expose_headers,omitempty" yaml:"expose_headers,omitempty"`

	// Allow credentials
	// Maps to CLI flag --cors-credentials
	Credentials *bool `json:"allowCredentials" toml:"allow_credentials" yaml:"allow_credentials"`

	// Max age for preflight requests (in seconds)
	// Maps to CLI flag --cors-max-age
	MaxAge *Duration `json:"maxAge,omitempty" toml:"max_age,omitempty" yaml:"max_age,omitempty"`
}

// MCPConfigSection contains MCP server configuration settings.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type MCPConfigSection struct {
	// Nested timeout configuration for MCP operations
	Timeout *MCPTimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Nested interval configuration for MCP operations
	Interval *MCPIntervalConfigSection `json:"interval,omitempty" toml:"interval,omitempty" yaml:"interval,omitempty"`
}

// MCPIntervalConfigSection contains MCP-specific interval configuration.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type MCPIntervalConfigSection struct {
	// MCP health check interval
	// Maps to CLI flag --interval-mcp-health
	Health *Duration `json:"health,omitempty" toml:"health,omitempty" yaml:"health,omitempty"`
}

// MCPTimeoutConfigSection contains MCP-specific timeout configuration.
// NOTE: if you add/remove fields you must review the associated ConfigGetter, ConfigSetter implementations.
type MCPTimeoutConfigSection struct {
	// MCP server shutdown timeout
	// Maps to CLI flag --timeout-mcp-shutdown
	Shutdown *Duration `json:"shutdown,omitempty" toml:"shutdown,omitempty" yaml:"shutdown,omitempty"`

	// MCP server initialization timeout
	// Maps to CLI flag --timeout-mcp-init
	Init *Duration `json:"init,omitempty" toml:"init,omitempty" yaml:"init,omitempty"`

	// MCP health check timeout
	// Maps to CLI flag --timeout-mcp-health
	Health *Duration `json:"health,omitempty" toml:"health,omitempty" yaml:"health,omitempty"`
}

// Get implements ConfigGetter for APIConfigSection.
// Returns all API configuration when called with no keys, or specific values when keys are provided.
func (a *APIConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
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

	key := strings.ToLower(strings.TrimSpace(keys[0]))

	if len(keys) == 1 {
		switch key {
		case "addr":
			if a.Addr == nil {
				return nil, fmt.Errorf("api.addr not set")
			}
			return *a.Addr, nil
		case "timeout":
			if a.Timeout == nil {
				return nil, fmt.Errorf("no API timeout configuration found")
			}
			return a.Timeout.Get()
		case "cors":
			if a.CORS == nil {
				return nil, fmt.Errorf("no CORS configuration found")
			}
			return a.CORS.Get()
		default:
			return nil, fmt.Errorf("unknown API config key: %s", key)
		}
	}

	switch key {
	case "timeout":
		if a.Timeout == nil {
			return nil, fmt.Errorf("no API timeout configuration found")
		}
		return a.Timeout.Get(keys[1:]...)
	case "cors":
		if a.CORS == nil {
			return nil, fmt.Errorf("no CORS configuration found")
		}
		return a.CORS.Get(keys[1:]...)
	default:
		return nil, fmt.Errorf("unknown API subsection: %s", key)
	}
}

// Get implements ConfigGetter for APITimeoutConfigSection.
// Returns all timeout configuration when called with no keys, or specific values when keys are provided.
func (a *APITimeoutConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		result := make(map[string]any)

		if a.Shutdown != nil {
			result["shutdown"] = a.Shutdown.String()
		}

		return result, nil
	}

	if len(keys) > 1 {
		return nil, fmt.Errorf("API timeout config does not support nested keys")
	}

	key := strings.ToLower(strings.TrimSpace(keys[0]))

	switch key {
	case "shutdown":
		if a.Shutdown == nil {
			return nil, fmt.Errorf("api.timeout.shutdown not set")
		}
		return a.Shutdown.String(), nil
	default:
		return nil, fmt.Errorf("unknown API timeout config key: %s", key)
	}
}

// Get implements ConfigGetter for CORSConfigSection.
// Returns all CORS configuration when called with no keys, or specific values when keys are provided.
func (c *CORSConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
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
			result["max_age"] = c.MaxAge.String()
		}

		return result, nil
	}

	if len(keys) > 1 {
		return nil, fmt.Errorf("CORS config does not support nested keys")
	}

	key := strings.ToLower(strings.TrimSpace(keys[0]))

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
		return c.MaxAge.String(), nil
	default:
		return nil, fmt.Errorf("unknown CORS config key: %s", key)
	}
}

// Get implements ConfigGetter for DaemonConfig.
// Routes configuration retrieval to the appropriate subsection.
// When called with no keys, returns the entire daemon configuration structure.
func (d *DaemonConfig) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
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

	section := strings.ToLower(strings.TrimSpace(keys[0]))

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

// EnableOrDefault returns the CORS enable setting, falling back to defaultEnable if not set.
func (c *CORSConfigSection) EnableOrDefault(defaultEnable bool) bool {
	if c == nil || c.Enable == nil {
		return defaultEnable
	}
	return *c.Enable
}

// Get implements ConfigGetter for MCPConfigSection.
// Returns all MCP configuration when called with no keys, or routes to subsections when keys are provided.
func (m *MCPConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
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

	subsection := strings.ToLower(strings.TrimSpace(keys[0]))

	switch subsection {
	case "timeout":
		if m.Timeout == nil {
			return nil, fmt.Errorf("no MCP timeout configuration found")
		}
		return m.Timeout.Get(keys[1:]...)
	case "interval":
		if m.Interval == nil {
			return nil, fmt.Errorf("no MCP interval configuration found")
		}
		return m.Interval.Get(keys[1:]...)
	default:
		return nil, fmt.Errorf("unknown MCP subsection: %s", subsection)
	}
}

// Get implements ConfigGetter for MCPIntervalConfigSection.
// Returns all interval configuration when called with no keys, or specific values when keys are provided.
func (m *MCPIntervalConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		result := make(map[string]any)

		if m.Health != nil {
			result["health"] = m.Health.String()
		}

		return result, nil
	}

	if len(keys) > 1 {
		return nil, fmt.Errorf("MCP interval config does not support nested keys")
	}

	key := strings.ToLower(strings.TrimSpace(keys[0]))

	switch key {
	case "health":
		if m.Health == nil {
			return nil, fmt.Errorf("mcp.interval.health not set")
		}
		return m.Health.String(), nil
	default:
		return nil, fmt.Errorf("unknown MCP interval config key: %s", key)
	}
}

// Get implements ConfigGetter for MCPTimeoutConfigSection.
// Returns all timeout configuration when called with no keys, or specific values when keys are provided.
func (m *MCPTimeoutConfigSection) Get(keys ...string) (any, error) {
	if len(keys) == 0 {
		result := make(map[string]any)

		if m.Shutdown != nil {
			result["shutdown"] = m.Shutdown.String()
		}
		if m.Init != nil {
			result["init"] = m.Init.String()
		}
		if m.Health != nil {
			result["health"] = m.Health.String()
		}

		return result, nil
	}

	if len(keys) > 1 {
		return nil, fmt.Errorf("MCP timeout config does not support nested keys")
	}

	key := strings.ToLower(strings.TrimSpace(keys[0]))

	switch key {
	case "shutdown":
		if m.Shutdown == nil {
			return nil, fmt.Errorf("mcp.timeout.shutdown not set")
		}
		return m.Shutdown.String(), nil
	case "init":
		if m.Init == nil {
			return nil, fmt.Errorf("mcp.timeout.init not set")
		}
		return m.Init.String(), nil
	case "health":
		if m.Health == nil {
			return nil, fmt.Errorf("mcp.timeout.health not set")
		}
		return m.Health.String(), nil
	default:
		return nil, fmt.Errorf("unknown MCP timeout config key: %s", key)
	}
}

// MarshalText implements encoding.TextMarshaler for Duration.
func (d *Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(*d).String()), nil
}

// Set implements ConfigSetter for APIConfigSection.
// Handles API-specific configuration including nested timeout and CORS settings.
func (a *APIConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("API config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	key := strings.ToLower(parts[0])

	if len(parts) == 1 {
		switch key {
		case "addr":
			old := a.Addr
			if value == "" {
				a.Addr = nil
				return determineStringPtrResult(old, nil), nil
			}
			a.Addr = &value
			return determineStringPtrResult(old, &value), nil
		default:
			return context.Noop, fmt.Errorf("unknown API config key: %s", key)
		}
	}

	subsection := key
	subPath := strings.Join(parts[1:], ".")

	switch subsection {
	case "timeout":
		if a.Timeout == nil {
			a.Timeout = &APITimeoutConfigSection{}
		}
		return a.Timeout.Set(subPath, value)
	case "cors":
		if a.CORS == nil {
			a.CORS = &CORSConfigSection{}
		}
		return a.CORS.Set(subPath, value)
	default:
		return context.Noop, fmt.Errorf("unknown API subsection: %s", subsection)
	}
}

// Set implements ConfigSetter for APITimeoutConfigSection.
// Handles API timeout configuration at the leaf level.
func (a *APITimeoutConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("API timeout config path cannot be empty")
	}

	key := strings.ToLower(path)

	switch key {
	case "shutdown":
		old := a.Shutdown
		if value == "" {
			a.Shutdown = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for shutdown: %s", value)
		}
		a.Shutdown = &duration
		return determineDurationPtrResult(old, &duration), nil
	default:
		return context.Noop, fmt.Errorf("unknown API timeout config key: %s", key)
	}
}

// Set implements ConfigSetter for CORSConfigSection.
// Handles CORS-specific configuration at the leaf level.
func (c *CORSConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("CORS config path cannot be empty")
	}

	key := strings.ToLower(path)

	switch key {
	case "enable":
		old := c.Enable
		if value == "" {
			c.Enable = nil
			return determineBoolPtrResult(old, nil), nil
		}
		boolVal, err := parseBool(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid boolean value for enable: %s", value)
		}
		c.Enable = &boolVal
		return determineBoolPtrResult(old, &boolVal), nil
	case "allow_origins":
		old := c.Origins
		if value == "" {
			c.Origins = []string{}
		} else {
			c.Origins = parseStringArray(value)
		}
		return determineStringSliceResult(old, c.Origins), nil
	case "allow_methods":
		old := c.Methods
		if value == "" {
			c.Methods = []string{}
		} else {
			c.Methods = parseStringArray(value)
		}
		return determineStringSliceResult(old, c.Methods), nil
	case "allow_headers":
		old := c.Headers
		if value == "" {
			c.Headers = []string{}
		} else {
			c.Headers = parseStringArray(value)
		}
		return determineStringSliceResult(old, c.Headers), nil
	case "expose_headers":
		old := c.ExposeHeaders
		if value == "" {
			c.ExposeHeaders = []string{}
		} else {
			c.ExposeHeaders = parseStringArray(value)
		}
		return determineStringSliceResult(old, c.ExposeHeaders), nil
	case "allow_credentials":
		old := c.Credentials
		if value == "" {
			c.Credentials = nil
			return determineBoolPtrResult(old, nil), nil
		}
		boolVal, err := parseBool(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid boolean value for allow_credentials: %s", value)
		}
		c.Credentials = &boolVal
		return determineBoolPtrResult(old, &boolVal), nil
	case "max_age":
		old := c.MaxAge
		if value == "" {
			c.MaxAge = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for max_age: %s", value)
		}
		c.MaxAge = &duration
		return determineDurationPtrResult(old, &duration), nil
	default:
		return context.Noop, fmt.Errorf("unknown CORS config key: %s", key)
	}
}

// Set implements ConfigSetter for DaemonConfig.
// Routes configuration changes to the appropriate subsection.
func (d *DaemonConfig) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.ToLower(strings.TrimSpace(path))
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	section := strings.ToLower(parts[0])

	switch section {
	case "api":
		if d.API == nil {
			d.API = &APIConfigSection{}
		}
		result, err := d.API.Set(strings.Join(parts[1:], "."), value)
		if err != nil {
			return context.Noop, fmt.Errorf("failed to set daemon config '%s': %w", path, err)
		}
		return result, nil
	case "mcp":
		if d.MCP == nil {
			d.MCP = &MCPConfigSection{}
		}
		result, err := d.MCP.Set(strings.Join(parts[1:], "."), value)
		if err != nil {
			return context.Noop, fmt.Errorf("failed to set daemon config '%s': %w", path, err)
		}
		return result, nil
	default:
		return context.Noop, fmt.Errorf("unknown daemon config section: %s", section)
	}
}

// Set implements ConfigSetter for MCPConfigSection.
// Routes MCP configuration changes to the appropriate subsection.
func (m *MCPConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("MCP config path cannot be empty")
	}

	parts := strings.Split(path, ".")
	subsection := strings.ToLower(parts[0])

	if len(parts) < 2 {
		return context.Noop, fmt.Errorf("invalid MCP path, expected subsection.key: %s", path)
	}

	subPath := strings.Join(parts[1:], ".")

	switch subsection {
	case "timeout":
		if m.Timeout == nil {
			m.Timeout = &MCPTimeoutConfigSection{}
		}
		return m.Timeout.Set(subPath, value)
	case "interval":
		if m.Interval == nil {
			m.Interval = &MCPIntervalConfigSection{}
		}
		return m.Interval.Set(subPath, value)
	default:
		return context.Noop, fmt.Errorf("unknown MCP subsection: %s", subsection)
	}
}

// Set implements ConfigSetter for MCPIntervalConfigSection.
// Handles MCP interval configuration at the leaf level.
func (m *MCPIntervalConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("MCP interval config path cannot be empty")
	}

	key := strings.ToLower(path)

	switch key {
	case "health":
		old := m.Health
		if value == "" {
			m.Health = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for health: %s", value)
		}
		m.Health = &duration
		return determineDurationPtrResult(old, &duration), nil
	default:
		return context.Noop, fmt.Errorf("unknown MCP interval config key: %s", key)
	}
}

// Set implements ConfigSetter for MCPTimeoutConfigSection.
// Handles MCP timeout configuration at the leaf level.
func (m *MCPTimeoutConfigSection) Set(path string, value string) (context.UpsertResult, error) {
	path = strings.TrimSpace(path)
	value = strings.TrimSpace(value)

	if path == "" {
		return context.Noop, fmt.Errorf("MCP timeout config path cannot be empty")
	}

	key := strings.ToLower(path)

	switch key {
	case "shutdown":
		old := m.Shutdown
		if value == "" {
			m.Shutdown = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for shutdown: %s", value)
		}
		m.Shutdown = &duration
		return determineDurationPtrResult(old, &duration), nil
	case "init":
		old := m.Init
		if value == "" {
			m.Init = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for init: %s", value)
		}
		m.Init = &duration
		return determineDurationPtrResult(old, &duration), nil
	case "health":
		old := m.Health
		if value == "" {
			m.Health = nil
			return determineDurationPtrResult(old, nil), nil
		}
		duration, err := parseDuration(value)
		if err != nil {
			return context.Noop, fmt.Errorf("invalid duration value for health: %s", value)
		}
		m.Health = &duration
		return determineDurationPtrResult(old, &duration), nil
	default:
		return context.Noop, fmt.Errorf("unknown MCP timeout config key: %s", key)
	}
}

// String returns a human-readable string representation of the duration.
func (d *Duration) String() string {
	if d == nil {
		return ""
	}

	dur := time.Duration(*d)
	switch {
	case dur%time.Hour == 0:
		return fmt.Sprintf("%dh", dur/time.Hour)
	case dur%time.Minute == 0:
		return fmt.Sprintf("%dm", dur/time.Minute)
	case dur%time.Second == 0:
		return fmt.Sprintf("%ds", dur/time.Second)
	case dur%time.Millisecond == 0:
		return fmt.Sprintf("%dms", dur/time.Millisecond)
	case dur%time.Microsecond == 0:
		return fmt.Sprintf("%dµs", dur/time.Microsecond)
	default:
		return fmt.Sprintf("%dns", dur)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler for Duration.
func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// AvailableKeys implements ConfigSchema for APITimeoutConfigSection.
func (a *APITimeoutConfigSection) AvailableKeys() []ConfigKey {
	return []ConfigKey{
		{Path: "shutdown", Type: "duration", Description: "API server graceful shutdown timeout"},
	}
}

// AvailableKeys implements ConfigSchema for CORSConfigSection.
func (c *CORSConfigSection) AvailableKeys() []ConfigKey {
	return []ConfigKey{
		{Path: "enable", Type: "bool", Description: "Enable CORS support"},
		{
			Path:        "allow_origins",
			Type:        "[]string",
			Description: "Allowed CORS origins (e.g., [\"http://localhost:3000\"])",
		},
		{Path: "allow_methods", Type: "[]string", Description: "Allowed HTTP methods (e.g., [\"GET\", \"POST\"])"},
		{Path: "allow_headers", Type: "[]string", Description: "Allowed request headers"},
		{Path: "expose_headers", Type: "[]string", Description: "Headers to expose to the client"},
		{Path: "allow_credentials", Type: "bool", Description: "Allow credentials in CORS requests"},
		{Path: "max_age", Type: "duration", Description: "Max age for preflight requests"},
	}
}

// AvailableKeys implements ConfigSchema for MCPIntervalConfigSection.
func (m *MCPIntervalConfigSection) AvailableKeys() []ConfigKey {
	return []ConfigKey{
		{Path: "health", Type: "duration", Description: "MCP health check interval"},
	}
}

// AvailableKeys implements ConfigSchema for MCPTimeoutConfigSection.
func (m *MCPTimeoutConfigSection) AvailableKeys() []ConfigKey {
	return []ConfigKey{
		{Path: "shutdown", Type: "duration", Description: "MCP server shutdown timeout"},
		{Path: "init", Type: "duration", Description: "MCP server initialization timeout"},
		{Path: "health", Type: "duration", Description: "MCP health check timeout"},
	}
}

// AvailableKeys implements ConfigSchema for APIConfigSection.
func (a *APIConfigSection) AvailableKeys() []ConfigKey {
	var keys []ConfigKey

	// Direct API keys
	keys = append(
		keys,
		ConfigKey{Path: "addr", Type: "string", Description: "API server bind address (e.g., \"0.0.0.0:8090\")"},
	)

	// Recurse to subsections and prefix their results
	for _, key := range (&APITimeoutConfigSection{}).AvailableKeys() {
		key.Path = "timeout." + key.Path
		keys = append(keys, key)
	}

	for _, key := range (&CORSConfigSection{}).AvailableKeys() {
		key.Path = "cors." + key.Path
		keys = append(keys, key)
	}

	return keys
}

// AvailableKeys implements ConfigSchema for MCPConfigSection.
func (m *MCPConfigSection) AvailableKeys() []ConfigKey {
	var keys []ConfigKey

	// Recurse to subsections and prefix their results
	for _, key := range (&MCPTimeoutConfigSection{}).AvailableKeys() {
		key.Path = "timeout." + key.Path
		keys = append(keys, key)
	}

	for _, key := range (&MCPIntervalConfigSection{}).AvailableKeys() {
		key.Path = "interval." + key.Path
		keys = append(keys, key)
	}

	return keys
}

// AvailableKeys implements ConfigSchema for DaemonConfig.
func (d *DaemonConfig) AvailableKeys() []ConfigKey {
	var keys []ConfigKey

	// Recurse to subsections and prefix their results
	for _, key := range (&APIConfigSection{}).AvailableKeys() {
		key.Path = "api." + key.Path
		keys = append(keys, key)
	}

	for _, key := range (&MCPConfigSection{}).AvailableKeys() {
		key.Path = "mcp." + key.Path
		keys = append(keys, key)
	}

	return keys
}

// determineBoolPtrResult compares old and new bool pointer values to determine the operation result.
func determineBoolPtrResult(old *bool, new *bool) context.UpsertResult {
	if old == nil && new == nil {
		return context.Noop
	}
	if old == nil {
		return context.Created
	}
	if new == nil {
		return context.Deleted
	}
	if *old == *new {
		return context.Noop
	}
	return context.Updated
}

// determineDurationPtrResult compares old and new Duration pointer values to determine the operation result.
func determineDurationPtrResult(old *Duration, new *Duration) context.UpsertResult {
	if old == nil && new == nil {
		return context.Noop
	}
	if old == nil {
		return context.Created
	}
	if new == nil {
		return context.Deleted
	}
	if *old == *new {
		return context.Noop
	}
	return context.Updated
}

// determineStringPtrResult compares old and new string pointer values to determine the operation result.
func determineStringPtrResult(old *string, new *string) context.UpsertResult {
	if old == nil && new == nil {
		return context.Noop
	}
	if old == nil {
		return context.Created
	}
	if new == nil {
		return context.Deleted
	}
	if *old == *new {
		return context.Noop
	}
	return context.Updated
}

// determineStringSliceResult compares old and new string slice values to determine the operation result.
func determineStringSliceResult(old []string, new []string) context.UpsertResult {
	if len(old) == 0 && len(new) == 0 {
		return context.Noop
	}
	if len(old) == 0 {
		return context.Created
	}
	if len(new) == 0 {
		return context.Deleted
	}
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

// parseBool parses a string value into a boolean using common representations.
func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "t", "1", "yes", "y":
		return true, nil
	case "false", "f", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

// parseDuration parses a string value into a Duration.
func parseDuration(value string) (Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return Duration(0), err
	}
	return Duration(duration), nil
}

// parseStringArray parses a comma-separated string into a slice of strings.
func parseStringArray(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}
	return result
}

// Example .mcpd.toml with daemon configuration:
//
//[[servers]]
//  name = "github"
//  package = "uvx::github-mcp-server@1.0.0"
//  tools = ["create_issue", "list_issues"]
//  required_env = ["GITHUB_TOKEN"]
//
//[daemon]
//  [daemon.api]
//    addr = "0.0.0.0:9876"
//
//    [daemon.api.timeout]
//      shutdown = "20s"
//
//    [daemon.api.cors]
//      enable = true
//      allow_origins = ["http://localhost:3000", "https://app.example.com"]
//      allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
//      credentials = false
//      max_age = "5m"
//
//  [daemon.mcp]
//    [daemon.mcp.timeout]
//      shutdown = "20s"
//      init = "30s"
//      health = "5s"
//
//    [daemon.mcp.interval]
//      health = "10s"
