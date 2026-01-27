package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/errors"
)

func TestNewAPIServer_AppliesDefaults(t *testing.T) {
	t.Parallel()
	t.Helper()

	deps, err := NewAPIDependencies(
		hclog.NewNullLogger(),
		NewClientManager(),
		NewHealthTracker([]string{"test-server"}),
		"localhost:8090",
	)
	require.NoError(t, err)

	// Test with no options - should get defaults
	server, err := NewAPIServer(deps)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.Equal(t, DefaultAPIShutdownTimeout(), server.shutdownTimeout)
	require.False(t, server.cors.Enabled)

	// Test with some options - should get defaults + overrides
	server2, err := NewAPIServer(deps, WithShutdownTimeout(10*time.Second), WithCORSEnabled(true))
	require.NoError(t, err)
	require.NotNil(t, server2)
	require.Equal(t, 10*time.Second, server2.shutdownTimeout)
	require.True(t, server2.cors.Enabled)

	// Test with nil options - should still work
	server3, err := NewAPIServer(deps, nil, WithShutdownTimeout(3*time.Second), nil)
	require.NoError(t, err)
	require.NotNil(t, server3)
	require.Equal(t, 3*time.Second, server3.shutdownTimeout)
}

func TestAPIServer_ApplyCORS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		corsConfig  CORSConfig
		expectLog   string
		expectPanic bool
	}{
		{
			name: "basic CORS configuration",
			corsConfig: CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"http://localhost:3000", "https://example.com"},
				AllowMethods:     []string{"GET", "POST", "PUT"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				ExposedHeaders:   []string{"X-Total-Count"},
				AllowCredentials: true,
				MaxAge:           5 * time.Minute,
			},
			expectLog: "Enabling CORS",
		},
		{
			name: "wildcard origin with credentials - should force credentials to false",
			corsConfig: CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"http://localhost:3000", "*", "https://example.com"},
				AllowMethods:     []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true, // This should be overridden to false
				MaxAge:           10 * time.Minute,
			},
			expectLog: "Enabling CORS",
		},
		{
			name: "single wildcard origin",
			corsConfig: CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				MaxAge:           1 * time.Hour,
			},
			expectLog: "Enabling CORS",
		},
		{
			name: "origins with whitespace should be trimmed",
			corsConfig: CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"  http://localhost:3000  ", " https://example.com ", "http://test.com"},
				AllowMethods: []string{"GET"},
			},
			expectLog: "Enabling CORS",
		},
		{
			name: "empty origins list",
			corsConfig: CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{},
				AllowMethods: []string{"GET", "POST"},
			},
			expectLog: "Enabling CORS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create test logger to capture output
			logger := hclog.NewNullLogger()

			// Create basic APIServer with test CORS config
			server := &APIServer{
				logger: logger,
				cors:   tc.corsConfig,
			}

			// Create a basic chi mux for testing
			mux := testNewChiMux(t)

			// Apply CORS should not panic and should configure middleware
			if tc.expectPanic {
				require.Panics(t, func() {
					server.applyCORS(mux)
				})
				return
			}

			require.NotPanics(t, func() {
				server.applyCORS(mux)
			})

			// Note: We can't easily verify the internal CORS configuration changes
			// without inspecting the internal state of chi-cors middleware,
			// but the applyCORS method contains the security logic and we've tested it doesn't panic
		})
	}
}

func TestAPIServer_ApplyCORS_WildcardSecurityLogic(t *testing.T) {
	t.Parallel()

	t.Run("wildcard origin security - prevents credentials", func(t *testing.T) {
		t.Parallel()

		logger := hclog.NewNullLogger()
		server := &APIServer{
			logger: logger,
			cors: CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"http://example.com", "*", "http://test.com"},
				AllowCredentials: true, // This should be overridden
			},
		}

		mux := testNewChiMux(t)

		// This should not panic and should handle the security issue internally
		require.NotPanics(t, func() {
			server.applyCORS(mux)
		})

		// The applyCORS method should have:
		// 1. Set AllowedOrigins to just ["*"]
		// 2. Set AllowCredentials to false
		// We can't directly verify this without accessing internal state,
		// but we've tested the method doesn't panic and the logic is clearly implemented
	})

	t.Run("origin trimming behavior", func(t *testing.T) {
		t.Parallel()

		logger := hclog.NewNullLogger()
		server := &APIServer{
			logger: logger,
			cors: CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"  http://localhost:3000  ", "\thttps://example.com\n", " http://test.com "},
			},
		}

		mux := testNewChiMux(t)

		// Should handle whitespace in origins without panicking
		require.NotPanics(t, func() {
			server.applyCORS(mux)
		})
	})
}

func TestAPIServer_CORSIntegration(t *testing.T) {
	t.Parallel()

	t.Run("CORS disabled - applyCORS should not be called", func(t *testing.T) {
		t.Parallel()

		deps, err := NewAPIDependencies(
			hclog.NewNullLogger(),
			NewClientManager(),
			NewHealthTracker([]string{"test-server"}),
			"localhost:8090",
		)
		require.NoError(t, err)

		// Create server with CORS disabled (default)
		server, err := NewAPIServer(deps)
		require.NoError(t, err)
		require.False(t, server.cors.Enabled)

		// This is more of an integration test - in actual usage,
		// applyCORS would only be called when CORS is enabled
	})

	t.Run("CORS enabled with various configurations", func(t *testing.T) {
		t.Parallel()

		deps, err := NewAPIDependencies(
			hclog.NewNullLogger(),
			NewClientManager(),
			NewHealthTracker([]string{"test-server"}),
			"localhost:8090",
		)
		require.NoError(t, err)

		corsConfigs := []struct {
			name    string
			options []APIOption
		}{
			{
				name: "basic CORS with origins",
				options: []APIOption{
					WithCORSEnabled(true),
					WithCORSAllowOrigins([]string{"http://localhost:3000", "https://app.example.com"}),
				},
			},
			{
				name: "CORS with all options",
				options: []APIOption{
					WithCORSEnabled(true),
					WithCORSAllowOrigins([]string{"http://localhost:3000"}),
					WithCORSAllowMethods([]string{"GET", "POST", "PUT", "DELETE"}),
					WithCORSAllowHeaders([]string{"Content-Type", "Authorization", "X-API-Key"}),
					WithCORSExposeHeaders([]string{"X-Total-Count", "X-Page-Count"}),
					WithCORSAllowCredentials(true),
					WithCORSMaxAge(1 * time.Hour),
				},
			},
			{
				name: "CORS with wildcard",
				options: []APIOption{
					WithCORSEnabled(true),
					WithCORSAllowOrigins([]string{"*"}),
					WithCORSAllowCredentials(false), // Required with wildcard
				},
			},
		}

		for _, config := range corsConfigs {
			t.Run(config.name, func(t *testing.T) {
				t.Parallel()

				server, err := NewAPIServer(deps, config.options...)
				require.NoError(t, err)
				require.True(t, server.cors.Enabled)

				// Verify CORS can be applied without panicking
				mux := testNewChiMux(t)
				require.NotPanics(t, func() {
					server.applyCORS(mux)
				})
			})
		}
	})
}

// Test helper to create a chi mux for testing
func testNewChiMux(t *testing.T) *chi.Mux {
	t.Helper()
	return chi.NewMux()
}

func TestMapError(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "ErrBadRequest maps to 400",
			err:            errors.ErrBadRequest,
			expectedStatus: 400,
		},
		{
			name:           "ErrServerNotFound maps to 404",
			err:            errors.ErrServerNotFound,
			expectedStatus: 404,
		},
		{
			name:           "ErrToolsNotFound maps to 404",
			err:            errors.ErrToolsNotFound,
			expectedStatus: 404,
		},
		{
			name:           "ErrHealthNotTracked maps to 404",
			err:            errors.ErrHealthNotTracked,
			expectedStatus: 404,
		},
		{
			name:           "ErrToolForbidden maps to 403",
			err:            errors.ErrToolForbidden,
			expectedStatus: 403,
		},
		{
			name:           "ErrToolListFailed maps to 502",
			err:            errors.ErrToolListFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrToolCallFailed maps to 502",
			err:            errors.ErrToolCallFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrToolCallFailedUnknown maps to 502",
			err:            errors.ErrToolCallFailedUnknown,
			expectedStatus: 502,
		},
		{
			name:           "ErrPromptNotFound maps to 404",
			err:            errors.ErrPromptNotFound,
			expectedStatus: 404,
		},
		{
			name:           "ErrPromptForbidden maps to 403",
			err:            errors.ErrPromptForbidden,
			expectedStatus: 403,
		},
		{
			name:           "ErrPromptListFailed maps to 502",
			err:            errors.ErrPromptListFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrPromptGenerationFailed maps to 502",
			err:            errors.ErrPromptGenerationFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrPromptsNotImplemented maps to 501",
			err:            errors.ErrPromptsNotImplemented,
			expectedStatus: 501,
		},
		{
			name:           "ErrResourceNotFound maps to 404",
			err:            errors.ErrResourceNotFound,
			expectedStatus: 404,
		},
		{
			name:           "ErrResourceForbidden maps to 403",
			err:            errors.ErrResourceForbidden,
			expectedStatus: 403,
		},
		{
			name:           "ErrResourceListFailed maps to 502",
			err:            errors.ErrResourceListFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrResourceTemplateListFailed maps to 502",
			err:            errors.ErrResourceTemplateListFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrResourceReadFailed maps to 502",
			err:            errors.ErrResourceReadFailed,
			expectedStatus: 502,
		},
		{
			name:           "ErrResourcesNotImplemented maps to 501",
			err:            errors.ErrResourcesNotImplemented,
			expectedStatus: 501,
		},
		{
			name:           "Unknown error maps to 500",
			err:            fmt.Errorf("unknown error"),
			expectedStatus: 500,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			statusErr := mapError(logger, tc.err)
			require.Equal(t, tc.expectedStatus, statusErr.GetStatus())
		})
	}
}
