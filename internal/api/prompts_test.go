package api

import (
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/mozilla-ai/mcpd/v2/internal/errors"
)

func TestAPI_HandleServerPrompts_Success(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		serverName     string
		prompts        []mcp.Prompt
		nextCursor     mcp.Cursor
		expectedCount  int
		expectedCursor string
	}{
		{
			name:       "single prompt",
			serverName: "test-server",
			prompts: []mcp.Prompt{
				{
					Name:        "test-prompt",
					Description: "A test prompt",
				},
			},
			expectedCount: 1,
		},
		{
			name:       "multiple prompts",
			serverName: "test-server",
			prompts: []mcp.Prompt{
				{
					Name:        "prompt1",
					Description: "First prompt",
				},
				{
					Name:        "prompt2",
					Description: "Second prompt",
					Arguments: []mcp.PromptArgument{
						{
							Name:        "arg1",
							Description: "Test argument",
							Required:    true,
						},
					},
				},
			},
			expectedCount: 2,
		},
		{
			name:       "prompts with cursor",
			serverName: "test-server",
			prompts: []mcp.Prompt{
				{
					Name:        "page-prompt",
					Description: "A paginated prompt",
				},
			},
			nextCursor:     "next-page",
			expectedCount:  1,
			expectedCursor: "next-page",
		},
		{
			name:          "empty prompts list",
			serverName:    "empty-server",
			prompts:       []mcp.Prompt{},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mockMCPClient{
				listPromptsResult: mcp.NewListPromptsResult(tc.prompts, tc.nextCursor),
			}

			accessor := newMockMCPClientAccessor()
			accessor.Add(tc.serverName, mockClient, []string{})

			result, err := handleServerPrompts(accessor, tc.serverName, "")

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Body.Prompts, tc.expectedCount)
			require.Equal(t, tc.expectedCursor, result.Body.NextCursor)

			if len(tc.prompts) > 0 {
				require.Equal(t, tc.prompts[0].Name, result.Body.Prompts[0].Name)
				require.Equal(t, tc.prompts[0].Description, result.Body.Prompts[0].Description)
			}
		})
	}
}

func TestAPI_HandleServerPrompts_WithCursor(t *testing.T) {
	t.Parallel()

	cursor := "test-cursor"
	mockClient := &mockMCPClient{
		listPromptsResult: mcp.NewListPromptsResult([]mcp.Prompt{
			{Name: "page2-prompt", Description: "page 2"},
		}, ""),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerPrompts(accessor, "test-server", cursor)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body.Prompts, 1)
}

func TestAPI_HandleServerPrompts_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	result, err := handleServerPrompts(accessor, "nonexistent-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrServerNotFound))
}

func TestAPI_HandleServerPrompts_ListError(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		listPromptsError: errors.New("prompt list failed"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerPrompts(accessor, "test-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptListFailed))
}

func TestAPI_HandleServerPrompts_NilResult(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		listPromptsResult: nil,
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerPrompts(accessor, "test-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptListFailed))
}

func TestAPI_HandleServerPrompts_MethodNotFound(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		listPromptsError: errors.New("Method not found"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	result, err := handleServerPrompts(accessor, "test-server", "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptsNotImplemented))
}

func TestAPI_HandleServerPromptGenerate_Success(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptResult: &mcp.GetPromptResult{
			Description: "Test prompt result",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "Hello, world!"},
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "test-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Test prompt result", result.Body.Description)
	require.Len(t, result.Body.Messages, 1)
	require.Equal(t, "user", result.Body.Messages[0].Role)
}

func TestAPI_HandleServerPromptGenerate_WithArguments(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptResult: &mcp.GetPromptResult{
			Description: "Parameterized prompt",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleAssistant,
					Content: mcp.TextContent{Type: "text", Text: "Templated response"},
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "template-prompt"
	arguments := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Parameterized prompt", result.Body.Description)
	require.Len(t, result.Body.Messages, 1)
	require.Equal(t, "assistant", result.Body.Messages[0].Role)
}

func TestAPI_HandleServerPromptGenerate_MultipleMessages(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptResult: &mcp.GetPromptResult{
			Description: "Multi-message prompt",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "First message"},
				},
				{
					Role:    mcp.RoleAssistant,
					Content: mcp.TextContent{Type: "text", Text: "Second message"},
				},
			},
		},
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "multi-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Body.Messages, 2)
	require.Equal(t, "user", result.Body.Messages[0].Role)
	require.Equal(t, "assistant", result.Body.Messages[1].Role)
}

func TestAPI_HandleServerPromptGenerate_ServerNotFound(t *testing.T) {
	t.Parallel()

	accessor := newMockMCPClientAccessor()

	promptName := "test-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "nonexistent-server", promptName, arguments)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrServerNotFound))
}

func TestAPI_HandleServerPromptGenerate_GenerateError(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptError: errors.New("prompt get failed"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "nonexistent-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptGetFailed))
}

func TestAPI_HandleServerPromptGenerate_NilResult(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptResult: nil,
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "test-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptGetFailed))
}

func TestAPI_HandleServerPromptGenerate_MethodNotFound(t *testing.T) {
	t.Parallel()

	mockClient := &mockMCPClient{
		getPromptError: errors.New("Method not found"),
	}

	accessor := newMockMCPClientAccessor()
	accessor.Add("test-server", mockClient, []string{})

	promptName := "test-prompt"
	arguments := map[string]string{}

	result, err := handleServerPromptGenerate(accessor, "test-server", promptName, arguments)

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, internalerrors.ErrPromptsNotImplemented))
}

func TestDomainPrompt_ToAPIType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prompt   mcp.Prompt
		expected Prompt
	}{
		{
			name: "simple prompt",
			prompt: mcp.Prompt{
				Name:        "simple",
				Description: "A simple prompt",
			},
			expected: Prompt{
				Name:        "simple",
				Description: "A simple prompt",
				Arguments:   []PromptArgument{},
			},
		},
		{
			name: "prompt with arguments",
			prompt: mcp.Prompt{
				Name:        "template",
				Description: "A template prompt",
				Arguments: []mcp.PromptArgument{
					{
						Name:        "param1",
						Description: "First parameter",
						Required:    true,
					},
					{
						Name:        "param2",
						Description: "Second parameter",
						Required:    false,
					},
				},
			},
			expected: Prompt{
				Name:        "template",
				Description: "A template prompt",
				Arguments: []PromptArgument{
					{
						Name:        "param1",
						Description: "First parameter",
						Required:    true,
					},
					{
						Name:        "param2",
						Description: "Second parameter",
						Required:    false,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := DomainPrompt(tc.prompt).ToAPIType()

			require.NoError(t, err)
			require.Equal(t, tc.expected.Name, result.Name)
			require.Equal(t, tc.expected.Description, result.Description)
			require.Len(t, result.Arguments, len(tc.expected.Arguments))

			for i, arg := range tc.expected.Arguments {
				require.Equal(t, arg.Name, result.Arguments[i].Name)
				require.Equal(t, arg.Description, result.Arguments[i].Description)
				require.Equal(t, arg.Required, result.Arguments[i].Required)
			}
		})
	}
}

func TestDomainPromptArgument_ToAPIType(t *testing.T) {
	t.Parallel()

	arg := mcp.PromptArgument{
		Name:        "test-arg",
		Description: "Test argument",
		Required:    true,
	}

	result, err := DomainPromptArgument(arg).ToAPIType()

	require.NoError(t, err)
	require.Equal(t, "test-arg", result.Name)
	require.Equal(t, "Test argument", result.Description)
	require.True(t, result.Required)
}

func TestDomainPromptMessage_ToAPIType(t *testing.T) {
	t.Parallel()

	message := mcp.PromptMessage{
		Role:    mcp.RoleUser,
		Content: mcp.TextContent{Type: "text", Text: "Hello"},
	}

	result, err := DomainPromptMessage(message).ToAPIType()

	require.NoError(t, err)
	require.Equal(t, "user", result.Role)
	require.NotNil(t, result.Content)
}
