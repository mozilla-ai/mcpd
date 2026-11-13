package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
