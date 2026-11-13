package daemon

import (
	"encoding/json"
	"mime"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// contentTypeJSON is the only media type the API accepts for a request body.
const contentTypeJSON = "application/json"

// methodsWithBody lists the request methods whose body the API reads.
var methodsWithBody = map[string]struct{}{
	http.MethodPatch: {},
	http.MethodPost:  {},
	http.MethodPut:   {},
}

// isOriginAllowed reports whether origin matches an entry in allowed.
//
// A bare '*' allows every origin. Any other entry may contain one '*' standing for a run of
// characters, which matches the pattern syntax the CORS layer accepts.
func isOriginAllowed(allowed []string, origin string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))

	for _, entry := range allowed {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}

		if entry == "*" {
			return true
		}

		prefix, suffix, found := strings.Cut(entry, "*")
		if !found {
			if entry == origin {
				return true
			}

			continue
		}

		if len(origin) >= len(prefix)+len(suffix) &&
			strings.HasPrefix(origin, prefix) &&
			strings.HasSuffix(origin, suffix) {
			return true
		}
	}

	return false
}

// requireAllowedOrigin refuses a request whose Origin is not one the operator allowed.
//
// A request without an Origin header is allowed, because only a browser sets one. A wildcard
// entry allows every origin.
//
// NOTE: This does not see a rebinding page, which is same-origin with the daemon and so sends no
// Origin header on a read.
func requireAllowedOrigin(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !isOriginAllowed(allowed, origin) {
				writeProblem(
					w,
					http.StatusForbidden,
					"Forbidden",
					"Requests from this origin are not allowed.",
				)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// requireJSONContentType rejects a body-carrying request that does not declare a JSON media type.
//
// NOTE: A browser sends a cross-site request with no Content-Type without asking permission first.
// Requiring JSON means such a request needs permission, which an unapproved origin cannot obtain.
func requireJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := methodsWithBody[r.Method]; !ok {
			next.ServeHTTP(w, r)
			return
		}

		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != contentTypeJSON {
			writeProblem(
				w,
				http.StatusUnsupportedMediaType,
				"Unsupported Media Type",
				"Request body must be sent with an 'application/json' content type.",
			)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeProblem writes an error response in the same shape the API uses elsewhere.
func writeProblem(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(huma.ErrorModel{
		Status: status,
		Title:  title,
		Detail: detail,
	})
}
