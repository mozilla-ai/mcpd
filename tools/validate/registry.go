//go:build validate_registry
// +build validate_registry

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(
			os.Stderr,
			"Usage: go run -tags=validate_registry ./tools/validate/registry.go <schema.json> <data.json>\n",
		)
		os.Exit(1)
	}

	schemaFile := os.Args[1]
	dataFile := os.Args[2]

	// Get absolute paths for file URIs.
	absSchemaPath, err := os.ReadFile(schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading schema file: %v\n", err)
		os.Exit(1)
	}

	absDataPath, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data file: %v\n", err)
		os.Exit(1)
	}

	// Use JSON loaders instead of file loaders for better compatibility.
	schemaLoader := gojsonschema.NewBytesLoader(absSchemaPath)
	documentLoader := gojsonschema.NewBytesLoader(absDataPath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating: %v\n", err)
		os.Exit(1)
	}

	if !result.Valid() {
		fmt.Println("❌ Validation failed:")
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field(), err.Description())
		}
		os.Exit(1)
	}

	// Additional business rule validation: check that map keys match server IDs
	var registry map[string]any
	err = json.Unmarshal(absDataPath, &registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing registry data for business rule validation: %v\n", err)
		os.Exit(1)
	}

	for key, value := range registry {
		server, ok := value.(map[string]any)
		if !ok {
			continue
		}

		id, exists := server["id"].(string)
		// This should never be able to happen as we've validated the schema, but no harm in the sanity check.
		if !exists {
			fmt.Fprintf(os.Stderr, "❌ Server at key '%s' missing required ID field\n", key)
			os.Exit(1)
		}

		if id != key {
			fmt.Fprintf(
				os.Stderr,
				"❌ Registry inconsistency: registry key '%s' does not match server ID '%s'\n",
				key,
				id,
			)
			os.Exit(1)
		}
	}

	fmt.Println("✅ Mozilla AI registry validation succeeded")
}
