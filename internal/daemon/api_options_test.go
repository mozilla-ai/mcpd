package daemon

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemon_NewAPIOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()

		opts, err := NewAPIOptions()
		require.NoError(t, err)
		assert.Equal(t, DefaultAPIShutdownTimeout(), opts.ShutdownTimeout)
		assert.False(t, opts.CORS.Enabled)
	})

	t.Run("with CORS option", func(t *testing.T) {
		t.Parallel()

		origins := []string{"http://localhost:3000", "https://example.com"}
		opts, err := NewAPIOptions(WithCORSAllowOrigins(origins))

		require.NoError(t, err)
		assert.False(t, opts.CORS.Enabled)
		assert.Equal(t, origins, opts.CORS.AllowOrigins)
		assert.Contains(t, opts.CORS.AllowMethods, http.MethodGet)
		assert.Contains(t, opts.CORS.AllowMethods, http.MethodPost)
		require.Len(t, opts.CORS.AllowedHeaders, 5)
		assert.Contains(t, opts.CORS.AllowedHeaders, "Accept")
		assert.Contains(t, opts.CORS.AllowedHeaders, "Accept-Language")
		assert.Contains(t, opts.CORS.AllowedHeaders, "Content-Language")
		assert.Contains(t, opts.CORS.AllowedHeaders, "Content-Type")
		assert.Contains(t, opts.CORS.AllowedHeaders, "Range")
		assert.Equal(t, 5*time.Minute, opts.CORS.MaxAge)
	})

	t.Run("with custom shutdown timeout", func(t *testing.T) {
		t.Parallel()

		customTimeout := 10 * time.Second
		opts, err := NewAPIOptions(WithShutdownTimeout(customTimeout))

		require.NoError(t, err)
		assert.Equal(t, customTimeout, opts.ShutdownTimeout)
	})

	t.Run("options override in order", func(t *testing.T) {
		t.Parallel()

		first := 5 * time.Second
		second := 10 * time.Second

		opts, err := NewAPIOptions(
			WithShutdownTimeout(first),
			WithShutdownTimeout(second), // This should win
		)

		require.NoError(t, err)
		assert.Equal(t, second, opts.ShutdownTimeout)
	})
}

func TestDaemon_APIOptions_WithShutdownTimeout(t *testing.T) {
	t.Parallel()

	t.Run("valid timeout", func(t *testing.T) {
		t.Parallel()

		timeout := 10 * time.Second
		opts, err := NewAPIOptions(WithShutdownTimeout(timeout))

		require.NoError(t, err)
		assert.Equal(t, timeout, opts.ShutdownTimeout)
	})

	t.Run("zero timeout fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewAPIOptions(WithShutdownTimeout(0))

		require.Error(t, err)
		require.EqualError(t, err, "shutdown timeout must be positive, got 0s")
	})

	t.Run("negative timeout fails", func(t *testing.T) {
		t.Parallel()

		_, err := NewAPIOptions(WithShutdownTimeout(-1 * time.Second))

		require.Error(t, err)
		require.EqualError(t, err, "shutdown timeout must be positive, got -1s")
	})
}

func TestDaemon_APIOptions_DefaultCORSHeaders(t *testing.T) {
	t.Parallel()

	headers := DefaultCORSAllowHeaders()
	require.Len(t, headers, 5)
	assert.Contains(t, headers, "Accept")
	assert.Contains(t, headers, "Accept-Language")
	assert.Contains(t, headers, "Content-Language")
	assert.Contains(t, headers, "Content-Type")
	assert.Contains(t, headers, "Range")
}

func TestDaemon_APIOptions_DefaultCORSMethods(t *testing.T) {
	t.Parallel()

	methods := DefaultCORSAllowMethods()

	assert.Contains(t, methods, http.MethodGet)
	assert.Contains(t, methods, http.MethodPost)
	assert.Contains(t, methods, http.MethodPut)
	assert.Contains(t, methods, http.MethodDelete)
	assert.Contains(t, methods, http.MethodOptions)
}

func TestDaemon_APIOptions_DefaultCORSAllowCredentials(t *testing.T) {
	t.Parallel()

	allowCredentials := DefaultCORSAllowCredentials()

	assert.Equal(t, DefaultCORSAllowCredentials(), allowCredentials)
	assert.False(t, allowCredentials)
}

func TestDaemon_APIOptions_DefaultCORSMaxAge(t *testing.T) {
	t.Parallel()

	maxAge := DefaultCORSMaxAge()

	assert.Equal(t, DefaultCORSMaxAge(), maxAge)
	assert.Equal(t, 5*time.Minute, maxAge)
}

func TestDaemon_ValidateAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "valid host and port",
			addr:    "localhost:8090",
			wantErr: false,
		},
		{
			name:    "valid IP and port",
			addr:    "127.0.0.1:8090",
			wantErr: false,
		},
		{
			name:    "empty host with port",
			addr:    ":8090",
			wantErr: false,
		},
		{
			name:    "missing port",
			addr:    "localhost",
			wantErr: true,
		},
		{
			name:    "invalid format",
			addr:    "invalid-address",
			wantErr: true,
		},
		{
			name:    "empty port",
			addr:    "localhost:",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateAddr(tc.addr)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
