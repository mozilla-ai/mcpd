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

// DomainPrompt wraps mcp.Prompt for API conversion.
type DomainPrompt mcp.Prompt

// DomainPromptArgument wraps mcp.PromptArgument for API conversion.
type DomainPromptArgument mcp.PromptArgument

// DomainPromptMessage wraps mcp.PromptMessage for API conversion.
type DomainPromptMessage mcp.PromptMessage

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

// ServerPromptsListRequest represents the incoming API request for listing prompts.
type ServerPromptsListRequest struct {
	Name   string `doc:"Name of the server" path:"name"`
	Cursor string `doc:"Pagination cursor"              query:"cursor"`
}

// ServerPromptGenerateRequest represents the incoming API request for generating a prompt.
type ServerPromptGenerateRequest struct {
	ServerName string                  `doc:"Name of the server" path:"name"`
	PromptName string                  `doc:"Name of the prompt" path:"promptName"`
	Body       PromptGenerateArguments `doc:"Prompt arguments"`
}

// PromptGenerateArguments contains arguments for generating a prompt from a template.
type PromptGenerateArguments struct {
	Arguments map[string]string `doc:"Arguments for templating the prompt" json:"arguments,omitempty"`
}

// PromptsListResponse represents the wrapped API response for listing Prompts.
type PromptsListResponse struct {
	Body Prompts
}

// GeneratePromptResponse represents the API response for generating a prompt from a template.
type GeneratePromptResponse struct {
	Body struct {
		// Description for the prompt.
		Description string `json:"description,omitempty"`

		// Messages that make up the prompt.
		Messages []PromptMessage `json:"messages"`
	}
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
) (*PromptsListResponse, error) {
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

	resp := &PromptsListResponse{}
	resp.Body = Prompts{
		Prompts:    prompts,
		NextCursor: string(result.NextCursor),
	}

	return resp, nil
}

// handleServerPromptGenerate generates a prompt from a template on a server.
func handleServerPromptGenerate(
	accessor contracts.MCPClientAccessor,
	serverName string,
	promptName string,
	arguments map[string]string,
) (*GeneratePromptResponse, error) {
	mcpClient, clientOk := accessor.Client(serverName)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, serverName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mcpClient.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      promptName,
			Arguments: arguments,
		},
	})
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrPromptsNotImplemented, serverName)
		}
		return nil, fmt.Errorf("%w: %s: %s: %w", errors.ErrPromptGenerationFailed, serverName, promptName, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: %s: no result", errors.ErrPromptGenerationFailed, serverName, promptName)
	}

	messages := make([]PromptMessage, 0, len(result.Messages))
	for _, message := range result.Messages {
		apiMessage, err := DomainPromptMessage(message).ToAPIType()
		if err != nil {
			return nil, err
		}
		messages = append(messages, apiMessage)
	}

	resp := &GeneratePromptResponse{}
	resp.Body.Description = result.Description
	resp.Body.Messages = messages

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
			Summary:     "List server prompt templates",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerPromptsListRequest) (*PromptsListResponse, error) {
			return handleServerPrompts(accessor, input.Name, input.Cursor)
		},
	)

	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "generatePrompt",
			Method:      "POST",
			Path:        "/{name}/prompts/{promptName}",
			Summary:     "Generate a prompt from a server template",
			Description: "Generates a prompt by filling in a template with the provided arguments",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerPromptGenerateRequest) (*GeneratePromptResponse, error) {
			return handleServerPromptGenerate(accessor, input.ServerName, input.PromptName, input.Body.Arguments)
		},
	)
}
