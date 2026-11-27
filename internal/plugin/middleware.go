package plugin

import (
	"bytes"
	"io"
	"net/http"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/internal/api"
)

const (
	// maxPluginStatusCode is the highest status code a plugin may report.
	maxPluginStatusCode = 599

	// minPluginStatusCode is the lowest status code a plugin may report.
	minPluginStatusCode = 200
)

// Middleware returns an HTTP middleware function that processes requests through the plugin pipeline.
// The middleware intercepts requests, runs them through configured plugins, and handles responses.
func (p *pipeline) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Convert request.
			httpReq, err := httpRequestToPlugin(r)
			if err != nil {
				http.Error(w, "Failed to process request", http.StatusInternalServerError)
				p.logger.Error("failed to convert http request", "error", err)
				return
			}

			// Run pipeline for request flow.
			httpResp, err := p.HandleRequest(ctx, httpReq)
			if err != nil {
				// Pipeline error (required plugin failed, or infrastructure error).
				w.Header().Set(api.HeaderErrorType, string(api.PipelineRequestFailure))
				http.Error(w, "Request processing failed", http.StatusInternalServerError)
				p.logger.Error("pipeline request flow failed", "error", err)
				return
			}

			// Check if plugin short-circuited (business logic rejection).
			if !httpResp.Continue {
				// Plugin wants to respond directly (e.g., 429 rate limit, 403 auth failed).
				writePluginResponse(p.logger, w, httpResp)
				return
			}

			// Continue to actual handler (capture response).
			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, applyModifiedRequest(r, httpResp.ModifiedRequest))

			// Convert handler response.
			handlerResp := &HTTPResponse{
				StatusCode: int32(recorder.statusCode),
				Headers:    convertHeadersToMap(recorder.Header()),
				Body:       recorder.body.Bytes(),
				Continue:   true,
			}

			// Run pipeline for response flow.
			finalResp, err := p.HandleResponse(ctx, handlerResp)
			if err != nil {
				// Pipeline error in response flow.
				w.Header().Set(api.HeaderErrorType, string(api.PipelineResponseFailure))
				http.Error(w, "Response processing failed", http.StatusInternalServerError)
				p.logger.Error("pipeline response flow failed", "error", err)
				return
			}

			// No plugin modified the response, so the recorder's own result is what the client gets.
			// Rewriting the headers here would project them through a map that holds one value per
			// name, which drops a second Set-Cookie the handler set.
			if finalResp == handlerResp {
				w.WriteHeader(recorder.statusCode)

				if recorder.body.Len() > 0 {
					_, _ = w.Write(recorder.body.Bytes())
				}

				return
			}

			// Write final (potentially modified) response.
			writePluginResponse(p.logger, w, finalResp)
		})
	}
}

// applyModifiedRequest returns a copy of r carrying the headers and body that mod sets.
// It returns r itself when mod is nil.
//
// NOTE: The method and the path in mod are ignored, because routing has not happened yet.
// Honoring them would let a plugin choose the route and so the handler, after the categories that gate
// access have already passed on the request the client sent.
// Headers are merged over the request, so a plugin replaces only the headers it sets and cannot delete one.
// A reserved header is skipped.
// An empty body leaves the request body unchanged, because proto3 omits an empty bytes field from the
// wire, so a plugin that sent one would be indistinguishable from a plugin that sent none.
func applyModifiedRequest(r *http.Request, mod *HTTPRequest) *http.Request {
	if mod == nil {
		return r
	}

	out := r.Clone(r.Context())

	for name, value := range mod.Headers {
		if isReservedHeader(name) {
			continue
		}

		out.Header.Set(name, value)
	}

	if len(mod.Body) != 0 {
		out.Body = io.NopCloser(bytes.NewReader(mod.Body))
		out.ContentLength = int64(len(mod.Body))
		out.Header.Del("Content-Length")
	}

	return out
}

// pluginStatusCode returns the status code to write for a plugin response.
// A code a plugin may not report becomes a server error.
//
// NOTE: A plugin that stops the pipeline without reporting a status code reaches here with zero, which
// must not become a success. net/http writes an implicit 200 on the first body write, so the client
// would read a rejection as a successful call.
func pluginStatusCode(code int32) int {
	status := int(code)
	if status < minPluginStatusCode || status > maxPluginStatusCode {
		return http.StatusInternalServerError
	}

	return status
}

// httpRequestToPlugin converts *http.Request → *HTTPRequest.
func httpRequestToPlugin(r *http.Request) (*HTTPRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body)) // Restore body for downstream handlers.

	// Convert headers (take first value only).
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &HTTPRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: headers,
		Body:    body,
	}, nil
}

// writePluginResponse writes a HTTPResponse.
//
// NOTE: A reserved header in resp is skipped, so this is safe for a response a plugin authored itself.
// A status code a plugin cannot report is written as a server error, because mcpd cannot know what the
// plugin meant and net/http panics on a code it does not accept.
func writePluginResponse(logger hclog.Logger, w http.ResponseWriter, resp *HTTPResponse) {
	if reserved := reservedHeaderNames(resp.Headers); len(reserved) > 0 {
		logger.Warn("ignoring reserved response headers set by plugin", "headers", reserved)
	}

	for name, value := range resp.Headers {
		if isReservedHeader(name) {
			continue
		}

		w.Header().Set(name, value)
	}

	status := pluginStatusCode(resp.StatusCode)
	if status != int(resp.StatusCode) {
		logger.Warn("replacing a status code a plugin may not report",
			"reported", resp.StatusCode, "written", status)
	}

	w.WriteHeader(status)

	if len(resp.Body) > 0 {
		_, _ = w.Write(resp.Body)
	}
}

// convertHeadersToMap converts http.Header → map[string]string (first value only).
func convertHeadersToMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

// responseRecorder captures the response from the next handler.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

// newResponseRecorder creates a new responseRecorder.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status.
	}
}

// WriteHeader captures the status code.
func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
}

// Write captures the response body.
func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}
