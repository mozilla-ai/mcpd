package plugin

import (
	"bytes"
	"io"
	"net/http"

	"github.com/mozilla-ai/mcpd/internal/api"
)

const (
	// maxHTTPStatusCode is the highest status code net/http accepts.
	maxHTTPStatusCode = 999

	// minHTTPStatusCode is the lowest status code net/http accepts.
	minHTTPStatusCode = 100
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
				writePluginResponse(w, httpResp)
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

			// Write final (potentially modified) response.
			writePluginResponse(w, finalResp)
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
// A plugin clears the body with an empty slice rather than a nil one.
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

	if mod.Body != nil {
		out.Body = io.NopCloser(bytes.NewReader(mod.Body))
		out.ContentLength = int64(len(mod.Body))
		out.Header.Del("Content-Length")
	}

	return out
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
// NOTE: A status code outside the range HTTP defines is written as a server error.
// net/http panics on one, and a plugin reports its own status code.
func writePluginResponse(w http.ResponseWriter, resp *HTTPResponse) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}

	if resp.StatusCode > 0 {
		status := int(resp.StatusCode)
		if status < minHTTPStatusCode || status > maxHTTPStatusCode {
			status = http.StatusInternalServerError
		}

		w.WriteHeader(status)
	}

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
