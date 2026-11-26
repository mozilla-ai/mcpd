package plugin

import (
	"net/http"

	"github.com/mozilla-ai/mcpd/internal/api"
)

// reservedHeaders are the headers a plugin must not set, keyed in canonical form.
// mcpd owns message framing and its own protocol header, and the operator owns the CORS policy.
var reservedHeaders = map[string]struct{}{
	"Access-Control-Allow-Credentials": {},
	"Access-Control-Allow-Headers":     {},
	"Access-Control-Allow-Methods":     {},
	"Access-Control-Allow-Origin":      {},
	"Access-Control-Expose-Headers":    {},
	"Access-Control-Max-Age":           {},
	"Connection":                       {},
	"Content-Length":                   {},
	"Keep-Alive":                       {},
	api.HeaderErrorType:                {},
	"Proxy-Authenticate":               {},
	"Proxy-Authorization":              {},
	"Te":                               {},
	"Trailer":                          {},
	"Transfer-Encoding":                {},
	"Upgrade":                          {},
}

// isReservedHeader reports whether name is a header a plugin must not set.
func isReservedHeader(name string) bool {
	_, ok := reservedHeaders[http.CanonicalHeaderKey(name)]

	return ok
}
