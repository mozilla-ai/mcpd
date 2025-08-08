package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// mockMCPClient is a test implementation of client.MCPClient
type mockMCPClient struct{}

func (m *mockMCPClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockMCPClient) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return nil
}

func (m *mockMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return nil
}

func (m *mockMCPClient) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListPrompts(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return nil, nil
}

func (m *mockMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

func (m *mockMCPClient) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return nil, nil
}

func (m *mockMCPClient) Close() error {
	return nil
}

func (m *mockMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {}

func TestClientManager_Add_Client_Tools(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	c := &mockMCPClient{}
	tools := []string{"tool1", "tool2"}
	name := "server1"

	cm.Add(name, c, tools)

	// Test Client retrieval
	rc, ok := cm.Client(name)
	require.True(t, ok)
	require.Equal(t, c, rc)

	// Test Tools retrieval
	rt, ok := cm.Tools(name)
	require.True(t, ok)
	require.Equal(t, tools, rt)
}

func TestClientManager_List(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add("server1", &mockMCPClient{}, []string{"a"})
	cm.Add("server2", &mockMCPClient{}, []string{"b"})

	names := cm.List()
	require.Len(t, names, 2)
	require.ElementsMatch(t, []string{"server1", "server2"}, names)
}

func TestClientManager_Remove(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add("server1", &mockMCPClient{}, []string{"tool"})
	cm.Remove("server1")

	_, ok := cm.Client("server1")
	require.False(t, ok)

	_, ok = cm.Tools("server1")
	require.False(t, ok)

	require.Empty(t, cm.List())
}

func TestClientManager_EmptyManager(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	_, ok := cm.Client("missing")
	require.False(t, ok)

	_, ok = cm.Tools("missing")
	require.False(t, ok)

	require.Empty(t, cm.List())
}

// TestClientManager_ConcurrentAccess can be run with: go test -race ./...
func TestClientManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		name := fmt.Sprintf("server-%d", i)
		go func() {
			defer wg.Done()
			cm.Add(name, &mockMCPClient{}, []string{"tool"})
		}()
		go func() {
			defer wg.Done()
			_, _ = cm.Client(name)
		}()
		go func() {
			defer wg.Done()
			cm.List()
		}()
	}

	wg.Wait()
}
