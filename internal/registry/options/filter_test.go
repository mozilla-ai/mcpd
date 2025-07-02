package options

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareFilters(t *testing.T) {
	orig := map[string]string{"version": "1.0.0"}
	out, err := PrepareFilters(orig, "mypkg", func(fs map[string]string) error {
		delete(fs, "version")
		return nil
	})

	require.NoError(t, err)

	if _, exists := out["version"]; exists {
		t.Fatalf("expected version to be removed")
	}
	if out[FilterKeyName] != "mypkg" {
		t.Fatalf("expected name filter to be injected")
	}
	if _, mutated := orig["version"]; !mutated {
		t.Fatalf("original filters should not be mutated")
	}
}

func TestDefaultMatchers_ContainsAllExpectedFilters(t *testing.T) {
	expectedFilters := []string{
		FilterKeyName,
		FilterKeyRuntime,
		FilterKeyTools,
		FilterKeyTags,
		FilterKeyCategories,
		FilterKeyVersion,
		FilterKeyLicense,
		FilterKeySource,
	}

	matchers := DefaultMatchers()

	for _, expectedFilter := range expectedFilters {
		t.Run(expectedFilter, func(t *testing.T) {
			_, exists := matchers[expectedFilter]
			require.True(t, exists, "Expected filter %s to be present in DefaultMatchers", expectedFilter)
		})
	}

	require.Equal(t, len(expectedFilters), len(matchers), "Number of matchers should match expected filters")
}
