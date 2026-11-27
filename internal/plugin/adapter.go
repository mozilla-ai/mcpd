package plugin

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1"

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
		Headers: flattenHeaders(req.Headers),
		Body:    req.Body,
	}

	grpcResp, err := a.client.HandleRequest(ctx, grpcReq)
	if err != nil {
		return nil, err
	}

	resp := &HTTPResponse{
		Continue:   grpcResp.Continue,
		StatusCode: grpcResp.StatusCode,
		Headers:    expandHeaders(grpcResp.Headers),
		Body:       grpcResp.Body,
	}

	if grpcResp.ModifiedRequest != nil {
		resp.ModifiedRequest = &HTTPRequest{
			Method:  grpcResp.ModifiedRequest.Method,
			Path:    grpcResp.ModifiedRequest.Path,
			Headers: changedHeaders(grpcReq.Headers, grpcResp.ModifiedRequest.Headers),
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
		Headers:    flattenHeaders(resp.Headers),
		Body:       resp.Body,
	}

	grpcResult, err := a.client.HandleResponse(ctx, grpcResp)
	if err != nil {
		return nil, err
	}

	// A plugin that continues the pipeline is patching the response it was given, so an echoed header
	// is not a change. A plugin that stops the pipeline authored this response itself.
	var headers http.Header
	if grpcResult.Continue {
		headers = changedHeaders(grpcResp.Headers, grpcResult.Headers)
	} else {
		headers = expandHeaders(grpcResult.Headers)
	}

	result := &HTTPResponse{
		Continue:   grpcResult.Continue,
		StatusCode: grpcResult.StatusCode,
		Headers:    headers,
		Body:       grpcResult.Body,
	}

	if grpcResult.ModifiedRequest != nil {
		result.ModifiedRequest = &HTTPRequest{
			Method:  grpcResult.ModifiedRequest.Method,
			Path:    grpcResult.ModifiedRequest.Path,
			Headers: expandHeaders(grpcResult.ModifiedRequest.Headers),
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

// flattenHeaders projects h onto the plugin wire format, which carries one value per name.
//
// NOTE: A name's remaining values are dropped, so a plugin cannot see them.
func flattenHeaders(h http.Header) map[string]string {
	if len(h) == 0 {
		return nil
	}

	out := make(map[string]string, len(h))
	for name, values := range h {
		if len(values) > 0 {
			out[name] = values[0]
		}
	}

	return out
}

// expandHeaders returns the plugin wire format as an http.Header holding canonical names.
// Names are read in sorted order, so two spellings of one name resolve to the same value on every call.
func expandHeaders(m map[string]string) http.Header {
	if len(m) == 0 {
		return nil
	}

	out := make(http.Header, len(m))
	for _, name := range slices.Sorted(maps.Keys(m)) {
		out.Set(name, m[name])
	}

	return out
}

// changedHeaders returns the entries in out whose value differs from the one sent in, as an http.Header
// holding canonical names. It returns nil when the plugin asked for no header change.
//
// NOTE: An entry matching what was sent is dropped. The wire format carries one value per name, so a
// plugin echoing that value has not asked mcpd to discard the values the plugin never saw.
func changedHeaders(sent map[string]string, out map[string]string) http.Header {
	var changed http.Header

	for _, name := range slices.Sorted(maps.Keys(out)) {
		if value, ok := sent[http.CanonicalHeaderKey(name)]; ok && value == out[name] {
			continue
		}

		if changed == nil {
			changed = make(http.Header, len(out))
		}

		changed.Set(name, out[name])
	}

	return changed
}
