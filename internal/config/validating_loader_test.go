package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockLoader is a test double for config.Loader.
type mockLoader struct {
	modifier Modifier
	err      error
}

func (m *mockLoader) Load(path string) (Modifier, error) {
	return m.modifier, m.err
}

// mockModifier is a test double for config.Modifier that is NOT a *Config.
type mockModifier struct{}

func (m *mockModifier) AddServer(entry ServerEntry) error { return nil }
func (m *mockModifier) RemoveServer(name string) error    { return nil }
func (m *mockModifier) ListServers() []ServerEntry        { return nil }
func (m *mockModifier) SaveConfig() error                 { return nil }

func TestNewValidatingLoader(t *testing.T) {
	t.Parallel()

	inner := &mockLoader{}
	loader := NewValidatingLoader(inner)

	require.NotNil(t, loader)
	require.Equal(t, inner, loader.Loader)
}

func TestValidatingLoader_Load_DelegatesError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("load failed")
	inner := &mockLoader{err: expectedErr}
	loader := NewValidatingLoader(inner)

	_, err := loader.Load("/some/path")

	require.ErrorIs(t, err, expectedErr)
}

func TestValidatingLoader_Load_RejectsNonConfig(t *testing.T) {
	t.Parallel()

	mock := &mockModifier{}
	inner := &mockLoader{modifier: mock}
	loader := NewValidatingLoader(inner)

	_, err := loader.Load("/some/path")

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config structure")
}

func TestValidatingLoader_Load_RunsPredicates(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	inner := &mockLoader{modifier: cfg}

	predicateCalled := false
	testPredicate := func(c *Config) error {
		predicateCalled = true
		require.Equal(t, cfg, c)
		return nil
	}

	loader := NewValidatingLoader(inner, testPredicate)
	result, err := loader.Load("/some/path")

	require.NoError(t, err)
	require.Equal(t, cfg, result)
	require.True(t, predicateCalled)
}

func TestValidatingLoader_Load_PredicateError(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	inner := &mockLoader{modifier: cfg}

	expectedErr := errors.New("validation failed")
	failingPredicate := func(c *Config) error {
		return expectedErr
	}

	loader := NewValidatingLoader(inner, failingPredicate)
	_, err := loader.Load("/some/path")

	require.ErrorIs(t, err, expectedErr)
}

func TestValidatingLoader_Load_SkipsWithoutPredicates(t *testing.T) {
	t.Parallel()

	// Without predicates, even invalid plugin config passes.
	cfg := &Config{
		Plugins: &PluginConfig{
			Dir: "/non/existent/directory",
			Authentication: []PluginEntry{
				{Name: "test-plugin", Flows: []Flow{FlowRequest}},
			},
		},
	}
	inner := &mockLoader{modifier: cfg}
	loader := NewValidatingLoader(inner)

	result, err := loader.Load("/some/path")

	require.NoError(t, err)
	require.Equal(t, cfg, result)
}

func TestValidatingLoader_WithPluginBinaryValidator(t *testing.T) {
	t.Parallel()

	t.Run("passes with valid plugin directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: dir,
			},
		}
		inner := &mockLoader{modifier: cfg}

		loader := NewValidatingLoader(inner, ValidatePluginBinaries)
		result, err := loader.Load("/some/path")

		require.NoError(t, err)
		require.Equal(t, cfg, result)
	})

	t.Run("returns validation error for missing directory", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Plugins: &PluginConfig{
				Dir: "/non/existent/directory",
				Authentication: []PluginEntry{
					{Name: "test-plugin", Flows: []Flow{FlowRequest}},
				},
			},
		}
		inner := &mockLoader{modifier: cfg}

		loader := NewValidatingLoader(inner, ValidatePluginBinaries)
		_, err := loader.Load("/some/path")

		require.Error(t, err)
	})
}
