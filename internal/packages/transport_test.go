package packages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransportConstants(t *testing.T) {
	require.Equal(t, "stdio", string(TransportStdio))
	require.Equal(t, "sse", string(TransportSSE))
	require.Equal(t, "streamable-http", string(TransportStreamableHTTP))
}

func TestAllTransports(t *testing.T) {
	all := AllTransports()
	require.Len(t, all, 3)
	require.Contains(t, all, TransportStdio)
	require.Contains(t, all, TransportSSE)
	require.Contains(t, all, TransportStreamableHTTP)
}

func TestDefaultTransports(t *testing.T) {
	defaults := DefaultTransports()
	require.Len(t, defaults, 1)
	require.Contains(t, defaults, TransportStdio)
}

func TestToStrings(t *testing.T) {
	transports := Transports{TransportStdio, TransportSSE}
	strings := transports.ToStrings()
	require.Equal(t, []string{"stdio", "sse"}, strings)
}

func TestFromStrings(t *testing.T) {
	t.Run("valid transports", func(t *testing.T) {
		strings := []string{"stdio", "sse", "streamable-http"}
		transports := FromStrings(strings)
		require.Len(t, transports, 3)
		require.Contains(t, transports, TransportStdio)
		require.Contains(t, transports, TransportSSE)
		require.Contains(t, transports, TransportStreamableHTTP)
	})

	t.Run("invalid transports default to stdio", func(t *testing.T) {
		strings := []string{"invalid", "unknown"}
		transports := FromStrings(strings)
		require.Equal(t, Transports{TransportStdio}, transports)
	})

	t.Run("mix of valid and invalid", func(t *testing.T) {
		strings := []string{"stdio", "invalid", "sse"}
		transports := FromStrings(strings)
		require.Len(t, transports, 2)
		require.Contains(t, transports, TransportStdio)
		require.Contains(t, transports, TransportSSE)
	})

	t.Run("empty input defaults to stdio", func(t *testing.T) {
		transports := FromStrings([]string{})
		require.Equal(t, Transports{TransportStdio}, transports)
	})
}

func TestHasTransport(t *testing.T) {
	transports := []Transport{TransportStdio, TransportSSE}

	require.True(t, HasTransport(transports, TransportStdio))
	require.True(t, HasTransport(transports, TransportSSE))
	require.False(t, HasTransport(transports, TransportStreamableHTTP))
}
