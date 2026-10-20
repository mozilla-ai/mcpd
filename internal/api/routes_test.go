package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// registeredOperations maps every operation ID in the spec to its method and path.
func registeredOperations(t *testing.T, spec *huma.OpenAPI) map[string]string {
	t.Helper()

	ops := make(map[string]string)

	for _, item := range spec.Paths {
		for _, op := range []*huma.Operation{
			item.Delete,
			item.Get,
			item.Head,
			item.Options,
			item.Patch,
			item.Post,
			item.Put,
			item.Trace,
		} {
			if op == nil {
				continue
			}

			ops[op.OperationID] = op.Method + " " + op.Path
		}
	}

	return ops
}

func TestRegisterRoutes_Operations(t *testing.T) {
	t.Parallel()

	mux := chi.NewMux()
	router := humachi.New(mux, huma.DefaultConfig("mcpd docs", APIVersion))

	apiPathPrefix, err := RegisterRoutes(router, newMockHealthMonitor(), newMockMCPClientAccessor())
	require.NoError(t, err)
	require.Equal(t, "/api/v1", apiPathPrefix)

	expected := map[string]string{
		"callTool":              "POST /api/v1/servers/{server}/tools/{tool}",
		"generatePrompt":        "POST /api/v1/servers/{name}/prompts/{promptName}",
		"getResourceContent":    "GET /api/v1/servers/{name}/resources/content",
		"getServerHealth":       "GET /api/v1/health/servers/{name}",
		"listPrompts":           "GET /api/v1/servers/{name}/prompts",
		"listResourceTemplates": "GET /api/v1/servers/{name}/resources/templates",
		"listResources":         "GET /api/v1/servers/{name}/resources",
		"listServers":           "GET /api/v1/servers",
		"listServersHealth":     "GET /api/v1/health/servers",
		"listTools":             "GET /api/v1/servers/{name}/tools",
	}

	require.Equal(t, expected, registeredOperations(t, router.OpenAPI()))
}

func TestRegisterRoutes_ServesRegisteredPath(t *testing.T) {
	t.Parallel()

	mux := chi.NewMux()
	router := humachi.New(mux, huma.DefaultConfig("mcpd docs", APIVersion))

	_, err := RegisterRoutes(router, newMockHealthMonitor(), newMockMCPClientAccessor())
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil))

	require.Equal(t, http.StatusOK, rec.Code)
}
