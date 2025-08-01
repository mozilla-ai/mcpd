package packages

import "strings"

// Tools is a wrapper for a collection of `Tool`s.
// This allows for receivers to be declared that operate on the collection.
type Tools []Tool

// Tool defines the structure for a tool that a client can call.
type Tool struct {
	// Name of the tool.
	// Display name precedence order for a tool is: title, annotations.title, then name.
	Name string `json:"name"`

	// Title is a human-readable and easily understood title for the tool.
	Title string `json:"title,omitempty"`

	// Description is a human-readable description of the tool.
	// This can be used by clients to improve the LLM's understanding of available tools.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`

	// InputSchema is JSONSchema defining the expected parameters for the tool.
	InputSchema JSONSchema `json:"inputSchema"`

	// OutputSchema is an optional JSONSchema defining the structure of the tool's
	// output returned in the structured content field of a tool call result.
	OutputSchema *JSONSchema `json:"outputSchema,omitempty"`

	// Annotations provide optional additional tool information.
	// Display name precedence order is: title, annotations.title when present, then tool name.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata to their interactions.
	// See https://modelcontextprotocol.io/specification/2025-06-18/basic#general-fields for notes on _meta usage.
	Meta map[string]any `json:"_meta,omitempty"`
}

// JSONSchema defines the structure for a JSON schema object.
type JSONSchema struct {
	// Type defines the type for this schema, e.g. "object".
	Type string `json:"type"`

	// Properties represents a property name and associated object definition.
	Properties map[string]any `json:"properties,omitempty"`

	// Required lists the (keys of) Properties that are required.
	Required []string `json:"required,omitempty"`
}

// ToolAnnotations provides additional properties describing a Tool to clients.
// NOTE: all properties in ToolAnnotations are **hints**.
// They are not guaranteed to provide a faithful description of tool behavior
// (including descriptive properties like `title`).
// Clients should never make tool use decisions based on ToolAnnotations received from untrusted servers.
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title *string `json:"title,omitempty"`

	// ReadOnlyHint if true, the tool should not modify its environment.
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`

	// DestructiveHint if true, the tool may perform destructive updates to its environment.
	// If false, the tool performs only additive updates.
	// (This property is meaningful only when ReadOnlyHint is false)
	DestructiveHint *bool `json:"destructiveHint,omitempty"`

	// IdempotentHint if true, calling the tool repeatedly with the same arguments
	// will have no additional effect on its environment.
	// (This property is meaningful only when ReadOnlyHint is false)
	IdempotentHint *bool `json:"idempotentHint,omitempty"`

	// OpenWorldHint if true, this tool may interact with an "open world" of external
	// entities. If false, the tool's domain of interaction is closed.
	// For example, the world of a web search tool is open, whereas that
	// of a memory tool is not.
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

// Names returns the names of all tools in the Tools collection.
func (t Tools) Names() []string {
	names := make([]string, len(t))
	for i, tool := range t {
		names[i] = strings.TrimSpace(tool.Name)
	}
	return names
}
