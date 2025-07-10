package api

import "github.com/mark3labs/mcp-go/mcp"

type DomainTool mcp.Tool

// Tools represents a collection of Tool.
type Tools struct {
	Tools []Tool `json:"tools"`
}

// ToolsResponse represents the wrapped API response for Tools.
type ToolsResponse struct {
	Body Tools
}

// ToolCallResponse represents the wrapped API response for calling a tool.
type ToolCallResponse struct {
	Body string
}

// Tool represents a callable tool, following the MCP spec.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema ToolInputSchema `json:"inputSchema"`
	Annotations ToolAnnotation  `json:"annotations"`
}

// ToolResponse represents the wrapped API response for a Tool.
type ToolResponse struct {
	Body Tool
}

// ToolInputSchema defines input params using JSON Schema
type ToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// ToolAnnotation defines behavioral hints for a tool
type ToolAnnotation struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    *bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
}

// ToAPIType can be used to convert a wrapped domain type to an API-safe type.
func (d DomainTool) ToAPIType() (Tool, error) {
	return Tool{
		Name:        d.Name,
		Description: d.Description,
		InputSchema: ToolInputSchema(d.InputSchema),
		Annotations: ToolAnnotation(d.Annotations),
	}, nil
}
