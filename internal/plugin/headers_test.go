package plugin

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsReservedHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		reserved bool
	}{
		{
			name:     "canonical reserved header",
			header:   "Content-Length",
			reserved: true,
		},
		{
			name:     "lowercase reserved header",
			header:   "access-control-allow-origin",
			reserved: true,
		},
		{
			name:     "mcpd protocol header",
			header:   "mcpd-error-type",
			reserved: true,
		},
		{
			name:     "plugin header",
			header:   "X-Team-ID",
			reserved: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.reserved, isReservedHeader(tc.header))
		})
	}
}

func TestReservedHeaders_KeysAreCanonical(t *testing.T) {
	t.Parallel()

	// A key that is not canonical is unreachable, because isReservedHeader canonicalizes its argument.
	for name := range reservedHeaders {
		require.Equal(t, http.CanonicalHeaderKey(name), name, "reserved header key must be canonical")
	}
}
