package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowedHosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addr     string
		expected []string
	}{
		{
			name:     "loopback IPv4 bind",
			addr:     "127.0.0.1:8090",
			expected: []string{"127.0.0.1", "::1", "localhost"},
		},
		{
			name:     "loopback name bind as used by dev mode",
			addr:     "localhost:8090",
			expected: []string{"127.0.0.1", "::1", "localhost"},
		},
		{
			name:     "loopback IPv6 bind",
			addr:     "[::1]:8090",
			expected: []string{"127.0.0.1", "::1", "localhost"},
		},
		{
			name:     "alternate loopback IPv4 bind",
			addr:     "127.0.0.2:8090",
			expected: []string{"127.0.0.1", "::1", "localhost", "127.0.0.2"},
		},
		{
			name:     "all interfaces permits any host",
			addr:     "0.0.0.0:8090",
			expected: nil,
		},
		{
			name:     "empty host permits any host",
			addr:     ":8090",
			expected: nil,
		},
		{
			name:     "routable address permits any host",
			addr:     "192.168.1.5:8090",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, allowedHosts(tc.addr))
		})
	}
}

func TestOwnOrigins(t *testing.T) {
	t.Parallel()

	loopback := []string{"http://127.0.0.1:8090", "http://[::1]:8090", "http://localhost:8090"}

	tests := []struct {
		name     string
		addr     string
		expected []string
	}{
		{
			name:     "loopback bind is already covered",
			addr:     "127.0.0.1:8090",
			expected: loopback,
		},
		{
			name:     "IPv6 loopback bind is already covered",
			addr:     "[::1]:8090",
			expected: loopback,
		},
		{
			name:     "alternate loopback bind is added",
			addr:     "127.0.0.2:8090",
			expected: append(append([]string{}, loopback...), "http://127.0.0.2:8090"),
		},
		{
			name:     "routable bind is added",
			addr:     "192.168.1.5:8090",
			expected: append(append([]string{}, loopback...), "http://192.168.1.5:8090"),
		},
		{
			name:     "all interfaces bind is added",
			addr:     "0.0.0.0:8090",
			expected: append(append([]string{}, loopback...), "http://0.0.0.0:8090"),
		},
		{
			name:     "empty host has nothing to add",
			addr:     ":8090",
			expected: loopback,
		},
		{
			name:     "an address without a port has no origins",
			addr:     "nonsense",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, ownOrigins(tc.addr))
		})
	}
}

func TestRequireAllowedHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		allowed        []string
		host           string
		expectedStatus int
		expectReached  bool
	}{
		{
			name:           "empty allowlist permits any host",
			allowed:        nil,
			host:           "mcpd:8090",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "loopback name is served",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "localhost:8090",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "loopback address is served",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "127.0.0.1:8090",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "bracketed IPv6 loopback is served",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "[::1]:8090",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "host without a port is served",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "localhost",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "rebinding host is refused",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "evil.example:8090",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
		{
			name:           "service name is refused on a loopback bind",
			allowed:        []string{"127.0.0.1", "::1", "localhost"},
			host:           "mcpd:8090",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reached := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
			req.Host = tc.host

			rec := httptest.NewRecorder()
			requireAllowedHost(tc.allowed)(next).ServeHTTP(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)
			require.Equal(t, tc.expectReached, reached)
		})
	}
}

func TestRequireAllowedOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		allowed        []string
		method         string
		origin         string
		expectedStatus int
		expectReached  bool
	}{
		{
			name:           "no origin header passes",
			allowed:        nil,
			method:         http.MethodPost,
			origin:         "",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "no origin header passes with an allowlist set",
			allowed:        []string{"https://app.example.com"},
			method:         http.MethodPost,
			origin:         "",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "allowed origin passes",
			allowed:        []string{"https://app.example.com"},
			method:         http.MethodPost,
			origin:         "https://app.example.com",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "allowed origin matches regardless of case",
			allowed:        []string{"https://App.Example.com"},
			method:         http.MethodPost,
			origin:         "https://app.example.com",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "subdomain wildcard matches",
			allowed:        []string{"https://*.example.com"},
			method:         http.MethodPost,
			origin:         "https://app.example.com",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "subdomain wildcard does not match another host",
			allowed:        []string{"https://*.example.com"},
			method:         http.MethodPost,
			origin:         "https://evil.example",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
		{
			name:           "unknown origin is refused",
			allowed:        []string{"https://app.example.com"},
			method:         http.MethodPost,
			origin:         "http://evil.example",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
		{
			name:           "empty allowlist refuses any origin",
			allowed:        nil,
			method:         http.MethodPost,
			origin:         "http://evil.example",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
		{
			name:           "wildcard allowlist allows any origin",
			allowed:        []string{"*"},
			method:         http.MethodPost,
			origin:         "http://evil.example",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "read methods are covered",
			allowed:        nil,
			method:         http.MethodGet,
			origin:         "http://evil.example",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
		{
			name:           "rebinding origin on the daemon port is refused",
			allowed:        nil,
			method:         http.MethodPost,
			origin:         "http://evil.example:8090",
			expectedStatus: http.StatusForbidden,
			expectReached:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reached := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, "/api/v1/servers/notes/tools/write_note", strings.NewReader(`{}`))
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			rec := httptest.NewRecorder()
			requireAllowedOrigin(tc.allowed)(next).ServeHTTP(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)
			require.Equal(t, tc.expectReached, reached)
		})
	}
}

func TestRequireJSONContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		expectReached  bool
	}{
		{
			name:           "POST with JSON content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"a":1}`,
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "POST with JSON content type and charset",
			method:         http.MethodPost,
			contentType:    "application/json; charset=utf-8",
			body:           `{"a":1}`,
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "POST without content type",
			method:         http.MethodPost,
			contentType:    "",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "POST with text content type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "POST with form content type",
			method:         http.MethodPost,
			contentType:    "multipart/form-data; boundary=x",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "POST with malformed content type",
			method:         http.MethodPost,
			contentType:    "application/json; charset",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "PUT without content type",
			method:         http.MethodPut,
			contentType:    "",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "PATCH without content type",
			method:         http.MethodPatch,
			contentType:    "",
			body:           `{"a":1}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectReached:  false,
		},
		{
			name:           "GET without content type",
			method:         http.MethodGet,
			contentType:    "",
			body:           "",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "DELETE without content type",
			method:         http.MethodDelete,
			contentType:    "",
			body:           "",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
		{
			name:           "OPTIONS preflight without content type",
			method:         http.MethodOptions,
			contentType:    "",
			body:           "",
			expectedStatus: http.StatusOK,
			expectReached:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reached := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, "/api/v1/servers/notes/tools/write_note", strings.NewReader(tc.body))
			req.Header.Del("Content-Type")
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			rec := httptest.NewRecorder()
			requireJSONContentType(next).ServeHTTP(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)
			require.Equal(t, tc.expectReached, reached)
		})
	}
}

func TestRequireJSONContentType_RejectionBody(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/notes/tools/write_note", strings.NewReader(`{}`))
	req.Header.Del("Content-Type")

	rec := httptest.NewRecorder()
	requireJSONContentType(next).ServeHTTP(rec, req)

	require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(http.StatusUnsupportedMediaType), body["status"])
	require.NotEmpty(t, body["detail"])
}
