package plugin

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// GRPCAdapter adapts a gRPC plugin client to the Plugin interface.
type GRPCAdapter struct {
	client  pluginv1.PluginClient
	timeout time.Duration
}

// NewGRPCAdapter creates a new gRPC plugin adapter.
func NewGRPCAdapter(client pluginv1.PluginClient, timeout time.Duration) (*GRPCAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("plugin client cannot be nil")
	}

	return &GRPCAdapter{
		client:  client,
		timeout: timeout,
	}, nil
}

// Capabilities returns the flows this plugin supports.
func (a *GRPCAdapter) Capabilities(ctx context.Context) ([]config.Flow, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	caps, err := a.client.GetCapabilities(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	flows := make([]config.Flow, 0, len(caps.Flows))
	for _, f := range caps.Flows {
		switch f {
		case pluginv1.FlowRequest:
			flows = append(flows, config.FlowRequest)
		case pluginv1.FlowResponse:
			flows = append(flows, config.FlowResponse)
		default:
			return nil, fmt.Errorf("unknown flow: %v", f)
		}
	}

	return flows, nil
}

// CheckHealth verifies the plugin is healthy.
func (a *GRPCAdapter) CheckHealth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	_, err := a.client.CheckHealth(ctx, &emptypb.Empty{})
	return err
}

// CheckReady verifies the plugin is ready to handle requests.
func (a *GRPCAdapter) CheckReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	_, err := a.client.CheckReady(ctx, &emptypb.Empty{})
	return err
}

// Configure sends configuration to the plugin.
func (a *GRPCAdapter) Configure(ctx context.Context, cfg *pluginv1.PluginConfig) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	_, err := a.client.Configure(ctx, cfg)
	return err
}

// HandleRequest processes a request.
func (a *GRPCAdapter) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	grpcReq := &pluginv1.HTTPRequest{
		Method:  req.Method,
		Path:    req.Path,
		Headers: req.Headers,
		Body:    req.Body,
	}

	grpcResp, err := a.client.HandleRequest(ctx, grpcReq)
	if err != nil {
		return nil, err
	}

	resp := &HTTPResponse{
		Continue:   grpcResp.Continue,
		StatusCode: grpcResp.StatusCode,
		Headers:    grpcResp.Headers,
		Body:       grpcResp.Body,
	}

	if grpcResp.ModifiedRequest != nil {
		resp.ModifiedRequest = &HTTPRequest{
			Method:  grpcResp.ModifiedRequest.Method,
			Path:    grpcResp.ModifiedRequest.Path,
			Headers: grpcResp.ModifiedRequest.Headers,
			Body:    grpcResp.ModifiedRequest.Body,
		}
	}

	return resp, nil
}

// HandleResponse processes a response.
func (a *GRPCAdapter) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	grpcResp := &pluginv1.HTTPResponse{
		Continue:   resp.Continue,
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}

	grpcResult, err := a.client.HandleResponse(ctx, grpcResp)
	if err != nil {
		return nil, err
	}

	result := &HTTPResponse{
		Continue:   grpcResult.Continue,
		StatusCode: grpcResult.StatusCode,
		Headers:    grpcResult.Headers,
		Body:       grpcResult.Body,
	}

	if grpcResult.ModifiedRequest != nil {
		result.ModifiedRequest = &HTTPRequest{
			Method:  grpcResult.ModifiedRequest.Method,
			Path:    grpcResult.ModifiedRequest.Path,
			Headers: grpcResult.ModifiedRequest.Headers,
			Body:    grpcResult.ModifiedRequest.Body,
		}
	}

	return result, nil
}

// Stop performs graceful shutdown.
func (a *GRPCAdapter) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	_, err := a.client.Stop(ctx, &emptypb.Empty{})
	return err
}
