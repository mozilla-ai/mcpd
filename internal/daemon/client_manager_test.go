package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/domain"
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
	tools := []domain.ToolName{"tool1", "tool2"}
	name := domain.ServerName("server1")

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

func TestClientManager_Add_ToolsNormalized(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	c := &mockMCPClient{}
	tools := []domain.ToolName{" Tool1 ", "TOOL2", "tool3"}
	name := domain.ServerName("server1")

	cm.Add(name, c, tools)

	// A tool name is stored as the operator configured it, because matching compares keys rather than
	// the stored form.
	rt, ok := cm.Tools(name)
	require.True(t, ok)
	require.Equal(t, tools, rt)

	keys := make([]string, 0, len(rt))
	for _, tool := range rt {
		keys = append(keys, tool.Key())
	}
	require.Equal(t, []string{"tool1", "tool2", "tool3"}, keys)
}

func TestClientManager_ServerNameNormalized(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	c := &mockMCPClient{}
	tools := []domain.ToolName{"tool1"}

	// Add with mixed case server name.
	cm.Add(" GitHub-Server ", c, tools)

	// Test retrieving with different cases should work.
	rc, ok := cm.Client(domain.ServerName("github-server"))
	require.True(t, ok)
	require.Equal(t, c, rc)

	rc, ok = cm.Client(domain.ServerName("GITHUB-SERVER"))
	require.True(t, ok)
	require.Equal(t, c, rc)

	rc, ok = cm.Client(domain.ServerName("  GitHub-Server  "))
	require.True(t, ok)
	require.Equal(t, c, rc)

	// Test Tools with different cases.
	rt, ok := cm.Tools(domain.ServerName("github-server"))
	require.True(t, ok)
	require.Equal(t, tools, rt)

	rt, ok = cm.Tools(domain.ServerName("GITHUB-SERVER"))
	require.True(t, ok)
	require.Equal(t, tools, rt)

	// Test Remove with different case.
	cm.Remove(domain.ServerName("GITHUB-SERVER"))
	_, ok = cm.Client(domain.ServerName("github-server"))
	require.False(t, ok)
}

func TestClientManager_List(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add(domain.ServerName("server1"), &mockMCPClient{}, []domain.ToolName{"a"})
	cm.Add(domain.ServerName("server2"), &mockMCPClient{}, []domain.ToolName{"b"})

	names := cm.List()
	require.Len(t, names, 2)
	require.ElementsMatch(t, []domain.ServerName{"server1", "server2"}, names)
}

// TestClientManager_List_PreservesConfiguredCase pins the name the API reports.
// List backs GET /api/v1/servers, so a name returned in a normalized form is not the name the operator
// wrote in .mcpd.toml.
func TestClientManager_List_PreservesConfiguredCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured domain.ServerName
		expected   string
	}{
		{
			name:       "camel case is kept",
			configured: "gitHub",
			expected:   "gitHub",
		},
		{
			name:       "pascal case is kept",
			configured: "GitHub",
			expected:   "GitHub",
		},
		{
			name:       "surrounding whitespace is dropped",
			configured: "  GitHub-Server  ",
			expected:   "GitHub-Server",
		},
		{
			name:       "lowercase is unchanged",
			configured: "github",
			expected:   "github",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cm := NewClientManager()
			cm.Add(tc.configured, &mockMCPClient{}, []domain.ToolName{"tool"})

			listed := cm.List()
			require.Len(t, listed, 1)
			require.Equal(t, tc.expected, listed[0].String())
		})
	}
}

func TestClientManager_Remove(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add(domain.ServerName("server1"), &mockMCPClient{}, []domain.ToolName{"tool"})
	cm.Remove(domain.ServerName("server1"))

	_, ok := cm.Client(domain.ServerName("server1"))
	require.False(t, ok)

	_, ok = cm.Tools(domain.ServerName("server1"))
	require.False(t, ok)

	require.Empty(t, cm.List())
}

func TestClientManager_EmptyManager(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	_, ok := cm.Client(domain.ServerName("missing"))
	require.False(t, ok)

	_, ok = cm.Tools(domain.ServerName("missing"))
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

		name := domain.ServerName(fmt.Sprintf("server-%d", i))
		go func() {
			defer wg.Done()
			cm.Add(name, &mockMCPClient{}, []domain.ToolName{"tool"})
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

// TestClientManager_UpdateTools pins what a tools-only reload preserves. Updating the tools must not
// disturb the configured name or the client, because a reload routes a tools-only change here rather than
// restarting the server.
func TestClientManager_UpdateTools(t *testing.T) {
	t.Parallel()

	t.Run("keeps the configured name and the client", func(t *testing.T) {
		t.Parallel()

		cm := NewClientManager()
		c := &mockMCPClient{}
		cm.Add("GitHub", c, []domain.ToolName{"createRepository"})

		require.NoError(t, cm.UpdateTools("github", []domain.ToolName{"listRepositories"}))

		listed := cm.List()
		require.Equal(t, []domain.ServerName{"GitHub"}, listed)

		got, ok := cm.Client("GITHUB")
		require.True(t, ok)
		require.Same(t, c, got)

		tools, ok := cm.Tools("GitHub")
		require.True(t, ok)
		require.Equal(t, []domain.ToolName{"listRepositories"}, tools)
	})

	t.Run("reports an unknown server", func(t *testing.T) {
		t.Parallel()

		cm := NewClientManager()

		err := cm.UpdateTools("absent", []domain.ToolName{"anything"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "absent")
	})

	t.Run("does not alias the caller's slice", func(t *testing.T) {
		t.Parallel()

		cm := NewClientManager()
		cm.Add("notes", &mockMCPClient{}, []domain.ToolName{"read"})

		tools := []domain.ToolName{"write"}
		require.NoError(t, cm.UpdateTools("notes", tools))
		tools[0] = "mutated"

		stored, ok := cm.Tools("notes")
		require.True(t, ok)
		require.Equal(t, []domain.ToolName{"write"}, stored)
	})
}

// TestClientManager_Tools_DoesNotShareStoredState pins that a caller cannot reach the stored allow list.
// The allow list decides which tools may run, so a handler mutating a returned slice would change what
// every later request is permitted to call, and would race readers outside the lock.
func TestClientManager_Tools_DoesNotShareStoredState(t *testing.T) {
	t.Parallel()

	cm := NewClientManager()
	cm.Add("notes", &mockMCPClient{}, []domain.ToolName{"read_note"})

	first, ok := cm.Tools("notes")
	require.True(t, ok)
	first[0] = "write_note"

	second, ok := cm.Tools("notes")
	require.True(t, ok)
	require.Equal(t, []domain.ToolName{"read_note"}, second)
}
