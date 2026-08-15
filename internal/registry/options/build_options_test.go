package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestDefaultCacheTTL(t *testing.T) {
	t.Parallel()

	ttl := DefaultCacheTTL()
	require.Equal(t, config.Duration(24*time.Hour), ttl)
	require.Equal(t, "24h", ttl.String())
}
