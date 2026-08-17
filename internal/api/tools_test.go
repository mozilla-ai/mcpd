package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/url"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// mockHumaContext implements huma.Context for testing.
type mockHumaContext struct {
	queryParams map[string]string
}

func (m *mockHumaContext) Query(name string) string {
	return m.queryParams[name]
}

// Minimal no-op implementations for other required methods.
func (m *mockHumaContext) Operation() *huma.Operation                 { return nil }
func (m *mockHumaContext) Context() context.Context                   { return context.Background() }
func (m *mockHumaContext) TLS() *tls.ConnectionState                  { return nil }
func (m *mockHumaContext) Version() huma.ProtoVersion                 { return huma.ProtoVersion{} }
func (m *mockHumaContext) Method() string                             { return "" }
func (m *mockHumaContext) Host() string                               { return "" }
func (m *mockHumaContext) RemoteAddr() string                         { return "" }
func (m *mockHumaContext) URL() url.URL                               { return url.URL{} }
func (m *mockHumaContext) Param(name string) string                   { return "" }
func (m *mockHumaContext) Header(name string) string                  { return "" }
func (m *mockHumaContext) EachHeader(cb func(name, value string))     {}
func (m *mockHumaContext) BodyReader() io.Reader                      { return nil }
func (m *mockHumaContext) GetMultipartForm() (*multipart.Form, error) { return nil, nil }
func (m *mockHumaContext) SetReadDeadline(t time.Time) error          { return nil }
func (m *mockHumaContext) SetStatus(code int)                         {}
func (m *mockHumaContext) Status() int                                { return 0 }
func (m *mockHumaContext) SetHeader(name, value string)               {}
func (m *mockHumaContext) AppendHeader(name, value string)            {}
func (m *mockHumaContext) BodyWriter() io.Writer                      { return nil }

func TestToolFieldSelectTransformer_Full(t *testing.T) {
	t.Parallel()

	// Create a sample body with full tool details.
	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
				InputSchema: &JSONSchema{
					Type: "object",
					Properties: map[string]any{
						"param1": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	// Mock context that returns "full" detail level.
	mockCtx := &mockHumaContext{queryParams: map[string]string{}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should return the original body unchanged.
	resultBody, ok := result.(ToolsResponseBody[Tool])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.NotNil(t, resultBody.Tools[0].InputSchema)
}

func TestToolFieldSelectTransformer_Minimal(t *testing.T) {
	t.Parallel()

	// Create a sample body with full tool details.
	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
				InputSchema: &JSONSchema{
					Type: "object",
				},
			},
		},
	}

	// Mock context that returns "minimal" detail level.
	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: "minimal"}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should return only minimal fields.
	resultBody, ok := result.(ToolsResponseBody[ToolMinimal])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.Equal(t, "test-tool", resultBody.Tools[0].Name)
	require.Equal(t, "Test Tool", resultBody.Tools[0].Title)
}

func TestToolFieldSelectTransformer_Summary(t *testing.T) {
	t.Parallel()

	// Create a sample body with full tool details.
	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
				InputSchema: &JSONSchema{
					Type: "object",
				},
			},
		},
	}

	// Mock context that returns "summary" detail level.
	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: "summary"}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should return summary fields.
	resultBody, ok := result.(ToolsResponseBody[ToolSummary])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.Equal(t, "test-tool", resultBody.Tools[0].Name)
	require.Equal(t, "Test Tool", resultBody.Tools[0].Title)
	require.Equal(t, "A test tool", resultBody.Tools[0].Description)
}

func TestToolFieldSelectTransformer_InvalidDetailFallsBackToFull(t *testing.T) {
	t.Parallel()

	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
				InputSchema: &JSONSchema{
					Type: "object",
				},
			},
		},
	}

	// Mock context with invalid detail value.
	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: "unknown"}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should fallback to full and return original body unchanged.
	resultBody, ok := result.(ToolsResponseBody[Tool])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.NotNil(t, resultBody.Tools[0].InputSchema)
}

func TestToolFieldSelectTransformer_NormalizesCase(t *testing.T) {
	t.Parallel()

	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
			},
		},
	}

	// Mock context with uppercase detail value.
	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: "MINIMAL"}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should normalize to minimal.
	resultBody, ok := result.(ToolsResponseBody[ToolMinimal])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.Equal(t, "test-tool", resultBody.Tools[0].Name)
}

func TestToolFieldSelectTransformer_NormalizesWhitespace(t *testing.T) {
	t.Parallel()

	body := ToolsResponseBody[Tool]{
		Tools: []Tool{
			{
				ToolSummary: ToolSummary{
					ToolMinimal: ToolMinimal{
						Name:  "test-tool",
						Title: "Test Tool",
					},
					Description: "A test tool",
				},
			},
		},
	}

	// Mock context with whitespace around detail value.
	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: " summary "}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", body)
	require.NoError(t, err)

	// Should normalize to summary.
	resultBody, ok := result.(ToolsResponseBody[ToolSummary])
	require.True(t, ok)
	require.Len(t, resultBody.Tools, 1)
	require.Equal(t, "A test tool", resultBody.Tools[0].Description)
}

func TestToolFieldSelectTransformer_PassesThroughNonToolsResponseBody(t *testing.T) {
	t.Parallel()

	// Non-ToolsResponseBody type.
	otherResponse := map[string]any{"something": "else"}

	mockCtx := &mockHumaContext{queryParams: map[string]string{queryParamDetail: "minimal"}}

	result, err := toolFieldSelectTransformer(mockCtx, "200", otherResponse)
	require.NoError(t, err)

	// Should pass through unchanged.
	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "else", resultMap["something"])
}

func TestToolAnnotations_IsZero(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false
	emptyStr := ""
	nonEmpty := "title"

	tests := []struct {
		name     string
		input    *ToolAnnotations
		expected bool
	}{
		{
			name:     "nil receiver",
			input:    nil,
			expected: true,
		},
		{
			name:     "all nil fields",
			input:    &ToolAnnotations{},
			expected: true,
		},
		{
			name:     "empty title string",
			input:    &ToolAnnotations{Title: &emptyStr},
			expected: true,
		},
		{
			name:     "non-empty title string",
			input:    &ToolAnnotations{Title: &nonEmpty},
			expected: false,
		},
		{
			name:     "read-only hint true",
			input:    &ToolAnnotations{ReadOnlyHint: &trueVal},
			expected: false,
		},
		{
			name:     "destructive hint false",
			input:    &ToolAnnotations{DestructiveHint: &falseVal},
			expected: false,
		},
		{
			name:     "idempotent hint true",
			input:    &ToolAnnotations{IdempotentHint: &trueVal},
			expected: false,
		},
		{
			name:     "open world hint true",
			input:    &ToolAnnotations{OpenWorldHint: &trueVal},
			expected: false,
		},
		{
			name: "mixed non-empty fields",
			input: &ToolAnnotations{
				Title:         &nonEmpty,
				OpenWorldHint: &trueVal,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, tc.input.IsZero())
		})
	}
}

func TestDomainTool_ToAPIType_PreservesDefsAndAdditionalProperties(t *testing.T) {
	t.Parallel()

	addressDef := map[string]any{
		"type":       "object",
		"properties": map[string]any{"street": map[string]any{"type": "string"}},
	}

	tool := mcp.Tool{
		Name: "create-user",
		InputSchema: mcp.ToolInputSchema{
			Type:                 "object",
			Defs:                 map[string]any{"Address": addressDef},
			Properties:           map[string]any{"home": map[string]any{"$ref": "#/$defs/Address"}},
			Required:             []string{"home"},
			AdditionalProperties: false,
		},
		OutputSchema: mcp.ToolOutputSchema{
			Type:                 "object",
			Defs:                 map[string]any{"Address": addressDef},
			Properties:           map[string]any{"billing": map[string]any{"$ref": "#/$defs/Address"}},
			AdditionalProperties: map[string]any{"type": "string"},
		},
	}

	got, err := domainTool(tool).ToAPIType()
	require.NoError(t, err)

	// Assert on the serialised form rather than the struct: the wire output is
	// what clients actually consume, and a $ref that survives without its $defs
	// sibling is a dangling pointer to a subschema that no longer exists.
	for name, schema := range map[string]any{
		"inputSchema":  got.InputSchema,
		"outputSchema": got.OutputSchema,
	} {
		raw, err := json.Marshal(schema)
		require.NoError(t, err)

		var wire map[string]any
		require.NoError(t, json.Unmarshal(raw, &wire))

		require.Contains(t, wire, "$defs", "%s dropped $defs, leaving its $ref dangling", name)
		require.Contains(t, wire["$defs"], "Address", "%s lost the referenced subschema", name)
		require.Contains(t, wire, "additionalProperties", "%s dropped additionalProperties", name)
	}
}

func TestOutputSchemaPresent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		schema mcp.ToolOutputSchema
		want   bool
	}{
		"zero value is absent": {
			schema: mcp.ToolOutputSchema{},
			want:   false,
		},
		"empty collections are absent": {
			schema: mcp.ToolOutputSchema{
				Defs:       map[string]any{},
				Properties: map[string]any{},
				Required:   []string{},
			},
			want: false,
		},
		"type alone is present": {
			schema: mcp.ToolOutputSchema{Type: "object"},
			want:   true,
		},
		"defs alone is present": {
			schema: mcp.ToolOutputSchema{Defs: map[string]any{"Address": map[string]any{}}},
			want:   true,
		},
		"properties alone is present": {
			schema: mcp.ToolOutputSchema{Properties: map[string]any{"name": map[string]any{}}},
			want:   true,
		},
		"required alone is present": {
			schema: mcp.ToolOutputSchema{Required: []string{"name"}},
			want:   true,
		},
		// A non-nil interface holding false is content, not absence: it means
		// "no additional properties allowed" and must not be treated as unset.
		"additionalProperties false is present": {
			schema: mcp.ToolOutputSchema{AdditionalProperties: false},
			want:   true,
		},
		"additionalProperties schema is present": {
			schema: mcp.ToolOutputSchema{AdditionalProperties: map[string]any{"type": "string"}},
			want:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, outputSchemaPresent(tc.schema))
		})
	}
}

func TestDomainTool_ToAPIType_OutputSchema(t *testing.T) {
	t.Parallel()

	addressDef := map[string]any{
		"type":       "object",
		"properties": map[string]any{"street": map[string]any{"type": "string"}},
	}

	tests := map[string]struct {
		schema mcp.ToolOutputSchema
		// wantKeys lists wire keys that must survive conversion; a nil slice
		// means the schema is expected to be dropped entirely.
		wantKeys []string
	}{
		"empty schema is dropped": {
			schema:   mcp.ToolOutputSchema{},
			wantKeys: nil,
		},
		// A valid JSON Schema may omit a top-level "type" yet still define
		// content. Dropping it would strip the $defs the $ref depends on.
		"typeless schema is preserved": {
			schema: mcp.ToolOutputSchema{
				Defs:                 map[string]any{"Address": addressDef},
				Properties:           map[string]any{"billing": map[string]any{"$ref": "#/$defs/Address"}},
				AdditionalProperties: false,
			},
			wantKeys: []string{"$defs", "properties", "additionalProperties"},
		},
		"typed schema is preserved": {
			schema: mcp.ToolOutputSchema{
				Type:                 "object",
				Defs:                 map[string]any{"Address": addressDef},
				AdditionalProperties: map[string]any{"type": "string"},
			},
			wantKeys: []string{"type", "$defs", "additionalProperties"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := domainTool(mcp.Tool{Name: "tool", OutputSchema: tc.schema}).ToAPIType()
			require.NoError(t, err)

			if tc.wantKeys == nil {
				require.Nil(t, got.OutputSchema, "empty output schema should be omitted")
				return
			}

			require.NotNil(t, got.OutputSchema)

			raw, err := json.Marshal(got.OutputSchema)
			require.NoError(t, err)

			var wire map[string]any
			require.NoError(t, json.Unmarshal(raw, &wire))

			for _, key := range tc.wantKeys {
				require.Contains(t, wire, key, "output schema dropped %s", key)
			}
		})
	}
}
