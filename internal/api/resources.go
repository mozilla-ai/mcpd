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

// DomainResource wraps mcp.Resource for API conversion.
type DomainResource mcp.Resource

// DomainResourceTemplate wraps mcp.ResourceTemplate for API conversion.
type DomainResourceTemplate mcp.ResourceTemplate

// Resources represents a collection of Resource types.
type Resources struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// Resource represents a known resource.
type Resource struct {
	// URI of this resource.
	URI string `json:"uri"`

	// Name is a human-readable name for this resource.
	Name string `json:"name"`

	// Description of what this resource represents.
	Description string `json:"description,omitempty"`

	// MIMEType of this resource, if known.
	MIMEType string `json:"mimeType,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta Meta `json:"_meta,omitempty"` //nolint:tagliatelle
}

// ResourceTemplates represents a collection of ResourceTemplate types.
type ResourceTemplates struct {
	Templates  []ResourceTemplate `json:"templates"`
	NextCursor string             `json:"nextCursor,omitempty"`
}

// ResourceTemplate represents a resource template.
type ResourceTemplate struct {
	// URITemplate is a URI template (RFC 6570) for constructing resource URIs.
	URITemplate string `json:"uriTemplate"`

	// Name is a human-readable name for the type of resource this template refers to.
	Name string `json:"name"`

	// Description of what this template is for.
	Description string `json:"description,omitempty"`

	// MIMEType for all resources that match this template.
	MIMEType string `json:"mimeType,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta Meta `json:"_meta,omitempty"` //nolint:tagliatelle
}

// ResourceContent represents the content of a resource.
type ResourceContent struct {
	// URI of this resource.
	URI string `json:"uri"`

	// MIMEType of this resource, if known.
	MIMEType string `json:"mimeType,omitempty"`

	// Text content (for text resources).
	Text string `json:"text,omitempty"`

	// Blob content (base64 encoded binary data).
	Blob string `json:"blob,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta Meta `json:"_meta,omitempty"` //nolint:tagliatelle
}

// ServerResourcesRequest represents the incoming API request for listing resources.
type ServerResourcesRequest struct {
	Name   string `doc:"Name of the server" path:"name"`
	Cursor string `doc:"Pagination cursor"              query:"cursor"`
}

// ServerResourceTemplatesRequest represents the incoming API request for listing resource templates.
type ServerResourceTemplatesRequest struct {
	Name   string `doc:"Name of the server" path:"name"`
	Cursor string `doc:"Pagination cursor"              query:"cursor"`
}

// ServerResourceContentRequest represents the incoming API request for getting resource content.
type ServerResourceContentRequest struct {
	Name string `doc:"Name of the server"  path:"name"`
	URI  string `doc:"URI of the resource"             query:"uri"`
}

// ResourcesResponse represents the wrapped API response for Resources.
type ResourcesResponse struct {
	Body Resources
}

// ResourceTemplatesResponse represents the wrapped API response for ResourceTemplates.
type ResourceTemplatesResponse struct {
	Body ResourceTemplates
}

// ResourceContentResponse represents the wrapped API response for getting resource content.
type ResourceContentResponse struct {
	Body []ResourceContent
}

// ToAPIType converts a domain resource to an API resource.
func (d DomainResource) ToAPIType() (Resource, error) {
	var meta Meta
	if d.Meta != nil {
		var err error
		meta, err = DomainMeta(*d.Meta).ToAPIType()
		if err != nil {
			return Resource{}, err
		}
	}

	return Resource{
		URI:         d.URI,
		Name:        d.Name,
		Description: d.Description,
		MIMEType:    d.MIMEType,
		Meta:        meta,
	}, nil
}

// ToAPIType converts a domain resource template to an API resource template.
func (d DomainResourceTemplate) ToAPIType() (ResourceTemplate, error) {
	uriTemplate := ""
	if d.URITemplate != nil {
		uriTemplate = d.URITemplate.Raw()
	}

	var meta Meta
	if d.Meta != nil {
		var err error
		meta, err = DomainMeta(*d.Meta).ToAPIType()
		if err != nil {
			return ResourceTemplate{}, err
		}
	}

	return ResourceTemplate{
		URITemplate: uriTemplate,
		Name:        d.Name,
		Description: d.Description,
		MIMEType:    d.MIMEType,
		Meta:        meta,
	}, nil
}

// handleServerResources returns the list of resources for a given server.
func handleServerResources(
	accessor contracts.MCPClientAccessor,
	name string,
	cursor string,
) (*ResourcesResponse, error) {
	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := mcp.ListResourcesRequest{}
	if cursor != "" {
		req.Params = mcp.PaginatedParams{
			Cursor: mcp.Cursor(cursor),
		}
	}

	result, err := mcpClient.ListResources(ctx, req)
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrResourcesNotImplemented, name)
		}
		return nil, fmt.Errorf("%w: %s: %w", errors.ErrResourceListFailed, name, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: no result", errors.ErrResourceListFailed, name)
	}

	resources := make([]Resource, 0, len(result.Resources))
	for _, res := range result.Resources {
		apiRes, err := DomainResource(res).ToAPIType()
		if err != nil {
			return nil, err
		}
		resources = append(resources, apiRes)
	}

	resp := &ResourcesResponse{}
	resp.Body = Resources{
		Resources:  resources,
		NextCursor: string(result.NextCursor),
	}

	return resp, nil
}

// handleServerResourceTemplates returns the list of resource templates for a given server.
func handleServerResourceTemplates(
	accessor contracts.MCPClientAccessor,
	name string,
	cursor string,
) (*ResourceTemplatesResponse, error) {
	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := mcp.ListResourceTemplatesRequest{}
	if cursor != "" {
		req.Params = mcp.PaginatedParams{
			Cursor: mcp.Cursor(cursor),
		}
	}

	result, err := mcpClient.ListResourceTemplates(ctx, req)
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrResourcesNotImplemented, name)
		}
		return nil, fmt.Errorf("%w: %s: %w", errors.ErrResourceTemplateListFailed, name, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: no result", errors.ErrResourceTemplateListFailed, name)
	}

	templates := make([]ResourceTemplate, 0, len(result.ResourceTemplates))
	for _, tmpl := range result.ResourceTemplates {
		apiTmpl, err := DomainResourceTemplate(tmpl).ToAPIType()
		if err != nil {
			return nil, err
		}
		templates = append(templates, apiTmpl)
	}

	resp := &ResourceTemplatesResponse{}
	resp.Body = ResourceTemplates{
		Templates:  templates,
		NextCursor: string(result.NextCursor),
	}

	return resp, nil
}

// handleServerResourceContent gets the content of a specific resource from a server.
func handleServerResourceContent(
	accessor contracts.MCPClientAccessor,
	name string,
	uri string,
) (*ResourceContentResponse, error) {
	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mcpClient.ReadResource(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	})
	if err != nil {
		// TODO: This string matching is fragile and should be replaced with proper JSON-RPC error code checking.
		// Once mcp-go preserves JSON-RPC error codes, use errors.Is(err, mcp.ErrMethodNotFound) instead.
		// See: https://github.com/mark3labs/mcp-go/issues/593
		if strings.Contains(err.Error(), methodNotFoundMessage) {
			return nil, fmt.Errorf("%w: %s", errors.ErrResourcesNotImplemented, name)
		}
		return nil, fmt.Errorf("%w: %s: %s: %w", errors.ErrResourceReadFailed, name, uri, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: %s: no result", errors.ErrResourceReadFailed, name, uri)
	}

	contents := make([]ResourceContent, 0, len(result.Contents))
	for _, content := range result.Contents {
		switch c := content.(type) {
		case mcp.TextResourceContents:
			var meta Meta
			if c.Meta != nil {
				var err error
				meta, err = DomainMeta(*c.Meta).ToAPIType()
				if err != nil {
					return nil, err
				}
			}
			contents = append(contents, ResourceContent{
				URI:      c.URI,
				MIMEType: c.MIMEType,
				Text:     c.Text,
				Meta:     meta,
			})
		case mcp.BlobResourceContents:
			var meta Meta
			if c.Meta != nil {
				var err error
				meta, err = DomainMeta(*c.Meta).ToAPIType()
				if err != nil {
					return nil, err
				}
			}
			contents = append(contents, ResourceContent{
				URI:      c.URI,
				MIMEType: c.MIMEType,
				Blob:     c.Blob,
				Meta:     meta,
			})
		}
	}

	resp := &ResourceContentResponse{}
	resp.Body = contents

	return resp, nil
}

// RegisterResourceRoutes registers resource-related routes under the servers API.
func RegisterResourceRoutes(parentAPI huma.API, accessor contracts.MCPClientAccessor) {
	tags := []string{"Resources"}

	huma.Register(
		parentAPI,
		huma.Operation{
			OperationID: "listResources",
			Method:      "GET",
			Path:        "/{name}/resources",
			Summary:     "List server resources",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerResourcesRequest) (*ResourcesResponse, error) {
			return handleServerResources(accessor, input.Name, input.Cursor)
		},
	)

	huma.Register(
		parentAPI,
		huma.Operation{
			OperationID: "listResourceTemplates",
			Method:      "GET",
			Path:        "/{name}/resources/templates",
			Summary:     "List server resource templates",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerResourceTemplatesRequest) (*ResourceTemplatesResponse, error) {
			return handleServerResourceTemplates(accessor, input.Name, input.Cursor)
		},
	)

	huma.Register(
		parentAPI,
		huma.Operation{
			OperationID: "getResourceContent",
			Method:      "GET",
			Path:        "/{name}/resources/content",
			Summary:     "Get resource content from a server",
			Description: "Retrieves the content of a resource by URI",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerResourceContentRequest) (*ResourceContentResponse, error) {
			return handleServerResourceContent(accessor, input.Name, input.URI)
		},
	)
}
