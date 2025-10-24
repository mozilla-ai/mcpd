package plugins

import (
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// mockLoader is a mock config.Loader for testing.
type mockLoader struct {
	cfg *config.Config
	err error
}

func (m *mockLoader) Load(_ string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cfg, nil
}
