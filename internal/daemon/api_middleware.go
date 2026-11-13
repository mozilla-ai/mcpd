package daemon

import (
	"encoding/json"
	"mime"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// contentTypeJSON is the only media type the API accepts for a request body.
const contentTypeJSON = "application/json"

// loopbackHostNames are the names a client uses to address a daemon bound to a loopback address.
//
// NOTE: "localhost" is included because a bind address may name it rather than an IP, and because
// a client resolving it reports it verbatim in the Host header.
var loopbackHostNames = []string{"127.0.0.1", "::1", "localhost"}

// methodsWithBody lists the request methods whose body the API reads.
var methodsWithBody = map[string]struct{}{
	http.MethodPatch: {},
	http.MethodPost:  {},
	http.MethodPut:   {},
}

// allowedHosts returns the host names a request may address for a daemon bound to addr.
//
// An empty result permits any host. A routable bind address gives no way to know the names
// clients will use, since they reach the daemon through a service name, an ingress or a proxy.
func allowedHosts(addr string) []string {
	if !isLoopbackBind(addr) {
		return nil
	}

	hosts := slices.Clone(loopbackHostNames)
	if host := bindHost(addr); host != "" && !slices.Contains(hosts, host) {
		hosts = append(hosts, host)
	}

	return hosts
}

// bindHost returns the lowercased host part of a bind address, without its port.
func bindHost(addr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(addr))
	}

	return strings.ToLower(host)
}

// isHostAllowed reports whether host, as sent in a Host header, is one the daemon serves.
func isHostAllowed(allowed []string, host string) bool {
	if len(allowed) == 0 {
		return true
	}

	name := strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(name); err == nil {
		name = h
	}
	name = strings.Trim(name, "[]")

	return slices.Contains(allowed, name)
}

// isLoopbackBind reports whether addr binds an address reachable only from the same machine.
//
// NOTE: The host part may be a name rather than an IP, so parsing it as an IP is not enough.
func isLoopbackBind(addr string) bool {
	host := bindHost(addr)
	if host == "" {
		return false
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return slices.Contains(loopbackHostNames, host)
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

// ownOrigins returns the origins that address the daemon itself for a daemon bound to addr.
//
// The daemon serves its own API documentation page, which calls the API from these origins. The
// configured bind host is included alongside the loopback names, so the page works when the
// daemon is reached at the address it was given.
func ownOrigins(addr string) []string {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil || port == "" {
		return nil
	}

	origins := make([]string, 0, len(loopbackHostNames)+1)
	for _, name := range loopbackHostNames {
		origins = append(origins, "http://"+net.JoinHostPort(name, port))
	}

	host = strings.ToLower(strings.Trim(host, "[]"))
	if host != "" && !slices.Contains(loopbackHostNames, host) {
		origins = append(origins, "http://"+net.JoinHostPort(host, port))
	}

	return origins
}

// requireAllowedHost refuses a request addressed to a host the daemon does not serve.
//
// NOTE: This is the defense against DNS rebinding. A browser omits Origin on a same-origin GET,
// and a rebound page is same-origin with the daemon, so the origin allowlist cannot see it.
func requireAllowedHost(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isHostAllowed(allowed, r.Host) {
				writeProblem(
					w,
					http.StatusForbidden,
					"Forbidden",
					"Requests addressed to this host are not served.",
				)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// requireAllowedOrigin refuses a request whose Origin is not one the operator allowed.
//
// A request without an Origin header is allowed, because only a browser sets one. A wildcard
// entry allows every origin.
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
