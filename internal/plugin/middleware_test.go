package plugin

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/api"
	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestMiddleware_AppliesContentModification(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{
		rewritePath:           "/rewritten",
		rewriteRequestBody:    `{"rewritten":true}`,
		rewriteRequestHeaders: map[string]string{"X-Added": "yes"},
	}
	p := newTestContentPipeline(t, content)

	var seen *http.Request
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		seen = r.Clone(r.Context())
		seen.Body = io.NopCloser(strings.NewReader(string(body)))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	})

	req := httptest.NewRequest(http.MethodPost, "/original", strings.NewReader(`{"original":true}`))
	req.Header.Set("Authorization", "Bearer original-token")
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.NotNil(t, seen)

	body, err := io.ReadAll(seen.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"rewritten":true}`, string(body))
	require.Equal(t, "yes", seen.Header.Get("X-Added"))

	// A header the plugin did not set survives, so a plugin cannot strip credentials by omission.
	require.Equal(t, "Bearer original-token", seen.Header.Get("Authorization"))

	// The plugin asked for a different path, which routing must not follow.
	require.Equal(t, "/original", seen.URL.Path)
	require.Equal(t, http.MethodPost, seen.Method)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "upstream", recorder.Body.String())
}

func TestMiddleware_IgnoresContentReservedRequestHeaders(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{
		rewriteRequestHeaders: map[string]string{
			"connection":     "upgrade",
			"Content-Length": "999999",
			"X-Added":        "yes",
		},
	}
	p := newTestContentPipeline(t, content)

	var seen http.Header
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/original", strings.NewReader(`{"original":true}`))
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, "yes", seen.Get("X-Added"))

	// Message framing and the transport belong to mcpd, whatever case a plugin spells them in.
	require.Empty(t, seen.Get("Connection"))
	require.Empty(t, seen.Get("Content-Length"))
}

func TestMiddleware_IgnoresContentMethodAndPathRewrite(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{
		rewriteMethod: http.MethodDelete,
		rewritePath:   "/api/v1/servers/alpha/tools/danger",
	}
	p := newTestContentPipeline(t, content)

	var (
		seenMethod string
		seenPath   string
	)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		seenPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, http.MethodGet, seenMethod)
	require.Equal(t, "/api/v1/servers", seenPath)
}

func TestMiddleware_AppliesContentResponseModification(t *testing.T) {
	t.Parallel()

	redactor := &testContentPlugin{rewriteResponseBody: `{"token":"[REDACTED]"}`}
	p := newTestContentPipeline(t, redactor)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"super-secret"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/original", nil)
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, []string{`{"token":"super-secret"}`}, redactor.seenResponseBodies)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"token":"[REDACTED]"}`, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), "super-secret")
}

func TestMiddleware_PassesOriginalResponseWhenUnmodified(t *testing.T) {
	t.Parallel()

	p := newTestContentPipeline(t, &testContentPlugin{})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"tools":[]}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/original", nil)
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.JSONEq(t, `{"tools":[]}`, recorder.Body.String())
}

func TestMiddleware_PassesOriginalRequestWhenUnmodified(t *testing.T) {
	t.Parallel()

	p := newTestContentPipeline(t, &testContentPlugin{})

	var (
		seenPath string
		seenBody string
	)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		seenPath = r.URL.Path
		seenBody = string(body)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	})

	req := httptest.NewRequest(http.MethodPost, "/original", strings.NewReader(`{"original":true}`))
	recorder := httptest.NewRecorder()
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, "/original", seenPath)
	require.JSONEq(t, `{"original":true}`, seenBody)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "upstream", recorder.Body.String())
}

func TestMiddleware_RequestPipelineFailure(t *testing.T) {
	t.Parallel()

	// Create pipeline with a required plugin that fails during request.
	logger := hclog.NewNullLogger()
	p := newPipeline(logger)

	// Create failing plugin instance.
	inst := &Instance{
		Plugin: &mockPlugin{
			capabilities: []config.Flow{config.FlowRequest},
			requestErr:   errors.New("plugin failed"),
		},
		name:     "failing-plugin",
		required: true,
	}

	// Configure for REQUEST.
	inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

	// Add to pipeline.
	p.plugins[config.CategoryAuthentication] = []*Instance{inst}

	// Create test handler.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when request pipeline fails")
	})

	// Apply middleware.
	handler := p.Middleware()(nextHandler)

	// Execute request.
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Assert: 500 status and correct header.
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, string(api.PipelineRequestFailure), recorder.Header().Get(api.HeaderErrorType))
	require.Contains(t, recorder.Body.String(), "Request processing failed")
}

func TestMiddleware_ResponsePipelineFailure(t *testing.T) {
	t.Parallel()

	// Create pipeline with a required plugin that fails during response.
	logger := hclog.NewNullLogger()
	p := newPipeline(logger)

	// Create failing plugin instance.
	inst := &Instance{
		Plugin: &mockPlugin{
			capabilities: []config.Flow{config.FlowResponse},
			responseErr:  errors.New("plugin failed"),
		},
		name:     "failing-plugin",
		required: true,
	}

	// Configure for RESPONSE.
	inst.SetFlows(map[config.Flow]struct{}{config.FlowResponse: {}})

	// Add to pipeline.
	p.plugins[config.CategoryAuthentication] = []*Instance{inst}

	// Create test handler that succeeds.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream success"))
	})

	// Apply middleware.
	handler := p.Middleware()(nextHandler)

	// Execute request.
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Assert: 500 status and correct header.
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, string(api.PipelineResponseFailure), recorder.Header().Get(api.HeaderErrorType))
	require.Contains(t, recorder.Body.String(), "Response processing failed")
}

func TestWritePluginResponse_LogsClampedStatusCode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rec := httptest.NewRecorder()
	writePluginResponse(newTestLogger(&buf), rec, &HTTPResponse{Body: []byte("denied")})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, buf.String(), "replacing a status code a plugin may not report")
}

// TestWritePluginResponse_LogsReservedHeaderNameNotValue pins the rule that a reserved header's value never
// reaches the log, because the value may be a credential.
func TestWritePluginResponse_LogsReservedHeaderNameNotValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rec := httptest.NewRecorder()
	writePluginResponse(newTestLogger(&buf), rec, &HTTPResponse{
		StatusCode: http.StatusForbidden,
		Headers:    map[string]string{"Proxy-Authenticate": "Basic realm=super-secret-value"},
	})

	logged := buf.String()
	require.Contains(t, logged, "ignoring reserved response headers set by plugin")
	require.Contains(t, logged, "Proxy-Authenticate")
	require.NotContains(t, logged, "super-secret-value")
	require.Empty(t, rec.Header().Get("Proxy-Authenticate"))
}

// TestWritePluginResponse_LogsNothingForAValidResponse guards against a log line on every response.
func TestWritePluginResponse_LogsNothingForAValidResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rec := httptest.NewRecorder()
	writePluginResponse(newTestLogger(&buf), rec, &HTTPResponse{
		StatusCode: http.StatusTooManyRequests,
		Headers:    map[string]string{"Retry-After": "30"},
		Body:       []byte("slow down"),
	})

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "30", rec.Header().Get("Retry-After"))
	require.Empty(t, buf.String())
}
