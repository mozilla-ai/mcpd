package daemon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := defaultOptions()

	assert.Nil(t, opts.APIOptions) // No API options by default - NewAPIServer will apply its own defaults
	assert.Equal(t, DefaultClientInitTimeout(), opts.ClientInitTimeout)
	assert.Equal(t, DefaultHealthCheckInterval(), opts.ClientHealthCheckInterval)
	assert.Equal(t, DefaultHealthCheckTimeout(), opts.ClientHealthCheckTimeout)
	assert.Equal(t, DefaultClientShutdownTimeout(), opts.ClientShutdownTimeout)
}

func TestNewOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()

		opts, err := NewOptions()

		require.NoError(t, err)
		assert.Nil(t, opts.APIOptions) // No API options by default - NewAPIServer will apply its own defaults
		assert.Equal(t, DefaultClientInitTimeout(), opts.ClientInitTimeout)
		assert.Equal(t, DefaultHealthCheckInterval(), opts.ClientHealthCheckInterval)
		assert.Equal(t, DefaultHealthCheckTimeout(), opts.ClientHealthCheckTimeout)
		assert.Equal(t, DefaultClientShutdownTimeout(), opts.ClientShutdownTimeout)
	})

	t.Run("with API options", func(t *testing.T) {
		t.Parallel()

		apiOptions := []APIOption{
			WithCORSEnabled(true),
			WithCORSAllowOrigins([]string{"http://localhost:3000"}),
			WithCORSMaxAge(10 * time.Minute),
			WithShutdownTimeout(10 * time.Second),
		}
		opts, err := NewOptions(WithAPIOptions(apiOptions...))

		require.NoError(t, err)
		require.Len(t, opts.APIOptions, 4)

		// Verify the options work by creating an APIOptions struct
		resultAPIOptions, err := NewAPIOptions(opts.APIOptions...)
		require.NoError(t, err)
		assert.True(t, resultAPIOptions.CORS.Enabled)
		assert.ElementsMatch(t, []string{"http://localhost:3000"}, resultAPIOptions.CORS.AllowOrigins)
		assert.Equal(t, 10*time.Minute, resultAPIOptions.CORS.MaxAge)
		assert.Equal(t, 10*time.Second, resultAPIOptions.ShutdownTimeout)
	})

	t.Run("with init timeout", func(t *testing.T) {
		t.Parallel()

		timeout := 60 * time.Second
		opts, err := NewOptions(WithMCPServerInitTimeout(timeout))

		require.NoError(t, err)
		assert.Equal(t, timeout, opts.ClientInitTimeout)
	})

	t.Run("with health check settings", func(t *testing.T) {
		t.Parallel()

		interval := 5 * time.Second
		timeout := 2 * time.Second
		opts, err := NewOptions(
			WithMCPServerHealthCheckInterval(interval),
			WithMCPServerHealthCheckTimeout(timeout),
		)

		require.NoError(t, err)
		assert.Equal(t, interval, opts.ClientHealthCheckInterval)
		assert.Equal(t, timeout, opts.ClientHealthCheckTimeout)
	})

	t.Run("with client shutdown timeout", func(t *testing.T) {
		t.Parallel()

		timeout := 10 * time.Second
		opts, err := NewOptions(WithMCPServerShutdownTimeout(timeout))

		require.NoError(t, err)
		assert.Equal(t, timeout, opts.ClientShutdownTimeout)
	})

	t.Run("options override in order", func(t *testing.T) {
		t.Parallel()

		first := 5 * time.Second
		second := 10 * time.Second

		opts, err := NewOptions(
			WithMCPServerInitTimeout(first),
			WithMCPServerInitTimeout(second), // This should win
		)

		require.NoError(t, err)
		assert.Equal(t, second, opts.ClientInitTimeout)
	})
}

func TestWithTimeouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr string
	}{
		{
			name:    "valid timeout",
			timeout: 10 * time.Second,
		},
		{
			name:    "zero timeout fails",
			timeout: 0,
			wantErr: "must be positive, got 0s",
		},
		{
			name:    "negative timeout fails",
			timeout: -1 * time.Second,
			wantErr: "must be positive, got -1s",
		},
	}

	timeoutOptions := []struct {
		name string
		opt  func(time.Duration) Option
	}{
		{"WithMCPServerInitTimeout", WithMCPServerInitTimeout},
		{"WithMCPServerHealthCheckInterval", WithMCPServerHealthCheckInterval},
		{"WithMCPServerHealthCheckTimeout", WithMCPServerHealthCheckTimeout},
		{"WithMCPServerShutdownTimeout", WithMCPServerShutdownTimeout},
	}

	for _, timeoutOpt := range timeoutOptions {
		for _, tc := range tests {
			t.Run(timeoutOpt.name+"_"+tc.name, func(t *testing.T) {
				t.Parallel()

				_, err := NewOptions(timeoutOpt.opt(tc.timeout))

				if tc.wantErr == "" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.wantErr)
				}
			})
		}
	}
}
