package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mozilla-ai/mcpd/internal/contracts"
	"github.com/mozilla-ai/mcpd/internal/filter"
)

const (
	// queryParamDetail is the name of the query parameter for detail level selection.
	queryParamDetail = "detail"

	// toolDetailFull returns all fields including schemas and annotations.
	toolDetailFull toolDetailLevel = "full"

	// toolDetailMinimal returns only name and title.
	toolDetailMinimal toolDetailLevel = "minimal"

	// toolDetailSummary returns name, title, and description.
	toolDetailSummary toolDetailLevel = "summary"
)

// toolDetailLevel defines the amount of information to return about tools.
type toolDetailLevel string

// ToolCallResponse represents the wrapped API response for calling a tool.
type ToolCallResponse struct {
	Body string
}

// ToolView is a union constraint for all tool view types.
// This ensures type safety when using generic ToolsResponse.
type ToolView interface {
	ToolMinimal | ToolSummary | Tool
}

// ToolsResponseBody represents the body of a tools response.
type ToolsResponseBody[T ToolView] struct {
	Tools []T `json:"tools"`
}

// ToolsResponse represents a generic wrapped API response for tool collections.
// The type parameter T must be one of the ToolView types (ToolMinimal, ToolSummary, or Tool).
type ToolsResponse[T ToolView] struct {
	Body ToolsResponseBody[T]
}

// ToolMinimal represents minimal tool information with name and title only.
type ToolMinimal struct {
	// Name of the tool.
	Name string `doc:"Name of the tool" json:"name"`

	// Title is a human-readable and easily understood title for the tool.
	Title string `doc:"Human-readable title" json:"title,omitempty"`
}

// ToolSummary represents summary tool information including name, title, and description.
type ToolSummary struct {
	ToolMinimal

	// Description is a human-readable description of the tool.
	// This can be used by clients to improve the LLM's understanding of available tools.
	// It can be thought of like a "hint" to the model.
	Description string `doc:"Description of what the tool does" json:"description"`
}

// Tool represents complete tool information including all schemas and annotations.
// This embeds ToolSummary (which embeds ToolMinimal) providing a full tool definition.
type Tool struct {
	ToolSummary

	// InputSchema is JSONSchema defining the expected parameters for the tool.
	InputSchema *JSONSchema `doc:"Input parameters schema" json:"inputSchema,omitempty"`

	// OutputSchema is an optional JSONSchema defining the structure of the tool's
	// output returned in the structured content field of a tool call result.
	OutputSchema *JSONSchema `doc:"Output structure schema" json:"outputSchema,omitempty"`

	// Annotations provide optional additional tool information.
	// Display name precedence order is: title, annotations.title when present, then tool name.
	Annotations *ToolAnnotations `doc:"Additional hints about the tool" json:"annotations,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata to their interactions.
	// See https://modelcontextprotocol.io/specification/2025-06-18/basic#general-fields for notes on _meta usage.
	Meta map[string]any `doc:"Additional metadata" json:"_meta,omitempty"` //nolint:tagliatelle
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

// domainTool wraps mcp.Tool for conversion to Tool via ToAPIType.
type domainTool mcp.Tool

// domainToolMinimal wraps Tool for projection to ToolMinimal via ToAPIType.
type domainToolMinimal Tool

// domainToolSummary wraps Tool for projection to ToolSummary via ToAPIType.
type domainToolSummary Tool

// Normalize handles case-insensitivity and trimming, providing a safe default.
func (t toolDetailLevel) Normalize() toolDetailLevel {
	normalized := toolDetailLevel(strings.ToLower(strings.TrimSpace(string(t))))
	switch normalized {
	case toolDetailMinimal, toolDetailSummary, toolDetailFull:
		return normalized
	default:
		return toolDetailFull // Safe default.
	}
}

// ToAPIType converts a wrapped domain type to Tool.
func (d domainTool) ToAPIType() (Tool, error) {
	title := d.Annotations.Title

	inputSchema := &JSONSchema{
		Type:       d.InputSchema.Type,
		Properties: d.InputSchema.Properties,
		Required:   d.InputSchema.Required,
	}

	var outputSchema *JSONSchema
	if d.OutputSchema.Type != "" {
		outputSchema = &JSONSchema{
			Type:       d.OutputSchema.Type,
			Properties: d.OutputSchema.Properties,
			Required:   d.OutputSchema.Required,
		}
	}

	annotations := &ToolAnnotations{
		Title:           &d.Annotations.Title,
		ReadOnlyHint:    d.Annotations.ReadOnlyHint,
		DestructiveHint: d.Annotations.DestructiveHint,
		OpenWorldHint:   d.Annotations.OpenWorldHint,
		IdempotentHint:  d.Annotations.IdempotentHint,
	}

	// Nil the annotations if they're essentially zero value so they can be omitted in the result.
	if annotations.IsZero() {
		annotations = nil
	}

	// Extract Meta if present.
	var meta map[string]any
	if d.Meta != nil && d.Meta.AdditionalFields != nil {
		meta = d.Meta.AdditionalFields
	}

	return Tool{
		ToolSummary: ToolSummary{
			ToolMinimal: ToolMinimal{
				Name:  filter.NormalizeString(d.Name),
				Title: title,
			},
			Description: d.Description,
		},
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Annotations:  annotations,
		Meta:         meta,
	}, nil
}

// ToAPIType projects Tool to ToolMinimal.
func (t domainToolMinimal) ToAPIType() (ToolMinimal, error) {
	return ToolMinimal{
		Name:  t.Name,
		Title: t.Title,
	}, nil
}

// ToAPIType projects Tool to ToolSummary.
func (t domainToolSummary) ToAPIType() (ToolSummary, error) {
	minimal, err := domainToolMinimal(t).ToAPIType()
	if err != nil {
		return ToolSummary{}, err
	}

	return ToolSummary{
		ToolMinimal: minimal,
		Description: t.Description,
	}, nil
}

// IsZero reports whether the ToolAnnotations struct has no meaningful values set.
// This is useful to avoid emitting empty "annotations" objects in JSON output.
func (a *ToolAnnotations) IsZero() bool {
	if a == nil {
		return true
	}

	if a.Title != nil && *a.Title != "" {
		return false
	}

	if a.ReadOnlyHint != nil || a.DestructiveHint != nil || a.IdempotentHint != nil || a.OpenWorldHint != nil {
		return false
	}

	return true
}

func RegisterToolRoutes(parentAPI huma.API, accessor contracts.MCPClientAccessor) {
	tags := []string{"Tools"}

	huma.Register(
		parentAPI,
		huma.Operation{
			OperationID: "listTools",
			Method:      http.MethodGet,
			Path:        "/{name}/tools",
			Summary:     "List server tools",
			Description: "Returns tools with configurable detail level via ?detail= query parameter (minimal, summary, full)",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerToolsRequest) (*ToolsResponse[Tool], error) {
			return handleServerTools(accessor, input.Name)
		},
	)

	huma.Register(
		parentAPI,
		huma.Operation{
			OperationID: "callTool",
			Method:      http.MethodPost,
			Path:        "/{server}/tools/{tool}",
			Summary:     "Call a tool for a server",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerToolCallRequest) (*ToolCallResponse, error) {
			return handleServerToolCall(accessor, input.Server, input.Tool, input.Body)
		},
	)
}

// toolFieldSelectTransformer transforms tool responses based on the detail query parameter.
// It filters the response to return only the requested level of detail: minimal, summary, or full.
func toolFieldSelectTransformer(ctx huma.Context, _ string, v any) (any, error) {
	detailParam := ctx.Query(queryParamDetail)
	if detailParam == "" {
		detailParam = string(toolDetailFull)
	}

	detail := toolDetailLevel(detailParam).Normalize()
	if detail == toolDetailFull {
		return v, nil
	}

	// Handle ToolsResponseBody[Tool].
	// Huma passes the Body field to transformers, not the full response.
	body, ok := v.(ToolsResponseBody[Tool])
	if !ok {
		return v, nil // Not our type, pass through.
	}

	// Transform each tool based on detail level.
	switch detail {
	case toolDetailMinimal:
		minimal := make([]ToolMinimal, len(body.Tools))
		for i, tool := range body.Tools {
			m, err := domainToolMinimal(tool).ToAPIType()
			if err != nil {
				return nil, err
			}
			minimal[i] = m
		}
		return ToolsResponseBody[ToolMinimal]{Tools: minimal}, nil

	case toolDetailSummary:
		summary := make([]ToolSummary, len(body.Tools))
		for i, tool := range body.Tools {
			sum, err := domainToolSummary(tool).ToAPIType()
			if err != nil {
				return nil, err
			}
			summary[i] = sum
		}
		return ToolsResponseBody[ToolSummary]{Tools: summary}, nil

	default:
		// Shouldn't reach here due to Normalize(), but pass through as safety.
		return v, nil
	}
}
