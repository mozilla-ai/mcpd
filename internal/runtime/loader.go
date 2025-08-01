package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LoadFromURL retrieves and decodes JSON from the given URL into the target registry object.
func LoadFromURL[T any](url, registryName string) (T, error) {
	var target T

	resp, err := http.Get(url)
	if err != nil {
		return target, fmt.Errorf("failed to fetch '%s' registry data from URL '%s': %w", registryName, url, err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return target, fmt.Errorf(
			"received non-OK HTTP status from '%s' registry for URL '%s': %d",
			registryName,
			url,
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return target, fmt.Errorf("failed to read '%s' registry response body from '%s': %w", registryName, url, err)
	}

	if err := json.Unmarshal(body, &target); err != nil {
		return target, fmt.Errorf("failed to unmarshal '%s' registry JSON from '%s': %w", registryName, url, err)
	}

	return target, nil
}
