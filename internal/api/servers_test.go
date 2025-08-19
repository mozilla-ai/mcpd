package api

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

// mockMCPClientAccessor implements the MCPClientAccessor interface for testing.
type mockMCPClientAccessor struct {
	clients map[string]client.MCPClient
	tools   map[string][]string
}

func newMockMCPClientAccessor() *mockMCPClientAccessor {
	return &mockMCPClientAccessor{
		clients: make(map[string]client.MCPClient),
		tools:   make(map[string][]string),
	}
}

func (m *mockMCPClientAccessor) Add(name string, c client.MCPClient, tools []string) {
	m.clients[name] = c
	m.tools[name] = tools
}

func (m *mockMCPClientAccessor) Client(name string) (client.MCPClient, bool) {
	c, ok := m.clients[name]
	return c, ok
}

func (m *mockMCPClientAccessor) Tools(name string) ([]string, bool) {
	tools, ok := m.tools[name]
	return tools, ok
}

func (m *mockMCPClientAccessor) List() []string {
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

func (m *mockMCPClientAccessor) Remove(name string) {
	delete(m.clients, name)
	delete(m.tools, name)
}

// mockMCPClient implements the client.MCPClient interface for testing.
type mockMCPClient struct {
	listToolsResult *mcp.ListToolsResult
	listToolsError  error
	callToolResult  *mcp.CallToolResult
	callToolError   error
}

func (m *mockMCPClient) Initialize(_ context.Context, _ mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Ping(_ context.Context) error {
	return nil
}

func (m *mockMCPClient) ListResourcesByPage(
	_ context.Context,
	_ mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResources(
	_ context.Context,
	_ mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResourceTemplatesByPage(
	_ context.Context,
	_ mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResourceTemplates(
	_ context.Context,
	_ mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ReadResource(
	_ context.Context,
	_ mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Subscribe(_ context.Context, _ mcp.SubscribeRequest) error {
	return nil
}

func (m *mockMCPClient) Unsubscribe(_ context.Context, _ mcp.UnsubscribeRequest) error {
	return nil
}

func (m *mockMCPClient) ListPromptsByPage(
	_ context.Context,
	_ mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListPrompts(
	_ context.Context,
	_ mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) GetPrompt(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListToolsByPage(
	_ context.Context,
	_ mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return m.listToolsResult, m.listToolsError
}

func (m *mockMCPClient) ListTools(_ context.Context, _ mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return m.listToolsResult, m.listToolsError
}

func (m *mockMCPClient) CallTool(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return m.callToolResult, m.callToolError
}

func (m *mockMCPClient) SetLevel(_ context.Context, _ mcp.SetLevelRequest) error {
	return nil
}

func (m *mockMCPClient) Complete(_ context.Context, _ mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Close() error {
	return nil
}

func (m *mockMCPClient) OnNotification(_ func(notification mcp.JSONRPCNotification)) {}

func TestHandleServerTools_CaseInsensitiveFiltering(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	// Mock client returns tools with mixed case.
	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				{Name: "GetTime", Description: "Gets current time"},
				{Name: "SET_ALARM", Description: "Sets an alarm"},
				{Name: "list_events", Description: "Lists events"},
			},
		},
	}

	// Server has allowed tools in mixed case, but they should be normalized for comparison.
	allowedTools := []string{"gettime", "set_alarm"}
	accessor.Add("testserver", mockClient, allowedTools)

	result, err := handleServerTools(accessor, "testserver")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return 2 tools that match (case-insensitive).
	assert.Len(t, result.Body.Tools, 2)

	toolNames := make([]string, len(result.Body.Tools))
	for i, tool := range result.Body.Tools {
		toolNames[i] = tool.Name
	}

	// Verify the correct tools are returned.
	assert.Contains(t, toolNames, "GetTime")
	assert.Contains(t, toolNames, "SET_ALARM")
	assert.NotContains(t, toolNames, "list_events")
}

func TestHandleServerToolCall_ToolNameNormalization(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	// Mock client that will be called.
	mockClient := &mockMCPClient{
		callToolResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Text: "Tool executed successfully"},
			},
		},
	}

	// Allowed tools are stored in normalized form.
	allowedTools := []string{"gettime", "setalarm"}
	accessor.Add("testserver", mockClient, allowedTools)

	// Call with mixed case tool name - should be normalized and match.
	result, err := handleServerToolCall(accessor, "testserver", " GetTime ", map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "Tool executed successfully", result.Body)
}

func TestHandleServerToolCall_ToolNotAllowed(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	mockClient := &mockMCPClient{}
	allowedTools := []string{"gettime"}
	accessor.Add("testserver", mockClient, allowedTools)

	// Try to call a tool that's not in the allowed list.
	result, err := handleServerToolCall(accessor, "testserver", "forbidden_tool", map[string]any{})
	require.Error(t, err)
	require.Nil(t, result)

	assert.ErrorIs(t, err, errors.ErrToolForbidden)
}

func TestHandleServerToolCall_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := handleServerToolCall(accessor, "nonexistent", "tool", map[string]any{})
	require.Error(t, err)
	require.Nil(t, result)

	assert.ErrorIs(t, err, errors.ErrServerNotFound)
}

func TestHandleServerTools_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := handleServerTools(accessor, "nonexistent")
	require.Error(t, err)
	require.Nil(t, result)

	assert.ErrorIs(t, err, errors.ErrServerNotFound)
}

func TestHandleServerTools_NoTools(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()
	mockClient := &mockMCPClient{}

	// Add server with no tools.
	accessor.Add("testserver", mockClient, []string{})

	result, err := handleServerTools(accessor, "testserver")
	require.Error(t, err)
	require.Nil(t, result)

	assert.ErrorIs(t, err, errors.ErrToolsNotFound)
}
