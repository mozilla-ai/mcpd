//go:build docsgen_api
// +build docsgen_api

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/internal/api"
	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/tools/docsgen/apiref"
)

// stubHealthTracker provides a stub implementation for documentation generation.
type stubHealthTracker struct{}

func (s *stubHealthTracker) Status(string) (domain.ServerHealth, error) {
	return domain.ServerHealth{}, nil
}
func (s *stubHealthTracker) List() []domain.ServerHealth                              { return nil }
func (s *stubHealthTracker) Update(string, domain.HealthStatus, *time.Duration) error { return nil }
func (s *stubHealthTracker) Add(string)                                               {}
func (s *stubHealthTracker) Remove(string)                                            {}

// stubClientManager provides a stub implementation for documentation generation.
type stubClientManager struct{}

func (s *stubClientManager) Add(string, client.MCPClient, []string) {}
func (s *stubClientManager) Client(string) (client.MCPClient, bool) { return nil, false }
func (s *stubClientManager) Tools(string) ([]string, bool)          { return nil, false }
func (s *stubClientManager) UpdateTools(string, []string) error     { return nil }
func (s *stubClientManager) List() []string                         { return nil }
func (s *stubClientManager) Remove(string)                          {}

// main generates the OpenAPI specification for the mcpd API.
// It assumes it is run from the repository root.
func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "mcpd.docsgen.api",
		Level:  hclog.Info,
		Output: os.Stderr,
	})

	// Output path for the OpenAPI spec, relative to the repository root.
	outputPath := "./docs/api/openapi.yaml"

	// Create a chi router (same as the daemon).
	mux := chi.NewMux()
	mux.Use(middleware.StripSlashes)

	// Create Huma config and router (same as the daemon).
	config := huma.DefaultConfig("mcpd docs", api.APIVersion)
	router := humachi.New(mux, config)

	// Register routes using stub dependencies.
	// The OpenAPI spec generation only needs the route definitions, not the actual handlers.
	apiPathPrefix, err := api.RegisterRoutes(router, &stubHealthTracker{}, &stubClientManager{})
	if err != nil {
		logger.Error("failed to register API routes", "error", err)
		os.Exit(1)
	}

	logger.Info("Routes registered", "prefix", apiPathPrefix)

	// Get the OpenAPI spec as YAML.
	spec := router.OpenAPI()
	yamlBytes, err := spec.YAML()
	if err != nil {
		logger.Error("failed to generate OpenAPI YAML", "error", err)
		os.Exit(1)
	}

	// Ensure the docs directory exists.
	docsDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		logger.Error("failed to create docs directory", "path", docsDir, "error", err)
		os.Exit(1)
	}

	// Write the YAML to the output file.
	if err := os.WriteFile(outputPath, yamlBytes, 0o644); err != nil {
		logger.Error("failed to write OpenAPI spec", "path", outputPath, "error", err)
		os.Exit(1)
	}

	logger.Info("OpenAPI spec generated", "path", outputPath, "size", fmt.Sprintf("%d bytes", len(yamlBytes)))

	// Write the GitBook API reference page derived from the same spec.
	refPath := "./docs/api-reference.md"
	if err := os.WriteFile(refPath, []byte(apiref.Render(operations(spec))), 0o644); err != nil {
		logger.Error("failed to write API reference", "path", refPath, "error", err)
		os.Exit(1)
	}

	logger.Info("API reference generated", "path", refPath)
}

// operations flattens the spec's paths into the operation list the API
// reference page is rendered from.
func operations(spec *huma.OpenAPI) []apiref.Operation {
	var ops []apiref.Operation
	for path, item := range spec.Paths {
		for method, op := range methodOperations(item) {
			if op == nil {
				continue
			}
			tag := ""
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}
			ops = append(ops, apiref.Operation{Tag: tag, Method: method, Path: path})
		}
	}
	return ops
}

// methodOperations maps each HTTP method to its operation on a path item.
func methodOperations(item *huma.PathItem) map[string]*huma.Operation {
	return map[string]*huma.Operation{
		"get":     item.Get,
		"put":     item.Put,
		"post":    item.Post,
		"delete":  item.Delete,
		"options": item.Options,
		"head":    item.Head,
		"patch":   item.Patch,
		"trace":   item.Trace,
	}
}
