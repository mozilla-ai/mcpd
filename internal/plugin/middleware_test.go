package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/api"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

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
