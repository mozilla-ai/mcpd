package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

// TODO: Remove this const once mcp-go preserves JSON-RPC error codes.
// See: https://github.com/mark3labs/mcp-go/issues/593
const methodNotFoundMessage = "Method not found"

// DomainPrompt wraps mcp.Prompt for API conversion.
type DomainPrompt mcp.Prompt

// DomainPromptArgument wraps mcp.PromptArgument for API conversion.
type DomainPromptArgument mcp.PromptArgument

// DomainPromptMessage wraps mcp.PromptMessage for API conversion.
type DomainPromptMessage mcp.PromptMessage

// DomainMeta wraps mcp.Meta for API conversion.
type DomainMeta mcp.Meta

// Meta represents metadata in API responses.
type Meta map[string]any

// Prompts represents a collection of Prompt types.
type Prompts struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// Prompt represents a prompt or prompt template that the server offers.
type Prompt struct {
	// Name of the prompt or prompt template.
	Name string `json:"name"`

	// Description of what this prompt provides.
	Description string `json:"description,omitempty"`

	// Arguments for templating the prompt.
	Arguments []PromptArgument `json:"arguments,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta Meta `json:"_meta,omitempty"` //nolint:tagliatelle
}

// PromptArgument describes an argument that a prompt template can accept.
type PromptArgument struct {
	// Name of the argument.
	Name string `json:"name"`

	// Description of the argument.
	Description string `json:"description,omitempty"`

	// Whether this argument must be provided.
	Required bool `json:"required,omitempty"`
}

// PromptMessage describes a message returned as part of a prompt.
type PromptMessage struct {
	// Role of the message sender.
	Role string `json:"role"`

	// Content can be text, image, audio, or embedded resource.
	Content interface{} `json:"content"`
}

// GetPromptResponse represents the result of getting a specific prompt.
type GetPromptResponse struct {
	// Description for the prompt.
	Description string `json:"description,omitempty"`

	// Messages that make up the prompt.
	Messages []PromptMessage `json:"messages"`
}

// ServerPromptsRequest represents the incoming API request for listing prompts.
type ServerPromptsRequest struct {
	Name   string `doc:"Name of the server" path:"name"`
	Cursor string `doc:"Pagination cursor"              query:"cursor"`
}

// ServerPromptGetRequest represents the incoming API request for getting a prompt.
type ServerPromptGetRequest struct {
	Name string        `doc:"Name of the server"    path:"name"`
	Body GetPromptBody `doc:"Prompt get parameters"`
}

// GetPromptBody contains parameters for getting a prompt.
type GetPromptBody struct {
	Name      string            `doc:"Name of the prompt to get"         json:"name"`
	Arguments map[string]string `doc:"Optional arguments for the prompt" json:"arguments,omitempty"`
}

// PromptsResponse represents the wrapped API response for Prompts.
type PromptsResponse struct {
	Body Prompts
}

// GetPromptResponseWrapper represents the wrapped API response for getting a prompt.
type GetPromptResponseWrapper struct {
	Body GetPromptResponse
}

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

// ToAPIType converts a domain prompt to an API prompt.
func (d DomainPrompt) ToAPIType() (Prompt, error) {
	var meta Meta
	if d.Meta != nil {
		var err error
		meta, err = DomainMeta(*d.Meta).ToAPIType()
		if err != nil {
			return Prompt{}, err
		}
	}

	arguments := make([]PromptArgument, 0, len(d.Arguments))
	for _, arg := range d.Arguments {
		apiArg, err := DomainPromptArgument(arg).ToAPIType()
		if err != nil {
			return Prompt{}, err
		}
		arguments = append(arguments, apiArg)
	}

	return Prompt{
		Name:        d.Name,
		Description: d.Description,
		Arguments:   arguments,
		Meta:        meta,
	}, nil
}

// ToAPIType converts a domain prompt argument to an API prompt argument.
func (d DomainPromptArgument) ToAPIType() (PromptArgument, error) {
	return PromptArgument(d), nil
}

// ToAPIType converts a domain prompt message to an API prompt message.
func (d DomainPromptMessage) ToAPIType() (PromptMessage, error) {
	return PromptMessage{
		Role:    string(d.Role),
		Content: d.Content,
	}, nil
}

// handleServerPrompts returns the list of prompts for a given server.
func handleServerPrompts(
	accessor contracts.MCPClientAccessor,
	name string,
	cursor string,
) (*PromptsResponse, error) {
	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := mcp.ListPromptsRequest{}
	if cursor != "" {
		req.Params = mcp.PaginatedParams{
			Cursor: mcp.Cursor(cursor),
		}
	}

	result, err := mcpClient.ListPrompts(ctx, req)
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrPromptsNotImplemented, name)
		}
		return nil, fmt.Errorf("%w: %s: %w", errors.ErrPromptListFailed, name, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: no result", errors.ErrPromptListFailed, name)
	}

	prompts := make([]Prompt, 0, len(result.Prompts))
	for _, prompt := range result.Prompts {
		apiPrompt, err := DomainPrompt(prompt).ToAPIType()
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, apiPrompt)
	}

	resp := &PromptsResponse{}
	resp.Body = Prompts{
		Prompts:    prompts,
		NextCursor: string(result.NextCursor),
	}

	return resp, nil
}

// handleServerPromptGet gets a specific prompt from a server.
func handleServerPromptGet(
	accessor contracts.MCPClientAccessor,
	name string,
	body GetPromptBody,
) (*GetPromptResponseWrapper, error) {
	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mcpClient.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      body.Name,
			Arguments: body.Arguments,
		},
	})
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrPromptsNotImplemented, name)
		}
		return nil, fmt.Errorf("%w: %s: %s: %w", errors.ErrPromptGetFailed, name, body.Name, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: %s: no result", errors.ErrPromptGetFailed, name, body.Name)
	}

	messages := make([]PromptMessage, 0, len(result.Messages))
	for _, message := range result.Messages {
		apiMessage, err := DomainPromptMessage(message).ToAPIType()
		if err != nil {
			return nil, err
		}
		messages = append(messages, apiMessage)
	}

	resp := &GetPromptResponseWrapper{}
	resp.Body = GetPromptResponse{
		Description: result.Description,
		Messages:    messages,
	}

	return resp, nil
}

// RegisterPromptRoutes registers prompt-related routes under the servers API.
func RegisterPromptRoutes(serversAPI huma.API, accessor contracts.MCPClientAccessor) {
	tags := []string{"Servers", "Prompts"}

	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "listPrompts",
			Method:      "GET",
			Path:        "/{name}/prompts",
			Summary:     "List server prompts",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerPromptsRequest) (*PromptsResponse, error) {
			return handleServerPrompts(accessor, input.Name, input.Cursor)
		},
	)

	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "getPrompt",
			Method:      "POST",
			Path:        "/{name}/prompts/get",
			Summary:     "Get a prompt from a server",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerPromptGetRequest) (*GetPromptResponseWrapper, error) {
			return handleServerPromptGet(accessor, input.Name, input.Body)
		},
	)
}
