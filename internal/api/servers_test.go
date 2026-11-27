package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/errors"
)

// mockMCPClientAccessor implements the MCPClientAccessor interface for testing.
// mockMCPClientAccessor indexes by each name's key, matching the real accessor, so a test exercises the
// same case-insensitive lookup the API relies on.
type mockMCPClientAccessor struct {
	clients map[string]client.MCPClient
	names   map[string]domain.ServerName
	tools   map[string][]domain.ToolName
}

func newMockMCPClientAccessor() *mockMCPClientAccessor {
	return &mockMCPClientAccessor{
		clients: make(map[string]client.MCPClient),
		names:   make(map[string]domain.ServerName),
		tools:   make(map[string][]domain.ToolName),
	}
}

func (m *mockMCPClientAccessor) Add(name domain.ServerName, c client.MCPClient, tools []domain.ToolName) {
	m.clients[name.Key()] = c
	m.names[name.Key()] = name
	m.tools[name.Key()] = tools
}

func (m *mockMCPClientAccessor) Client(name domain.ServerName) (client.MCPClient, bool) {
	c, ok := m.clients[name.Key()]

	return c, ok
}

func (m *mockMCPClientAccessor) Tools(name domain.ServerName) ([]domain.ToolName, bool) {
	tools, ok := m.tools[name.Key()]

	return tools, ok
}

func (m *mockMCPClientAccessor) List() []domain.ServerName {
	names := make([]domain.ServerName, 0, len(m.names))
	for _, name := range m.names {
		names = append(names, name)
	}

	return names
}

func (m *mockMCPClientAccessor) UpdateTools(name domain.ServerName, tools []domain.ToolName) error {
	if _, ok := m.clients[name.Key()]; !ok {
		return fmt.Errorf("server '%s' not found", name)
	}
	m.tools[name.Key()] = tools

	return nil
}

func (m *mockMCPClientAccessor) Remove(name domain.ServerName) {
	delete(m.clients, name.Key())
	delete(m.names, name.Key())
	delete(m.tools, name.Key())
}

// mockMCPClient implements the client.MCPClient interface for testing.
type mockMCPClient struct {
	// Tools
	listToolsResult  *mcp.ListToolsResult
	listToolsError   error
	listToolsContext context.Context
	callToolResult   *mcp.CallToolResult
	callToolError    error
	callToolContext  context.Context
	calledToolName   string
	// Prompts
	listPromptsResult *mcp.ListPromptsResult
	listPromptsError  error
	getPromptResult   *mcp.GetPromptResult
	getPromptError    error
	// Resources
	listResourcesResult *mcp.ListResourcesResult
	listResourcesError  error
	listTemplatesResult *mcp.ListResourceTemplatesResult
	listTemplatesError  error
	readResourceResult  *mcp.ReadResourceResult
	readResourceError   error
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
	return m.listResourcesResult, m.listResourcesError
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
	return m.listTemplatesResult, m.listTemplatesError
}

func (m *mockMCPClient) ReadResource(
	_ context.Context,
	_ mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return m.readResourceResult, m.readResourceError
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
	return m.listPromptsResult, m.listPromptsError
}

func (m *mockMCPClient) GetPrompt(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return m.getPromptResult, m.getPromptError
}

func (m *mockMCPClient) ListToolsByPage(
	_ context.Context,
	_ mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return m.listToolsResult, m.listToolsError
}

func (m *mockMCPClient) ListTools(ctx context.Context, _ mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	m.listToolsContext = ctx

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return m.listToolsResult, m.listToolsError
}

func (m *mockMCPClient) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.callToolContext = ctx
	m.calledToolName = req.Params.Name

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

func TestHandleTools_MatchesAnAllowedNameExactly(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				{Name: "GetTime", Description: "Gets current time"},
				{Name: "SET_ALARM", Description: "Sets an alarm"},
				{Name: "list_events", Description: "Lists events"},
			},
		},
	}

	// One allowed name matches the server's spelling and one differs only by case.
	allowedTools := []domain.ToolName{"GetTime", "set_alarm"}
	accessor.Add("testserver", mockClient, allowedTools)

	result, err := newServerRoutes(accessor, RouteOptions{}).handleTools(context.Background(), "testserver")
	require.NoError(t, err)
	require.NotNil(t, result)

	toolNames := make([]string, len(result.Body.Tools))
	for i, tool := range result.Body.Tools {
		toolNames[i] = tool.Name
	}

	require.Equal(t, []string{"GetTime"}, toolNames)
}

func TestHandleToolCall_TrimsTheSuppliedName(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	mockClient := &mockMCPClient{
		callToolResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Text: "Tool executed successfully"},
			},
		},
	}

	accessor.Add("testserver", mockClient, []domain.ToolName{"GetTime"})

	// An MCP tool name holds no surrounding whitespace, so trimming cannot reach a second tool.
	result, err := newServerRoutes(accessor, RouteOptions{}).handleToolCall(
		context.Background(),
		"testserver",
		" GetTime ",
		map[string]any{},
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, "Tool executed successfully", result.Body)
	require.Equal(t, "GetTime", mockClient.calledToolName)
}

func TestHandleToolCall_RefusesANameThatDiffersOnlyByCase(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{{Name: "GetTime"}},
		},
		callToolResult: &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Text: "Tool executed successfully"}},
		},
	}

	// The allow list holds one spelling and the server reports another, which are two distinct tools.
	accessor.Add("testserver", mockClient, []domain.ToolName{"gettime"})

	_, err := newServerRoutes(accessor, RouteOptions{}).handleToolCall(
		context.Background(),
		"testserver",
		"GetTime",
		map[string]any{},
	)

	require.ErrorIs(t, err, errors.ErrToolForbidden)
	require.Empty(t, mockClient.calledToolName)
}

func TestNewServerRoutes_ToolCallTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{name: "zero uses default", timeout: 0, expected: DefaultToolCallTimeout()},
		{name: "negative uses default", timeout: -time.Second, expected: DefaultToolCallTimeout()},
		{name: "positive is kept", timeout: 30 * time.Second, expected: 30 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			routes := newServerRoutes(newMockMCPClientAccessor(), RouteOptions{ToolCallTimeout: tc.timeout})
			require.Equal(t, tc.expected, routes.toolCallTimeout)
		})
	}
}

func TestHandleToolCall_UsesConfiguredTimeout(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()
	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{Tools: []mcp.Tool{{Name: "gettime"}}},
		callToolResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Text: "Tool executed successfully"},
			},
		},
	}
	accessor.Add("testserver", mockClient, []domain.ToolName{"gettime"})

	timeout := 30 * time.Second
	routes := newServerRoutes(accessor, RouteOptions{ToolCallTimeout: timeout})
	result, err := routes.handleToolCall(
		context.Background(),
		"testserver",
		"gettime",
		map[string]any{},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, mockClient.callToolContext)

	deadline, ok := mockClient.callToolContext.Deadline()
	require.True(t, ok)

	remaining := time.Until(deadline)
	require.Greater(t, remaining, 29*time.Second)
	require.LessOrEqual(t, remaining, timeout)
}

func TestHandleToolCall_ToolNotAllowed(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	mockClient := &mockMCPClient{}
	allowedTools := []domain.ToolName{"gettime"}
	accessor.Add("testserver", mockClient, allowedTools)

	// Try to call a tool that's not in the allowed list.
	result, err := newServerRoutes(accessor, RouteOptions{}).handleToolCall(
		context.Background(),
		"testserver",
		"forbidden_tool",
		map[string]any{},
	)
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrToolForbidden)
}

func TestHandleToolCall_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := newServerRoutes(accessor, RouteOptions{}).handleToolCall(
		context.Background(),
		"nonexistent",
		"tool",
		map[string]any{},
	)
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrServerNotFound)
}

func TestHandleTools_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := newServerRoutes(accessor, RouteOptions{}).handleTools(context.Background(), "nonexistent")
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrServerNotFound)
}

func TestHandleTools_NoTools(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()
	mockClient := &mockMCPClient{}

	// Add server with no tools.
	accessor.Add("testserver", mockClient, []domain.ToolName{})

	result, err := newServerRoutes(accessor, RouteOptions{}).handleTools(context.Background(), "testserver")
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrToolsNotFound)
}

func TestHandleTools_PropagatesRequestCancellation(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()
	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{{Name: "gettime"}},
		},
	}
	accessor.Add("testserver", mockClient, []domain.ToolName{"gettime"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newServerRoutes(accessor, RouteOptions{}).handleTools(ctx, "testserver")
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, errors.ErrToolListFailed)
}

func TestHandleTools_AppliesRequestTimeout(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()
	mockClient := &mockMCPClient{
		listToolsResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{{Name: "gettime"}},
		},
	}
	accessor.Add("testserver", mockClient, []domain.ToolName{"gettime"})

	_, err := newServerRoutes(accessor, RouteOptions{}).handleTools(context.Background(), "testserver")
	require.NoError(t, err)
	require.NotNil(t, mockClient.listToolsContext)

	deadline, ok := mockClient.listToolsContext.Deadline()
	require.True(t, ok)
	require.LessOrEqual(t, time.Until(deadline), mcpRequestTimeout)
}
