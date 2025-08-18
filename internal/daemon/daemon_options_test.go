package daemon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := defaultOptions()

	require.Nil(t, opts.APIOptions) // No API options by default - NewAPIServer will apply its own defaults
	require.Equal(t, DefaultClientInitTimeout(), opts.ClientInitTimeout)
	require.Equal(t, DefaultHealthCheckInterval(), opts.ClientHealthCheckInterval)
	require.Equal(t, DefaultHealthCheckTimeout(), opts.ClientHealthCheckTimeout)
	require.Equal(t, DefaultClientShutdownTimeout(), opts.ClientShutdownTimeout)
}

func TestNewOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()

		opts, err := NewOptions()

		require.NoError(t, err)
		require.Nil(t, opts.APIOptions) // No API options by default - NewAPIServer will apply its own defaults
		require.Equal(t, DefaultClientInitTimeout(), opts.ClientInitTimeout)
		require.Equal(t, DefaultHealthCheckInterval(), opts.ClientHealthCheckInterval)
		require.Equal(t, DefaultHealthCheckTimeout(), opts.ClientHealthCheckTimeout)
		require.Equal(t, DefaultClientShutdownTimeout(), opts.ClientShutdownTimeout)
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
		require.True(t, resultAPIOptions.CORS.Enabled)
		require.ElementsMatch(t, []string{"http://localhost:3000"}, resultAPIOptions.CORS.AllowOrigins)
		require.Equal(t, 10*time.Minute, resultAPIOptions.CORS.MaxAge)
		require.Equal(t, 10*time.Second, resultAPIOptions.ShutdownTimeout)
	})

	t.Run("with init timeout", func(t *testing.T) {
		t.Parallel()

		timeout := 60 * time.Second
		opts, err := NewOptions(WithMCPServerInitTimeout(timeout))

		require.NoError(t, err)
		require.Equal(t, timeout, opts.ClientInitTimeout)
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
		require.Equal(t, interval, opts.ClientHealthCheckInterval)
		require.Equal(t, timeout, opts.ClientHealthCheckTimeout)
	})

	t.Run("with client shutdown timeout", func(t *testing.T) {
		t.Parallel()

		timeout := 10 * time.Second
		opts, err := NewOptions(WithMCPServerShutdownTimeout(timeout))

		require.NoError(t, err)
		require.Equal(t, timeout, opts.ClientShutdownTimeout)
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
		require.Equal(t, second, opts.ClientInitTimeout)
	})
}

func TestWithTimeouts(t *testing.T) {
	t.Parallel()

	timeoutOptions := []struct {
		name        string
		opt         func(time.Duration) Option
		zeroErr     string
		negativeErr string
	}{
		{
			"WithMCPServerInitTimeout",
			WithMCPServerInitTimeout,
			"init timeout must be positive, got 0s",
			"init timeout must be positive, got -1s",
		},
		{
			"WithMCPServerHealthCheckInterval",
			WithMCPServerHealthCheckInterval,
			"health check interval must be positive, got 0s",
			"health check interval must be positive, got -1s",
		},
		{
			"WithMCPServerHealthCheckTimeout",
			WithMCPServerHealthCheckTimeout,
			"health check timeout must be positive, got 0s",
			"health check timeout must be positive, got -1s",
		},
		{
			"WithMCPServerShutdownTimeout",
			WithMCPServerShutdownTimeout,
			"server shutdown timeout must be positive, got 0s",
			"server shutdown timeout must be positive, got -1s",
		},
	}

	for _, timeoutOpt := range timeoutOptions {
		t.Run(timeoutOpt.name+"_valid_timeout", func(t *testing.T) {
			t.Parallel()
			_, err := NewOptions(timeoutOpt.opt(10 * time.Second))
			require.NoError(t, err)
		})

		t.Run(timeoutOpt.name+"_zero_timeout_fails", func(t *testing.T) {
			t.Parallel()
			_, err := NewOptions(timeoutOpt.opt(0))
			require.Error(t, err)
			require.EqualError(t, err, timeoutOpt.zeroErr)
		})

		t.Run(timeoutOpt.name+"_negative_timeout_fails", func(t *testing.T) {
			t.Parallel()
			_, err := NewOptions(timeoutOpt.opt(-1 * time.Second))
			require.Error(t, err)
			require.EqualError(t, err, timeoutOpt.negativeErr)
		})
	}
}
