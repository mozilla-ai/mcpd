//go:build validate_registry
// +build validate_registry

package main

import (
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run -tags=validate_registry ./tools/validate/registry.go <schema.json> <data.json>\n")
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

	fmt.Println("✅ Mozilla AI registry validation succeeded")
}
