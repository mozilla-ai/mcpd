package packages

const (
	// TransportStdio represents standard input/output transport (default).
	// This is the most common transport used by MCP servers.
	TransportStdio Transport = "stdio"

	// TransportSSE represents SSE transport.
	TransportSSE Transport = "sse"

	// TransportStreamableHTTP represents streamable-HTTP (websocket) transport.
	TransportStreamableHTTP Transport = "streamable-http"
)

// Transport represents the supported transport mechanisms for MCP servers.
type Transport string

type Transports []Transport

// AllTransports returns all supported transport types.
func AllTransports() []Transport {
	return Transports{
		TransportStdio,
		TransportSSE,
		TransportStreamableHTTP,
	}
}

// DefaultTransports returns the default transports that most MCP servers support.
// By convention, all MCP servers support stdio transport.
func DefaultTransports() Transports {
	return []Transport{TransportStdio}
}

// ToStrings converts a slice of Transport to a slice of strings.
func (t Transports) ToStrings() []string {
	result := make([]string, len(t))
	for i, transport := range t {
		result[i] = string(transport)
	}
	return result
}

// FromStrings converts a slice of strings to a slice of Transport.
// Unknown transport types are skipped.
func FromStrings(ts []string) Transports {
	var result []Transport
	validTransports := map[string]Transport{
		string(TransportStdio):          TransportStdio,
		string(TransportSSE):            TransportSSE,
		string(TransportStreamableHTTP): TransportStreamableHTTP,
	}

	for _, str := range ts {
		if transport, ok := validTransports[str]; ok {
			result = append(result, transport)
		}
	}

	// Always ensure stdio is included if no valid transports found
	if len(result) == 0 {
		result = DefaultTransports()
	}

	return result
}

// HasTransport checks if a slice of transports contains a specific transport.
func HasTransport(transports Transports, transport Transport) bool {
	for _, t := range transports {
		if t == transport {
			return true
		}
	}
	return false
}
