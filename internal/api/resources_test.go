package api

import (
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/mozilla-ai/mcpd/v2/internal/errors"
)

func TestAPI_HandleServerResources_Success(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		serverName     string
		resources      []mcp.Resource
		nextCursor     mcp.Cursor
		expectedCount  int
		expectedCursor string
	}{
		{
			name:       "single resource",
			serverName: "test-server",
			resources: []mcp.Resource{
				{
					URI:         "file:///test.txt",
					Name:        "test file",
					Description: "A test file",
					MIMEType:    "text/plain",
				},
			},
			expectedCount:  1,
			expectedCursor: "",
		},
		{
			name:       "multiple resources with cursor",
			serverName: "test-server",
			resources: []mcp.Resource{
				{
					URI:  "file:///test1.txt",
					Name: "test file 1",
				},
				{
					URI:  "file:///test2.txt",
					Name: "test file 2",
				},
			},
			nextCursor:     "next-page-token",
			expectedCount:  2,
			expectedCursor: "next-page-token",
		},
		{
			name:          "empty resources list",
			serverName:    "empty-server",
			resources:     []mcp.Resource{},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockMCPClient{
				listResourcesResult: mcp.NewListResourcesResult(tc.resources, tc.nextCursor),
			}

			accessor := newMockMCPClientAccessor()
			accessor.Add(tc.serverName, mockClient, []string{})

			result, err := handleServerResources(accessor, tc.serverName, "")

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Body.Resources, tc.expectedCount)
			require.Equal(t, tc.expectedCursor, result.Body.NextCursor)

			if len(tc.resources) > 0 {
				require.Equal(t, tc.resources[0].URI, result.Body.Resources[0].URI)
				require.Equal(t, tc.resources[0].Name, result.Body.Resources[0].Name)
			}
		})
	}
}

func TestAPI_HandleServerResources_WithCursor(t *testing.T) {
	t.Parallel()

	cursor := "test-cursor"
	mockClient := &mockMCPClient{
		listResourcesResult: mcp.NewListResourcesResult([]mcp.Resource{
			{URI: "file:///page2.txt", Name: "page 2"},
		}, ""),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerResources(accessor, "test-server", cursor)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body.Resources, 1)
}

func TestAPI_HandleServerResources_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := handleServerResources(accessor, "nonexistent-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrServerNotFound))
}

func TestAPI_HandleServerResources_MCPError(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		listResourcesError: errors.New("MCP communication error"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerResources(accessor, "test-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrResourceListFailed))
}

func TestAPI_HandleServerResources_NilResult(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		listResourcesResult: nil,
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerResources(accessor, "test-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrResourceListFailed))
}

func TestAPI_HandleServerResourceTemplates_Success(t *testing.T) {
	t.Parallel()

	templates := []mcp.ResourceTemplate{
		{
			Name:        "file template",
			Description: "A file template",
			MIMEType:    "text/plain",
		},
		{
			Name:        "api template",
			Description: "An API template",
		},
	}

	mockClient := &mockMCPClient{
		listTemplatesResult: mcp.NewListResourceTemplatesResult(templates, "template-cursor"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerResourceTemplates(accessor, "test-server", "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body.Templates, 2)
	require.Equal(t, "template-cursor", result.Body.NextCursor)
	require.Equal(t, "file template", result.Body.Templates[0].Name)
	require.Equal(t, "api template", result.Body.Templates[1].Name)
}

func TestAPI_HandleServerResourceTemplates_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := handleServerResourceTemplates(accessor, "nonexistent-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrServerNotFound))
}

func TestAPI_HandleServerResourceContent_TextContent(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		readResourceResult: &mcp.ReadResourceResult{
			Contents: []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "file:///test.txt",
					MIMEType: "text/plain",
					Text:     "Hello, world!",
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	uri := "file:///test.txt"

	result, err := handleServerResourceContent(accessor, "test-server", uri)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body, 1)
	require.Equal(t, "file:///test.txt", result.Body[0].URI)
	require.Equal(t, "Hello, world!", result.Body[0].Text)
	require.Empty(t, result.Body[0].Blob)
}

func TestAPI_HandleServerResourceContent_BlobContent(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		readResourceResult: &mcp.ReadResourceResult{
			Contents: []mcp.ResourceContents{
				mcp.BlobResourceContents{
					URI:      "file:///image.png",
					MIMEType: "image/png",
					Blob:     "iVBORw0KGgo=",
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	uri := "file:///image.png"

	result, err := handleServerResourceContent(accessor, "test-server", uri)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body, 1)
	require.Equal(t, "file:///image.png", result.Body[0].URI)
	require.Equal(t, "iVBORw0KGgo=", result.Body[0].Blob)
	require.Empty(t, result.Body[0].Text)
}

func TestAPI_HandleServerResourceContent_MultipleContents(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		readResourceResult: &mcp.ReadResourceResult{
			Contents: []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "file:///multi.txt",
					MIMEType: "text/plain",
					Text:     "Text part",
				},
				mcp.BlobResourceContents{
					URI:      "file:///multi.bin",
					MIMEType: "application/octet-stream",
					Blob:     "YmluYXJ5",
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	uri := "file:///multi"

	result, err := handleServerResourceContent(accessor, "test-server", uri)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body, 2)
	require.Equal(t, "Text part", result.Body[0].Text)
	require.Equal(t, "YmluYXJ5", result.Body[1].Blob)
}

func TestAPI_HandleServerResourceContent_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	uri := "file:///test.txt"

	result, err := handleServerResourceContent(accessor, "nonexistent-server", uri)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrServerNotFound))
}

func TestAPI_HandleServerResourceContent_ReadError(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		readResourceError: errors.New("resource read failed"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	uri := "file:///nonexistent.txt"

	result, err := handleServerResourceContent(accessor, "test-server", uri)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrResourceReadFailed))
}

func TestAPI_HandleServerResourceContent_NilResult(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		readResourceResult: nil,
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	uri := "file:///test.txt"

	result, err := handleServerResourceContent(accessor, "test-server", uri)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrResourceReadFailed))
}
