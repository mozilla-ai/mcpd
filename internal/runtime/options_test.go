package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSupportedRuntimes(t *testing.T) {
	runtimes := DefaultSupportedRuntimes()
	require.Len(t, runtimes, 2)
	require.Contains(t, runtimes, NPX)
	require.Contains(t, runtimes, UVX)
}

func TestNewOptions_Defaults(t *testing.T) {
	opts, err := NewOptions()
	require.NoError(t, err)
	require.NotNil(t, opts.SupportedRuntimes)
	require.Contains(t, opts.SupportedRuntimes, NPX)
	require.Contains(t, opts.SupportedRuntimes, UVX)
}

func TestWithSupportedRuntimes_Valid(t *testing.T) {
	opts, err := NewOptions(WithSupportedRuntimes(NPX))
	require.NoError(t, err)
	require.Len(t, opts.SupportedRuntimes, 1)
	require.Contains(t, opts.SupportedRuntimes, NPX)
	require.NotContains(t, opts.SupportedRuntimes, UVX)
}

func TestWithSupportedRuntimes_Empty(t *testing.T) {
	_, err := NewOptions(WithSupportedRuntimes())
	require.Error(t, err)
	require.EqualError(t, err, "must specify at least one supported runtime")
}
