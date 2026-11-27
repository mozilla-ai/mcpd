package plugin

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1"
)

// stubPluginClient is a PluginClient that records what it was sent and returns a canned result.
// A nil result means the plugin returns a response that changes nothing.
type stubPluginClient struct {
	sentRequest  *pluginv1.HTTPRequest
	sentResponse *pluginv1.HTTPResponse

	// echoRequestHeaders returns the request headers the plugin was given as a modified request.
	echoRequestHeaders bool

	// echoResponseHeaders returns the response headers the plugin was given.
	echoResponseHeaders bool

	// extraRequestHeaders are added to the modified request the plugin returns.
	extraRequestHeaders map[string]string

	// extraResponseHeaders are added to the response the plugin returns.
	extraResponseHeaders map[string]string

	// rewriteRequestBody is returned as the modified request body when it is not empty.
	rewriteRequestBody string

	// stopWithHeaders is returned as a stopping plugin's own response headers when it is not nil.
	stopWithHeaders map[string]string
}

func (s *stubPluginClient) Configure(
	_ context.Context,
	_ *pluginv1.PluginConfig,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *stubPluginClient) Stop(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *stubPluginClient) GetMetadata(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.Metadata, error) {
	return &pluginv1.Metadata{}, nil
}

func (s *stubPluginClient) GetCapabilities(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.Capabilities, error) {
	return &pluginv1.Capabilities{
		Flows: []pluginv1.Flow{pluginv1.FlowRequest, pluginv1.FlowResponse},
	}, nil
}

func (s *stubPluginClient) CheckHealth(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *stubPluginClient) CheckReady(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *stubPluginClient) HandleRequest(
	_ context.Context,
	in *pluginv1.HTTPRequest,
	_ ...grpc.CallOption,
) (*pluginv1.HTTPResponse, error) {
	s.sentRequest = in

	if s.stopWithHeaders != nil {
		return &pluginv1.HTTPResponse{
			Continue:   false,
			StatusCode: http.StatusForbidden,
			Headers:    s.stopWithHeaders,
			Body:       []byte("denied"),
		}, nil
	}

	out := &pluginv1.HTTPResponse{Continue: true}
	if !s.echoRequestHeaders && s.extraRequestHeaders == nil && s.rewriteRequestBody == "" {
		return out, nil
	}

	mod := &pluginv1.HTTPRequest{Method: in.Method, Path: in.Path, Body: in.Body}
	if s.rewriteRequestBody != "" {
		mod.Body = []byte(s.rewriteRequestBody)
	}

	mod.Headers = make(map[string]string, len(in.Headers)+len(s.extraRequestHeaders))
	if s.echoRequestHeaders {
		for name, value := range in.Headers {
			mod.Headers[name] = value
		}
	}
	for name, value := range s.extraRequestHeaders {
		mod.Headers[name] = value
	}
	out.ModifiedRequest = mod

	return out, nil
}

func (s *stubPluginClient) HandleResponse(
	_ context.Context,
	in *pluginv1.HTTPResponse,
	_ ...grpc.CallOption,
) (*pluginv1.HTTPResponse, error) {
	s.sentResponse = in

	if s.stopWithHeaders != nil {
		return &pluginv1.HTTPResponse{
			Continue:   false,
			StatusCode: http.StatusForbidden,
			Headers:    s.stopWithHeaders,
			Body:       []byte("denied"),
		}, nil
	}

	out := &pluginv1.HTTPResponse{Continue: true}
	if !s.echoResponseHeaders && s.extraResponseHeaders == nil {
		return out, nil
	}

	out.Headers = make(map[string]string, len(in.Headers)+len(s.extraResponseHeaders))
	if s.echoResponseHeaders {
		for name, value := range in.Headers {
			out.Headers[name] = value
		}
	}
	for name, value := range s.extraResponseHeaders {
		out.Headers[name] = value
	}

	return out, nil
}

func newTestAdapter(t *testing.T, client *stubPluginClient) *GRPCAdapter {
	t.Helper()

	adapter, err := NewGRPCAdapter(client, time.Second)
	require.NoError(t, err)

	return adapter
}

// TestGRPCAdapter_HandleRequest_SendsOneValuePerHeader documents the wire format's limit, which is the
// reason an echo cannot be read as a request to change a header.
func TestGRPCAdapter_HandleRequest_SendsOneValuePerHeader(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{}
	adapter := newTestAdapter(t, client)

	_, err := adapter.HandleRequest(context.Background(), &HTTPRequest{
		Method:  http.MethodPost,
		Path:    "/original",
		Headers: http.Header{"Accept": {"application/json", "text/event-stream"}},
	})
	require.NoError(t, err)

	require.Equal(t, "application/json", client.sentRequest.Headers["Accept"])
}

// TestGRPCAdapter_HandleRequest_EchoedHeaderIsNotAChange pins the rule that a plugin returning the
// headers it was given has asked for no header change.
func TestGRPCAdapter_HandleRequest_EchoedHeaderIsNotAChange(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{echoRequestHeaders: true, rewriteRequestBody: `{"rewritten":true}`}
	adapter := newTestAdapter(t, client)

	resp, err := adapter.HandleRequest(context.Background(), &HTTPRequest{
		Method:  http.MethodPost,
		Path:    "/original",
		Headers: http.Header{"Accept": {"application/json", "text/event-stream"}},
	})
	require.NoError(t, err)

	require.Equal(t, []byte(`{"rewritten":true}`), resp.ModifiedRequest.Body)
	require.Empty(t, resp.ModifiedRequest.Headers)
}

// TestGRPCAdapter_HandleRequest_ReportsOnlyAChangedHeader pins that a plugin's real change survives
// alongside the headers it echoed.
func TestGRPCAdapter_HandleRequest_ReportsOnlyAChangedHeader(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{
		echoRequestHeaders:  true,
		extraRequestHeaders: map[string]string{"X-Added": "yes", "Accept": "text/plain"},
	}
	adapter := newTestAdapter(t, client)

	resp, err := adapter.HandleRequest(context.Background(), &HTTPRequest{
		Method:  http.MethodPost,
		Path:    "/original",
		Headers: http.Header{"Accept": {"application/json"}, "X-Kept": {"same"}},
	})
	require.NoError(t, err)

	require.Equal(t, "yes", resp.ModifiedRequest.Headers.Get("X-Added"))
	require.Equal(t, "text/plain", resp.ModifiedRequest.Headers.Get("Accept"))
	require.Empty(t, resp.ModifiedRequest.Headers.Get("X-Kept"))
}

// TestGRPCAdapter_HandleRequest_StoppingPluginKeepsItsOwnHeaders pins that a plugin which authors a
// response is not treated as echoing a request it was given.
func TestGRPCAdapter_HandleRequest_StoppingPluginKeepsItsOwnHeaders(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{stopWithHeaders: map[string]string{"Retry-After": "30"}}
	adapter := newTestAdapter(t, client)

	resp, err := adapter.HandleRequest(context.Background(), &HTTPRequest{
		Method:  http.MethodGet,
		Path:    "/original",
		Headers: http.Header{"Retry-After": {"30"}},
	})
	require.NoError(t, err)

	require.False(t, resp.Continue)
	require.Equal(t, "30", resp.Headers.Get("Retry-After"))
}

// TestGRPCAdapter_HandleResponse_EchoedHeaderIsNotAChange pins the echo rule on the response flow.
func TestGRPCAdapter_HandleResponse_EchoedHeaderIsNotAChange(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{echoResponseHeaders: true}
	adapter := newTestAdapter(t, client)

	result, err := adapter.HandleResponse(context.Background(), &HTTPResponse{
		Continue:   true,
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Length": {"3"}, "Set-Cookie": {"a=1", "b=2"}},
	})
	require.NoError(t, err)

	require.Empty(t, result.Headers)
}

// TestGRPCAdapter_HandleResponse_StoppingPluginKeepsItsOwnHeaders pins that a plugin which authors a
// response keeps the headers it set, rather than having them read as an echo.
func TestGRPCAdapter_HandleResponse_StoppingPluginKeepsItsOwnHeaders(t *testing.T) {
	t.Parallel()

	client := &stubPluginClient{stopWithHeaders: map[string]string{"Content-Type": "text/plain"}}
	adapter := newTestAdapter(t, client)

	result, err := adapter.HandleResponse(context.Background(), &HTTPResponse{
		Continue:   true,
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Type": {"text/plain"}},
	})
	require.NoError(t, err)

	require.False(t, result.Continue)
	require.Equal(t, "text/plain", result.Headers.Get("Content-Type"))
}
