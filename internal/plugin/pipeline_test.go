package plugin

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
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
