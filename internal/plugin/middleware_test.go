package plugin

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		rewriteRequestHeaders: http.Header{"X-Added": {"yes"}},
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

func TestMiddleware_IgnoresContentReservedRequestHeaders(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{
		rewriteRequestHeaders: http.Header{
			"connection":     {"upgrade"},
			"Content-Length": {"999999"},
			"X-Added":        {"yes"},
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

// TestWritePluginResponse_LogsNothingForAValidResponse guards against a log line on every response.
func TestWritePluginResponse_LogsNothingForAValidResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rec := httptest.NewRecorder()
	writePluginResponse(newTestLogger(&buf), rec, &HTTPResponse{
		StatusCode: http.StatusTooManyRequests,
		Headers:    http.Header{"Retry-After": {"30"}},
		Body:       []byte("slow down"),
	})

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "30", rec.Header().Get("Retry-After"))
	require.Empty(t, buf.String())
}

// TestWritePluginResponse_SkipsReservedHeaderWithoutLogging pins that the writer still withholds a
// reserved header, and names no plugin for a header it cannot attribute to one.
func TestWritePluginResponse_SkipsReservedHeaderWithoutLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	rec := httptest.NewRecorder()
	writePluginResponse(newTestLogger(&buf), rec, &HTTPResponse{
		StatusCode: http.StatusForbidden,
		Headers:    http.Header{"Proxy-Authenticate": {"Basic realm=super-secret-value"}},
	})

	require.Empty(t, rec.Header().Get("Proxy-Authenticate"))
	require.Empty(t, buf.String())
}

// TestMiddleware_PreservesMultiValueResponseHeader pins that mcpd does not rewrite the headers of a
// response no plugin modified, because the plugin view holds one value per name.
func TestMiddleware_PreservesMultiValueResponseHeader(t *testing.T) {
	t.Parallel()

	p := newTestContentPipeline(t, &testContentPlugin{})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/original", nil)
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, []string{"a=1", "b=2"}, recorder.Result().Header.Values("Set-Cookie"))
	require.Equal(t, "upstream", recorder.Body.String())
}

// TestMiddleware_LogsNothingWhenNoPluginIsRegistered guards against a warning on every response, which
// an operator reads as a plugin asking for a change that no plugin asked for.
func TestMiddleware_LogsNothingWhenNoPluginIsRegistered(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	p := newPipeline(newTestLogger(&buf))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set by the operator's CORS layer, which runs outside this middleware.
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", "https://example.test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/original", nil)
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Empty(t, buf.String())
	require.Equal(t, "Origin", recorder.Header().Get("Vary"))
}

// TestMiddleware_PreservesMultiValueRequestHeaderThroughAPlugin runs the real gRPC adapter, because a
// plugin only ever sees one value per header name and so can only ever echo one back.
func TestMiddleware_PreservesMultiValueRequestHeaderThroughAPlugin(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	client := &stubPluginClient{echoRequestHeaders: true, rewriteRequestBody: `{"rewritten":true}`}
	adapter, err := NewGRPCAdapter(client, time.Second)
	require.NoError(t, err)

	p := newPipeline(newTestLogger(&buf))
	require.NoError(t, p.Register(config.CategoryContent, newTestContentInstance("content-0", adapter)))

	var (
		seenAccept []string
		seenBody   string
	)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAccept = r.Header.Values("Accept")

		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr)
		seenBody = string(body)

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/original", strings.NewReader(`{"original":true}`))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", "text/event-stream")
	// A real server carries Content-Length in the request header map, where httptest.NewRequest does not.
	req.Header.Set("Content-Length", "17")

	p.Middleware()(nextHandler).ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, []string{"application/json", "text/event-stream"}, seenAccept)
	require.JSONEq(t, `{"rewritten":true}`, seenBody)
	require.Empty(t, buf.String())
}

// TestMiddleware_LogsNothingForHeadersAPluginDidNotSet guards against naming a plugin for a reserved
// header the handler or the operator's middleware set, which happens on any response a plugin touched.
func TestMiddleware_LogsNothingForHeadersAPluginDidNotSet(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	content := &testContentPlugin{rewriteResponseHeaders: http.Header{"X-Scanned": {"yes"}}}
	p := newTestLoggingPipeline(t, &buf, config.CategoryContent, content)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set by the operator's CORS layer, which runs outside this middleware.
		w.Header().Set("Vary", "Origin")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/original", nil)
	p.Middleware()(nextHandler).ServeHTTP(recorder, req)

	require.Equal(t, "yes", recorder.Header().Get("X-Scanned"))
	require.Equal(t, "Origin", recorder.Header().Get("Vary"))
	require.Empty(t, buf.String())
}
