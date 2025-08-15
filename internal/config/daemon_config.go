package config

import (
	"fmt"
	"time"
)

type Duration time.Duration

// DaemonConfig represents daemon-specific configuration that can be stored in .mcpd.toml.
// This extends the existing Config struct with daemon settings.
type DaemonConfig struct {
	// API configuration (includes address and nested timeout/cors)
	API *APIConfigSection `json:"api,omitempty" toml:"api,omitempty" yaml:"api,omitempty"`

	// MCP configuration (includes nested timeout and interval settings)
	MCP *MCPConfigSection `json:"mcp,omitempty" toml:"mcp,omitempty" yaml:"mcp,omitempty"`
}

// APIConfigSection contains API server configuration settings.
type APIConfigSection struct {
	// Address to bind the API server (e.g., "0.0.0.0:8090")
	// Maps to CLI flag --addr
	Addr *string `json:"addr,omitempty" toml:"addr,omitempty" yaml:"addr,omitempty"`

	// Nested timeout configuration for API operations
	Timeout *APITimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// CORS configuration for API server
	CORS *CORSConfigSection `json:"cors,omitempty" toml:"cors,omitempty" yaml:"cors,omitempty"`
}

// CORSConfigSection contains CORS configuration.
// Maps to CLI flags --cors-*
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

	Headers []string `json:"allowHeaders,omitempty" toml:"allow_headers,omitempty" yaml:"allow_headers,omitempty"`

	ExposeHeaders []string `json:"exposeHeaders,omitempty" toml:"expose_headers,omitempty" yaml:"expose_headers,omitempty"`

	// Allow credentials
	// Maps to CLI flag --cors-credentials
	Credentials *bool `json:"allowCredentials" toml:"allow_credentials" yaml:"allow_credentials"`

	// Max age for preflight requests (in seconds)
	// Maps to CLI flag --cors-max-age
	MaxAge *Duration `json:"maxAge,omitempty" toml:"max_age,omitempty" yaml:"max_age,omitempty"`
}

// APITimeoutConfigSection contains API-specific timeout configuration.
type APITimeoutConfigSection struct {
	// API shutdown timeout for graceful shutdown
	// Maps to CLI flag --timeout-api-shutdown
	Shutdown *Duration `json:"shutdown,omitempty" toml:"shutdown,omitempty" yaml:"shutdown,omitempty"`
}

// MCPConfigSection contains MCP server configuration settings.
type MCPConfigSection struct {
	// Nested timeout configuration for MCP operations
	Timeout *MCPTimeoutConfigSection `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Nested interval configuration for MCP operations
	Interval *MCPIntervalConfigSection `json:"interval,omitempty" toml:"interval,omitempty" yaml:"interval,omitempty"`
}

// MCPTimeoutConfigSection contains MCP-specific timeout configuration.
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

// MCPIntervalConfigSection contains MCP-specific interval configuration.
type MCPIntervalConfigSection struct {
	// MCP health check interval
	// Maps to CLI flag --interval-mcp-health
	Health *Duration `json:"health,omitempty" toml:"health,omitempty" yaml:"health,omitempty"`
}

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d *Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(*d).String()), nil
}

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

func (c *CORSConfigSection) EnableOrDefault(defaultEnable bool) bool {
	if c == nil || c.Enable == nil {
		return defaultEnable
	}
	return *c.Enable
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
