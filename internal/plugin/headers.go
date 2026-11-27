package plugin

import (
	"net/http"
	"slices"

	"github.com/mozilla-ai/mcpd/internal/api"
)

// reservedHeaders are the headers a plugin must not set, keyed in canonical form.
// mcpd owns message framing and its own protocol header, and the operator owns the CORS policy.
// Vary is reserved because the CORS layer sets it, and a plugin that replaced it would let a shared cache
// serve one origin's response to another.
// NOTE: This variable must not be mutated within the package.
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
	"Vary":                             {},
}

// isReservedHeader reports whether name is a header a plugin must not set.
func isReservedHeader(name string) bool {
	_, ok := reservedHeaders[http.CanonicalHeaderKey(name)]

	return ok
}

// reservedHeaderNames returns the reserved header names in headers, in canonical form and sorted.
// The result is sorted so a log line listing them is stable across calls.
//
// NOTE: This returns names only. A reserved header's value may be a credential, so a caller must not log it.
func reservedHeaderNames(headers map[string]string) []string {
	var names []string

	for name := range headers {
		if isReservedHeader(name) {
			names = append(names, http.CanonicalHeaderKey(name))
		}
	}

	slices.Sort(names)

	return names
}
