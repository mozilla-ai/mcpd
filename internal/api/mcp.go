package api

import "github.com/mark3labs/mcp-go/mcp"

// methodNotFoundMessage is the error message returned by MCP servers when a method is not implemented.
// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
// See: https://github.com/mark3labs/mcp-go/issues/593
const methodNotFoundMessage = "Method not found"

// DomainMeta wraps mcp.Meta for API conversion.
type DomainMeta mcp.Meta

// Meta represents metadata in API responses.
type Meta map[string]any

// ToAPIType converts a domain meta to an API meta type.
// This creates a flat _meta object structure as defined by the MCP specification.
// Returns empty Meta{} if domain type is nil.
// See: https://modelcontextprotocol.io/specification/2025-06-18/basic/index#meta
func (d DomainMeta) ToAPIType() (Meta, error) {
	if (*mcp.Meta)(&d) == nil {
		return Meta{}, nil
	}

	// The _meta field is MCP's reserved extensibility mechanism that allows both:
	// - progressToken: for out-of-band progress notifications (defined by spec)
	// - Additional fields: custom metadata from servers/clients (extensible)
	// Both types of fields are merged at the same level in the resulting map.
	result := make(Meta)

	// Add progressToken if present (using MCP spec-defined field name).
	if d.ProgressToken != nil {
		result["progressToken"] = d.ProgressToken
	}

	// Merge additional fields at the same level.
	for k, v := range d.AdditionalFields {
		result[k] = v
	}

	return result, nil
}
