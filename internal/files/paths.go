package files

import (
	"net/url"
	"path/filepath"
	"regexp"
)

// windowsDrivePath matches a Windows drive-letter path (e.g. "C:/Users/...").
var windowsDrivePath = regexp.MustCompile(`^[a-zA-Z]:`)

// windowsURIDrivePath matches a Windows drive-letter path as a file URI (e.g. "/C:/Users/...").
var windowsURIDrivePath = regexp.MustCompile(`^/[a-zA-Z]:`)

// PathToFileURL converts a filesystem path to a file:// URL.
//
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

// FileURLToPath converts a parsed file:// URL back to a filesystem path.
//
// It strips the leading slash added before a Windows drive letter ("/C:/..." → "C:/...")
// and also handles file://C:/... URLs where net/url places the drive letter in u.Host.
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
