package registry

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"
)

// PackageSearcher defines the interface for searching for packages in a registry.
type PackageSearcher interface {
	// Search finds packages based on a query string and optional filters.
	// The name query should ideally be the name or package name (e.g., "time" or "mcp-server-time")
	// Filters can include "runtime", "tool", "transport", "version", etc. (case-insensitive values).
	Search(name string, filters map[string]string) ([]types.PackageResult, error)
}

// PackageGetter defines the interface for retrieving a specific version of a package from a registry.
type PackageGetter interface {
	// Get retrieves a specific version of a package by its unique ID.
	// If version is not supplied as an option, it should use 'latest'.
	// This may be ignored if version filtering is not supported.
	Get(id string, opts ...types.GetterOption) (types.PackageResult, error)
}

// PackageResolver defines the common interface for any type that can provide
// package search and retrieval capabilities.
type PackageResolver interface {
	PackageSearcher
	PackageGetter
	// ID returns the ID of this PackageResolver
	ID() string
}

type GetOptions struct {
	Version string
	Runtime types.Runtime
	Tools   []string
	// Add more fields as needed
}

// Registry combines multiple PackageResolver implementations and allows searching
// and retrieving packages across all configured sources. This is intended to be
// the main entry point for package discovery in the application.
type Registry struct {
	logger     hclog.Logger
	registries map[string]PackageResolver
}

// NewRegistry creates a new Registry instance with the given individual package registries.
func NewRegistry(logger hclog.Logger, regs ...PackageResolver) (*Registry, error) {
	m := make(map[string]PackageResolver, len(regs))
	for _, r := range regs {
		id := r.ID()
		if _, exists := m[id]; exists {
			return nil, fmt.Errorf("duplicate PackageResolver ID detected: %q", id)
		}
		m[id] = r
	}
	return &Registry{
		registries: m,
		logger:     logger.Named("aggregator"),
	}, nil
}

func (r *Registry) ID() string {
	return "aggregator" // TODO: sort out naming.
}

// Search implements the PackageSearcher interface for Registry.
// It iterates through all contained registries, calls their Search method,
// and then aggregates and de-duplicates the results.
// The name parameter should be the raw package string from config.ServerEntry.Package,
// including any prefixes like "uvx::" and version suffixes like "@0.6.2". This method will handle parsing.
func (r *Registry) Search(name string, filters map[string]string) ([]types.PackageResult, error) {
	var allResults []types.PackageResult
	// Use a map to track seen IDs to avoid duplicates across registries.
	seenAggregatedIDs := make(map[string]struct{})

	// Parse the name string to extract runtime, base ID, and version
	runtime, baseID, version := parsePackageString(name)

	// Add parsed runtime and version as filters
	if filters == nil {
		filters = make(map[string]string)
	}
	if runtime != "" {
		filters["runtime"] = runtime
	}
	if version != "" {
		filters["version"] = version // Add version filter
	}
	// Always inject name filter
	filters["name"] = name

	for _, reg := range r.registries {
		// Pass the *parsed* base ID (as the name) and the updated filters to individual registries
		results, err := reg.Search(baseID, filters)
		if err != nil {
			// TODO: logger
			fmt.Printf("Error searching registry %T: %v\n", reg, err)
			continue // Continue searching other registries even if one fails
		}
		for _, res := range results {
			// Create a unique ID for aggregation across different sources
			aggregatedID := fmt.Sprintf("%s_%s", res.Source, res.ID)
			if _, seen := seenAggregatedIDs[aggregatedID]; !seen {
				allResults = append(allResults, res)
				seenAggregatedIDs[aggregatedID] = struct{}{}
			}
		}
	}
	return allResults, nil
}

// Get implements the PackageGetter interface for Registry.
// It attempts to retrieve a package by ID from each contained registry in order,
// returning the first one found that matches the runtime and version criteria.
// The id parameter should be the raw package string from config.ServerEntry.Package,
// including any prefixes like "uvx::" and version suffixes like "@0.6.2".
func (r *Registry) Get(id string, opts ...types.GetterOption) (types.PackageResult, error) {
	// Parse options
	options, err := types.GetGetterOpts(opts...)
	if err != nil {
		return types.PackageResult{}, err
	}

	r.logger.Debug("Getting package", "id", id, "version", options.Version)

	// Parse the id string to extract runtime, base ID, and version
	// Note: We use the 'version' parameter passed to Get first, then fallback to parsing it from 'id'
	// if 'version' parameter is empty. This allows explicit 'version' overrides.
	requestedRuntime, baseID, _ := parsePackageString(id)

	getOpts := []types.GetterOption{types.WithVersion(options.Version)}
	if requestedRuntime != "" {
		getOpts = append(getOpts, types.WithRuntime(types.Runtime(requestedRuntime)))
	}

	for regName, reg := range r.registries {
		// Attempt to get the package using the parsed base ID and determined version
		result, err := reg.Get(baseID, getOpts...)
		if err != nil {
			r.logger.Error("error getting package from registry", "registry", regName, "package", baseID, "error", err)
			continue
		}

		// If a specific runtime was requested via prefix, ensure the found package supports it
		if requestedRuntime != "" {
			foundRuntime := false
			for _, r := range result.Runtimes {
				if strings.EqualFold(r, requestedRuntime) {
					foundRuntime = true
					break
				}
			}
			if !foundRuntime {
				// Package found by ID and version, but doesn't support the requested runtime prefix
				r.logger.Warn("Package does not support requested runtime ", "registry", regName, "package", baseID, "runtime", requestedRuntime)
				continue
			}
		}

		return result, nil
	}

	return types.PackageResult{}, fmt.Errorf("package not found in any registry: package '%s', version '%s', runtime '%s'", baseID, options.Version, requestedRuntime)
}

// parsePackageString extracts runtime, base ID, and version from a package string like "uvx::time@0.6.2".
func parsePackageString(packageString string) (runtime, baseID, version string) {
	parts := strings.SplitN(packageString, "::", 2)
	if len(parts) == 2 {
		runtime = parts[0]
		packageString = parts[1] // Remaining part after runtime prefix
	}

	// Check for version suffix
	if atIndex := strings.LastIndex(packageString, "@"); atIndex != -1 {
		baseID = packageString[:atIndex]
		version = packageString[atIndex+1:]
	} else {
		baseID = packageString
		version = "" // Default to empty string, which Get will treat as 'latest'
	}
	return runtime, baseID, version
}
