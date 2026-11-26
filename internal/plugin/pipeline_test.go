package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// mockPlugin implements the Plugin interface for testing.
type mockPlugin struct {
	capabilities []config.Flow
	healthErr    error
	readyErr     error
	requestResp  *HTTPResponse
	requestErr   error
	responseResp *HTTPResponse
	responseErr  error
	stopErr      error
}

// testContentPlugin is a mock content plugin that records the state it was given and returns a modification.
type testContentPlugin struct {
	// seenRequestPaths records the request path passed to each HandleRequest call, in call order.
	seenRequestPaths []string

	// seenResponseBodies records the response body passed to each HandleResponse call, in call order.
	seenResponseBodies []string

	// rewritePath is returned as the modified request path when it is not empty.
	rewritePath string

	// rewriteRequestBody is returned as the modified request body when it is not empty.
	rewriteRequestBody string

	// rewriteRequestHeaders is returned as the modified request headers when it is not nil.
	rewriteRequestHeaders map[string]string

	// rewriteBody is returned as the modified response body when it is not empty.
	rewriteBody string

	// rewriteStatus is returned as the modified response status when it is not zero.
	rewriteStatus int32
}

// testTrackingPlugin is a mock that calls callbacks to track execution.
type testTrackingPlugin struct {
	capabilities []config.Flow
	onRequest    func()
	onResponse   func()
}

func (m *mockPlugin) Capabilities(ctx context.Context) ([]config.Flow, error) {
	return m.capabilities, nil
}

func (m *mockPlugin) CheckHealth(ctx context.Context) error {
	return m.healthErr
}

func (m *mockPlugin) CheckReady(ctx context.Context) error {
	return m.readyErr
}

func (m *mockPlugin) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	if m.requestErr != nil {
		return nil, m.requestErr
	}
	return m.requestResp, nil
}

func (m *mockPlugin) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	if m.responseErr != nil {
		return nil, m.responseErr
	}
	return m.responseResp, nil
}

func (m *mockPlugin) Stop(ctx context.Context) error {
	return m.stopErr
}

func (c *testContentPlugin) Capabilities(ctx context.Context) ([]config.Flow, error) {
	return []config.Flow{config.FlowRequest, config.FlowResponse}, nil
}

func (c *testContentPlugin) CheckHealth(ctx context.Context) error {
	return nil
}

func (c *testContentPlugin) CheckReady(ctx context.Context) error {
	return nil
}

func (c *testContentPlugin) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	c.seenRequestPaths = append(c.seenRequestPaths, req.Path)

	resp := &HTTPResponse{Continue: true}
	if c.rewritePath == "" && c.rewriteRequestBody == "" && c.rewriteRequestHeaders == nil {
		return resp, nil
	}

	mod := &HTTPRequest{
		Method:  req.Method,
		Path:    req.Path,
		Headers: req.Headers,
		Body:    req.Body,
	}
	if c.rewritePath != "" {
		mod.Path = c.rewritePath
	}
	if c.rewriteRequestBody != "" {
		mod.Body = []byte(c.rewriteRequestBody)
	}
	if c.rewriteRequestHeaders != nil {
		mod.Headers = c.rewriteRequestHeaders
	}
	resp.ModifiedRequest = mod

	return resp, nil
}

func (c *testContentPlugin) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	c.seenResponseBodies = append(c.seenResponseBodies, string(resp.Body))

	out := &HTTPResponse{Continue: true}
	if c.rewriteStatus != 0 {
		out.StatusCode = c.rewriteStatus
	}
	if c.rewriteBody != "" {
		out.Body = []byte(c.rewriteBody)
	}

	return out, nil
}

func (c *testContentPlugin) Stop(ctx context.Context) error {
	return nil
}

func (t *testTrackingPlugin) Capabilities(ctx context.Context) ([]config.Flow, error) {
	return t.capabilities, nil
}

func (t *testTrackingPlugin) CheckHealth(ctx context.Context) error {
	return nil
}

func (t *testTrackingPlugin) CheckReady(ctx context.Context) error {
	return nil
}

func (t *testTrackingPlugin) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	if t.onRequest != nil {
		t.onRequest()
	}
	return &HTTPResponse{Continue: true}, nil
}

func (t *testTrackingPlugin) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	if t.onResponse != nil {
		t.onResponse()
	}
	return &HTTPResponse{Continue: true}, nil
}

func (t *testTrackingPlugin) Stop(ctx context.Context) error {
	return nil
}

// newTestContentInstance returns a content plugin instance configured for the request and response flows.
func newTestContentInstance(name string, plugin *testContentPlugin) *Instance {
	inst := &Instance{
		Plugin: plugin,
		name:   name,
	}
	inst.SetFlows(map[config.Flow]struct{}{
		config.FlowRequest:  {},
		config.FlowResponse: {},
	})

	return inst
}

// newTestContentPipeline returns a pipeline with the given content plugins registered in call order.
func newTestContentPipeline(t *testing.T, plugins ...*testContentPlugin) *pipeline {
	t.Helper()

	p := newPipeline(hclog.NewNullLogger())
	for i, plugin := range plugins {
		err := p.Register(config.CategoryContent, newTestContentInstance(fmt.Sprintf("content-%d", i), plugin))
		require.NoError(t, err)
	}

	return p
}

func TestPipeline_NewPipeline(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	pipeline := newPipeline(logger)

	require.NotNil(t, pipeline)
	require.NotNil(t, pipeline.logger)
	require.NotNil(t, pipeline.plugins)
	require.Empty(t, pipeline.plugins)
}

func TestPipeline_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		category    config.Category
		instance    *Instance
		expectError bool
	}{
		{
			name:     "valid authentication plugin",
			category: config.CategoryAuthentication,
			instance: &Instance{
				Plugin: &mockPlugin{capabilities: []config.Flow{config.FlowRequest}},
				name:   "test-auth",
			},
			expectError: false,
		},
		{
			name:     "valid observability plugin",
			category: config.CategoryObservability,
			instance: &Instance{
				Plugin: &mockPlugin{capabilities: []config.Flow{config.FlowRequest, config.FlowResponse}},
				name:   "test-observability",
			},
			expectError: false,
		},
		{
			name:        "nil instance",
			category:    config.CategoryAuthentication,
			instance:    nil,
			expectError: true,
		},
		{
			name:     "invalid category",
			category: config.Category("invalid"),
			instance: &Instance{
				Plugin: &mockPlugin{capabilities: []config.Flow{config.FlowRequest}},
				name:   "test-plugin",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			err := pipeline.Register(tc.category, tc.instance)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, pipeline.plugins[tc.category], 1)
			}
		})
	}
}

func TestPipeline_HandleRequest_Serial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plugins        map[config.Category][]*Instance
		request        *HTTPRequest
		expectError    bool
		expectContinue bool
		expectStatus   int32
	}{
		{
			name: "all plugins pass",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp:  &HTTPResponse{Continue: true},
					},
					name: "auth-plugin",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return map[config.Category][]*Instance{
					config.CategoryAuthentication: {inst},
				}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
		{
			name: "plugin stops pipeline",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp: &HTTPResponse{
							Continue:   false,
							StatusCode: 401,
							Body:       []byte("Unauthorized"),
						},
					},
					name: "auth-plugin",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return map[config.Category][]*Instance{
					config.CategoryAuthentication: {inst},
				}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: false,
			expectStatus:   401,
		},
		{
			name: "required plugin fails",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestErr:   errors.New("plugin error"),
					},
					name:     "auth-plugin",
					required: true,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return map[config.Category][]*Instance{
					config.CategoryAuthentication: {inst},
				}
			}(),
			request:     &HTTPRequest{Method: "GET", Path: "/test"},
			expectError: true,
		},
		{
			name: "optional plugin fails",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestErr:   errors.New("plugin error"),
					},
					name:     "auth-plugin",
					required: false,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return map[config.Category][]*Instance{
					config.CategoryAuthentication: {inst},
				}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
		{
			name: "content modification",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp: &HTTPResponse{
							Continue: true,
							ModifiedRequest: &HTTPRequest{
								Method:  "POST",
								Path:    "/modified",
								Headers: map[string]string{"X-Modified": "true"},
							},
						},
					},
					name: "content-plugin",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return map[config.Category][]*Instance{
					config.CategoryContent: {inst},
				}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			for category, instances := range tc.plugins {
				for _, instance := range instances {
					err := pipeline.Register(category, instance)
					require.NoError(t, err)
				}
			}

			resp, err := pipeline.HandleRequest(context.Background(), tc.request)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.expectContinue, resp.Continue)
				if !tc.expectContinue {
					require.Equal(t, tc.expectStatus, resp.StatusCode)
				}
			}
		})
	}
}

func TestPipeline_HandleRequest_Parallel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plugins        []*Instance
		request        *HTTPRequest
		expectError    bool
		expectContinue bool
	}{
		{
			name: "all plugins pass",
			plugins: func() []*Instance {
				inst1 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp:  &HTTPResponse{Continue: true},
					},
					name: "obs-plugin-1",
				}
				inst1.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				inst2 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp:  &HTTPResponse{Continue: true},
					},
					name: "obs-plugin-2",
				}
				inst2.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				return []*Instance{inst1, inst2}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
		{
			name: "optional plugin rejects but ignored in observability",
			plugins: func() []*Instance {
				inst1 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp:  &HTTPResponse{Continue: true},
					},
					name: "obs-plugin-1",
				}
				inst1.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				inst2 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp: &HTTPResponse{
							Continue:   false,
							StatusCode: 429,
						},
					},
					name:     "obs-plugin-2",
					required: false,
				}
				inst2.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				return []*Instance{inst1, inst2}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
		{
			name: "required plugin rejects in observability",
			plugins: func() []*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestResp: &HTTPResponse{
							Continue:   false,
							StatusCode: 429,
						},
					},
					name:     "obs-plugin-required",
					required: true,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return []*Instance{inst}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: false,
		},
		{
			name: "required plugin fails",
			plugins: func() []*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestErr:   errors.New("plugin error"),
					},
					name:     "obs-plugin-1",
					required: true,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return []*Instance{inst}
			}(),
			request:     &HTTPRequest{Method: "GET", Path: "/test"},
			expectError: true,
		},
		{
			name: "optional plugin fails",
			plugins: func() []*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
						requestErr:   errors.New("plugin error"),
					},
					name:     "obs-plugin-1",
					required: false,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return []*Instance{inst}
			}(),
			request:        &HTTPRequest{Method: "GET", Path: "/test"},
			expectError:    false,
			expectContinue: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			for _, instance := range tc.plugins {
				err := pipeline.Register(config.CategoryObservability, instance)
				require.NoError(t, err)
			}

			resp, err := pipeline.HandleRequest(context.Background(), tc.request)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.expectContinue, resp.Continue)
			}
		})
	}
}

func TestPipeline_HandleRequest_ChainsContentModifications(t *testing.T) {
	t.Parallel()

	first := &testContentPlugin{rewritePath: "/first"}
	second := &testContentPlugin{rewritePath: "/second"}
	p := newTestContentPipeline(t, first, second)

	resp, err := p.HandleRequest(context.Background(), &HTTPRequest{Path: "/original"})
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, []string{"/original"}, first.seenRequestPaths)
	require.Equal(t, []string{"/first"}, second.seenRequestPaths)

	require.NotNil(t, resp.ModifiedRequest)
	require.Equal(t, "/second", resp.ModifiedRequest.Path)
}

func TestPipeline_HandleRequest_DeliversContentModification(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{rewritePath: "/rewritten"}
	p := newTestContentPipeline(t, content)

	req := &HTTPRequest{
		Method: http.MethodPost,
		Path:   "/original",
		Body:   []byte(`{"original":true}`),
	}

	resp, err := p.HandleRequest(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Continue)

	require.NotNil(t, resp.ModifiedRequest)
	require.Equal(t, "/rewritten", resp.ModifiedRequest.Path)
	require.Equal(t, http.MethodPost, resp.ModifiedRequest.Method)
	require.Equal(t, []byte(`{"original":true}`), resp.ModifiedRequest.Body)
}

func TestPipeline_HandleRequest_UnmodifiedRequestReportsNoModification(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{}
	p := newTestContentPipeline(t, content)

	resp, err := p.HandleRequest(context.Background(), &HTTPRequest{Path: "/original"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Continue)
	require.Nil(t, resp.ModifiedRequest)
}

func TestPipeline_HandleResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plugins        map[config.Category][]*Instance
		response       *HTTPResponse
		expectError    bool
		expectContinue bool
		expectStatus   int32
	}{
		{
			name: "all plugins pass",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowResponse},
						responseResp: &HTTPResponse{
							Continue:   true,
							StatusCode: 200,
						},
					},
					name: "audit-plugin",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowResponse: {}})
				return map[config.Category][]*Instance{
					config.CategoryAudit: {inst},
				}
			}(),
			response: &HTTPResponse{
				Continue:   true,
				StatusCode: 200,
				Body:       []byte("OK"),
			},
			expectError:    false,
			expectContinue: true,
			expectStatus:   200,
		},
		{
			name: "plugin stops pipeline",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowResponse},
						responseResp: &HTTPResponse{
							Continue:   false,
							StatusCode: 500,
						},
					},
					name: "audit-plugin",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowResponse: {}})
				return map[config.Category][]*Instance{
					config.CategoryAudit: {inst},
				}
			}(),
			response: &HTTPResponse{
				Continue:   true,
				StatusCode: 200,
			},
			expectError:    false,
			expectContinue: false,
			expectStatus:   500,
		},
		{
			name: "required plugin fails",
			plugins: func() map[config.Category][]*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowResponse},
						responseErr:  errors.New("plugin error"),
					},
					name:     "audit-plugin",
					required: true,
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowResponse: {}})
				return map[config.Category][]*Instance{
					config.CategoryAudit: {inst},
				}
			}(),
			response: &HTTPResponse{
				Continue:   true,
				StatusCode: 200,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			for category, instances := range tc.plugins {
				for _, instance := range instances {
					err := pipeline.Register(category, instance)
					require.NoError(t, err)
				}
			}

			resp, err := pipeline.HandleResponse(context.Background(), tc.response)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.expectContinue, resp.Continue)
				require.Equal(t, tc.expectStatus, resp.StatusCode)
			}
		})
	}
}

func TestPipeline_HandleResponse_ChainsContentModifications(t *testing.T) {
	t.Parallel()

	first := &testContentPlugin{rewriteStatus: http.StatusCreated, rewriteBody: "first"}
	second := &testContentPlugin{rewriteStatus: http.StatusAccepted, rewriteBody: "second"}
	p := newTestContentPipeline(t, first, second)

	resp, err := p.HandleResponse(context.Background(), &HTTPResponse{
		Continue:   true,
		StatusCode: http.StatusOK,
		Body:       []byte("upstream"),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Equal(t, []string{"upstream"}, first.seenResponseBodies)
	require.Equal(t, []string{"first"}, second.seenResponseBodies)

	require.Equal(t, int32(http.StatusAccepted), resp.StatusCode)
	require.Equal(t, []byte("second"), resp.Body)
}

func TestPipeline_HandleResponse_DeliversContentModification(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{rewriteStatus: http.StatusCreated, rewriteBody: "modified"}
	p := newTestContentPipeline(t, content)

	resp, err := p.HandleResponse(context.Background(), &HTTPResponse{
		Continue:   true,
		StatusCode: http.StatusOK,
		Body:       []byte("upstream"),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Continue)
	require.Equal(t, int32(http.StatusCreated), resp.StatusCode)
	require.Equal(t, []byte("modified"), resp.Body)
}

func TestPipeline_HandleResponse_PassingContentPluginKeepsUpstreamResponse(t *testing.T) {
	t.Parallel()

	content := &testContentPlugin{}
	p := newTestContentPipeline(t, content)

	resp, err := p.HandleResponse(context.Background(), &HTTPResponse{
		Continue:   true,
		StatusCode: http.StatusTeapot,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"tools":[]}`),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Continue)
	require.Equal(t, int32(http.StatusTeapot), resp.StatusCode)
	require.Equal(t, map[string]string{"Content-Type": "application/json"}, resp.Headers)
	require.Equal(t, []byte(`{"tools":[]}`), resp.Body)
}

func TestPipeline_Shutdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		plugins     map[config.Category][]*Instance
		expectError bool
	}{
		{
			name: "all plugins stop successfully",
			plugins: map[config.Category][]*Instance{
				config.CategoryAuthentication: {
					{
						Plugin: &mockPlugin{},
						name:   "auth-plugin",
					},
				},
				config.CategoryObservability: {
					{
						Plugin: &mockPlugin{},
						name:   "obs-plugin",
					},
				},
			},
			expectError: false,
		},
		{
			name: "plugin stop fails",
			plugins: map[config.Category][]*Instance{
				config.CategoryAuthentication: {
					{
						Plugin: &mockPlugin{
							stopErr: errors.New("stop failed"),
						},
						name: "auth-plugin",
					},
				},
			},
			expectError: true,
		},
		{
			name:        "no plugins",
			plugins:     map[config.Category][]*Instance{},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			for category, instances := range tc.plugins {
				for _, instance := range instances {
					err := pipeline.Register(category, instance)
					require.NoError(t, err)
				}
			}

			err := pipeline.Shutdown(context.Background())

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Empty(t, pipeline.plugins)
			}
		})
	}
}

func TestPipeline_FilterByFlow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		instances   []*Instance
		flow        config.Flow
		expectCount int
		expectError bool
	}{
		{
			name: "all plugins support request flow",
			instances: func() []*Instance {
				inst1 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
					},
					name: "plugin-1",
				}
				inst1.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				inst2 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest, config.FlowResponse},
					},
					name: "plugin-2",
				}
				inst2.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}, config.FlowResponse: {}})

				return []*Instance{inst1, inst2}
			}(),
			flow:        config.FlowRequest,
			expectCount: 2,
			expectError: false,
		},
		{
			name: "only some plugins support response flow",
			instances: func() []*Instance {
				inst1 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
					},
					name: "plugin-1",
				}
				inst1.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})

				inst2 := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowResponse},
					},
					name: "plugin-2",
				}
				inst2.SetFlows(map[config.Flow]struct{}{config.FlowResponse: {}})

				return []*Instance{inst1, inst2}
			}(),
			flow:        config.FlowResponse,
			expectCount: 1,
			expectError: false,
		},
		{
			name: "no plugins support flow",
			instances: func() []*Instance {
				inst := &Instance{
					Plugin: &mockPlugin{
						capabilities: []config.Flow{config.FlowRequest},
					},
					name: "plugin-1",
				}
				inst.SetFlows(map[config.Flow]struct{}{config.FlowRequest: {}})
				return []*Instance{inst}
			}(),
			flow:        config.FlowResponse,
			expectCount: 0,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pipeline := newPipeline(hclog.NewNullLogger())
			filtered, err := pipeline.filterByFlow(context.Background(), tc.instances, tc.flow)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, filtered, tc.expectCount)
			}
		})
	}
}

func TestPipeline_CategoryOrdering(t *testing.T) {
	t.Parallel()

	var executionOrder []string
	var mu sync.Mutex

	// Create a tracking plugin that appends to executionOrder when executed.
	createTrackingPlugin := func(name string, flow config.Flow) *Instance {
		instance := &Instance{
			Plugin: &testTrackingPlugin{
				capabilities: []config.Flow{flow},
				onRequest: func() {
					mu.Lock()
					executionOrder = append(executionOrder, name)
					mu.Unlock()
				},
			},
			name: name,
		}
		instance.SetFlows(map[config.Flow]struct{}{flow: {}})
		return instance
	}

	pipeline := newPipeline(hclog.NewNullLogger())

	// Register plugins in all 7 categories.
	categories := []config.Category{
		config.CategoryAuthentication,
		config.CategoryAuthorization,
		config.CategoryRateLimiting,
		config.CategoryValidation,
		config.CategoryContent,
		config.CategoryAudit,
		config.CategoryObservability,
	}

	for _, category := range categories {
		plugin := createTrackingPlugin(string(category), config.FlowRequest)
		err := pipeline.Register(category, plugin)
		require.NoError(t, err)
	}

	// Execute pipeline.
	_, err := pipeline.HandleRequest(context.Background(), &HTTPRequest{
		Method: "GET",
		Path:   "/test",
	})
	require.NoError(t, err)

	// Verify execution order matches category order.
	require.Len(t, executionOrder, len(categories))

	// Map execution order to expected category order from the source of truth.
	expected := config.OrderedCategories()
	for i, categoryName := range executionOrder {
		expectedCategory := expected[i]
		require.Equal(t, string(expectedCategory), categoryName,
			"plugin at position %d should be from category %s", i, expectedCategory)
	}
}
