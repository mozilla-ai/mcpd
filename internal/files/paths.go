// net/url parses "file://C:/path" as host="C:", path="/path" — the drive letter becomes
// the URL host, not part of the path. PathToFileURL and FileURLToPath encapsulate the
// Windows workaround (a leading slash before the drive letter) so callers don't need to.
package files

import (
	"net/url"
	"path/filepath"
	"regexp"
)

// matches a Windows drive-letter path (e.g. "C:/Users/...").
var windowsDrivePath = regexp.MustCompile(`^[a-zA-Z]:`)

// matches a Windows drive-letter path as file URI (e.g. "/C:/Users/...").
var windowsURIDrivePath = regexp.MustCompile(`^/[a-zA-Z]:`)

// On Windows, a leading slash is prepended before the drive letter so it isn't
// misread as the URL host by net/url ("file:///C:/..." rather than "file://C:/...").
func PathToFileURL(path string) string {
	p := filepath.ToSlash(path)
	if windowsDrivePath.MatchString(p) {
		p = "/" + p
	}

	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}

// Strips the leading slash added before a Windows drive letter ("/C:/..." → "C:/...").
// Also handles file://C:/... URLs where net/url places the drive letter in u.Host.
func FileURLToPath(u *url.URL) string {
	if windowsDrivePath.MatchString(u.Host) {
		return u.Host + u.Path
	}

	path := u.Path
	if windowsURIDrivePath.MatchString(path) {
		return path[1:]
	}

	return path
}
