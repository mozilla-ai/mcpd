package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// LoadFromURL retrieves and decodes JSON from the given URL into the target registry object.
// Supports both HTTP(S) and file:// URLs.
func LoadFromURL[T any](registryURL string, registryName string) (T, error) {
	var target T

	// Parse URL to determine scheme
	parsedURL, err := url.Parse(registryURL)
	if err != nil {
		return target, fmt.Errorf("invalid URL '%s': %w", registryURL, err)
	}

	var body []byte

	switch parsedURL.Scheme {
	case "file":
		// Handle file:// URLs
		path := parsedURL.Path

		body, err = os.ReadFile(path)
		if err != nil {
			return target, fmt.Errorf("failed to read file '%s' for registry '%s': %w", path, registryName, err)
		}

	case "http", "https", "":
		// Handle HTTP(S) URLs (empty scheme defaults to HTTP for backward compatibility)
		if parsedURL.Scheme == "" {
			registryURL = "http://" + registryURL
		}

		resp, err := http.Get(registryURL)
		if err != nil {
			return target, fmt.Errorf(
				"failed to fetch '%s' registry data from URL '%s': %w",
				registryName,
				registryURL,
				err,
			)
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return target, fmt.Errorf(
				"received non-OK HTTP status from '%s' registry for URL '%s': %d",
				registryName,
				registryURL,
				resp.StatusCode,
			)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return target, fmt.Errorf(
				"failed to read '%s' registry response body from '%s': %w",
				registryName,
				registryURL,
				err,
			)
		}

	default:
		return target, fmt.Errorf("unsupported URL scheme '%s' for registry '%s'", parsedURL.Scheme, registryName)
	}

	// Unmarshal JSON (common for both paths)
	if err := json.Unmarshal(body, &target); err != nil {
		return target, fmt.Errorf(
			"failed to unmarshal '%s' registry JSON from '%s': %w",
			registryName,
			registryURL,
			err,
		)
	}

	return target, nil
}
