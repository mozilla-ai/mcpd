package daemon

import (
	"encoding/json"
	"mime"
	"net/http"

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
