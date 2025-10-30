package api

import (
	"maps"

	"github.com/mark3labs/mcp-go/mcp"
)

// DomainMeta wraps mcp.Meta for API conversion.
type DomainMeta mcp.Meta

// Meta represents metadata in API responses.
type Meta map[string]any

// ToAPIType converts a domain meta to an API meta type.
// This creates a flat _meta object structure as defined by the MCP specification.
// Returns empty Meta{} if domain type is nil.
// See: https://modelcontextprotocol.io/specification/2025-06-18/basic/index#meta
func (d DomainMeta) ToAPIType() (Meta, error) {
	m := (*mcp.Meta)(&d)
	if m == nil || m.AdditionalFields == nil {
		return Meta{}, nil
	}

	return maps.Clone(m.AdditionalFields), nil
}
