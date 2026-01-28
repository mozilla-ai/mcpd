package options

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/registry"
)

type fakeLoader struct {
	config.Loader
}

type fakeBuilder struct {
	registry.Builder
}

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	require.NotNil(t, opts.ConfigLoader)
	require.NotNil(t, opts.ConfigInitializer)
	require.NotNil(t, opts.RegistryBuilder)
}

func TestNewOptions_NoOverrides(t *testing.T) {
	opts, err := NewOptions()
	assert.NoError(t, err)

	require.NotNil(t, opts.ConfigLoader)
	require.NotNil(t, opts.ConfigInitializer)
	require.NotNil(t, opts.RegistryBuilder)
}

func TestNewOptions_WithOverrides(t *testing.T) {
	loader := &fakeLoader{}
	builder := &fakeBuilder{}

	opts, err := NewOptions(
		WithConfigLoader(loader),
		WithRegistryBuilder(builder),
	)
	require.NoError(t, err)

	require.Equal(t, loader, opts.ConfigLoader)
	require.Equal(t, builder, opts.RegistryBuilder)
}

func TestNewOptions_WithNilOption(t *testing.T) {
	opts, err := NewOptions(nil)
	require.NoError(t, err)
	require.NotNil(t, opts)
}

func TestNewOptions_WithFailingOption(t *testing.T) {
	badOpt := func(*CmdOptions) error {
		return errors.New("fail")
	}

	_, err := NewOptions(badOpt)
	require.Error(t, err)
	require.ErrorContains(t, err, "fail")
}

func TestWithConfigLoader_NilLoader(t *testing.T) {
	t.Parallel()

	_, err := NewOptions(WithConfigLoader(nil))

	require.Error(t, err)
	require.ErrorContains(t, err, "config loader cannot be nil")
}

func TestWithConfigLoader_NilInterface(t *testing.T) {
	t.Parallel()

	var loader *fakeLoader // Typed nil.

	_, err := NewOptions(WithConfigLoader(loader))

	require.Error(t, err)
	require.ErrorContains(t, err, "config loader cannot be nil")
}
