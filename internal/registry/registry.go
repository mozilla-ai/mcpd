package registry

import (
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
)

const registryName = "aggregator"

// Ensure Registry implements PackageProvider
var (
	_ PackageProvider = (*Registry)(nil)
)

// PackageSearcher defines the interface for searching for packages in a registry.
type PackageSearcher interface {
	// Search finds packages based on a query string and optional filters.
	// The name query should ideally be the name or package name (e.g., "time" or "mcp-server-time")
	// Filters can include "runtime", "tools", "license", "version", etc. (case-insensitive values).
	Search(name string, filters map[string]string, opt ...options.SearchOption) ([]packages.Package, error)
}

// PackageResolver defines the interface for retrieving a specific version of a package from a registry.
type PackageResolver interface {
	// Resolve retrieves a specific version of a package by its unique ID.
	// If version is not supplied as an option, it should use 'latest'.
	// This may be ignored if version filtering is not supported.
	Resolve(name string, opt ...options.ResolveOption) (packages.Package, error)
}

// PackageProvider defines the common interface for any type that can provide
// package search and retrieval capabilities.
type PackageProvider interface {
	PackageSearcher
	PackageResolver

	// ID returns the ID of this PackageResolver
	ID() string
}

type Builder interface {
	Build() (PackageProvider, error)
}

// Registry combines multiple PackageResolver implementations and allows searching
// and retrieving packages across all configured sources. This is intended to be
// the main entry point for package discovery in the application.
type Registry struct {
	logger     hclog.Logger
	registries map[string]PackageProvider
}

// NewRegistry creates a new Registry instance which will aggregate operations across the supplied package providers.
func NewRegistry(logger hclog.Logger, regs ...PackageProvider) (*Registry, error) {
	m := make(map[string]PackageProvider, len(regs))
	for _, r := range regs {
		id := r.ID()
		if _, exists := m[id]; exists {
			return nil, fmt.Errorf("duplicate registry ID detected: %s", id)
		}
		m[id] = r
	}
	return &Registry{
		registries: m,
		logger:     logger.Named(registryName),
	}, nil
}

func (r *Registry) ID() string {
	return registryName
}

// Resolve implements the PackageGetter interface for Registry.
// It attempts to retrieve a package by name from each contained registry in order,
// returning the first one that matches any optional supplied resolution criteria.
func (r *Registry) Resolve(name string, opt ...options.ResolveOption) (packages.Package, error) {
	// Handle name.
	name = filter.NormalizeString(name)
	if name == "" {
		return packages.Package{}, fmt.Errorf("name is required")
	}

	// Handle options.
	opts, err := options.NewResolveOptions(opt...)
	if err != nil {
		return packages.Package{}, err
	}

	r.logger.Debug(
		"Resolving package",
		"name", name,
		"version", opts.Version,
		"runtime", opts.Runtime,
		"source", opts.Source,
	)

	if opts.Source != "" {
		reg, ok := r.registries[opts.Source]
		if !ok {
			return packages.Package{}, fmt.Errorf("required source registry not found: %s", opts.Source)
		}

		result, err := reg.Resolve(name, opt...)
		if err != nil {
			r.logger.Error(
				"Error resolving package in registry",
				"registry", reg.ID(),
				"package", name,
				"error", err,
			)
			return packages.Package{}, fmt.Errorf(
				"error resolving package '%s' from registry '%s': %w",
				name,
				reg.ID(),
				err,
			)
		}

		return result, nil
	}

	// Search over registries, returning the first resolved package.
	for regName, reg := range r.registries {
		result, err := reg.Resolve(name, opt...)
		if err != nil {
			r.logger.Warn(
				"error getting package from registry",
				"name", name,
				"registry", regName,
				"error", err,
			)
			continue
		}
		return result, nil
	}

	err = fmt.Errorf(
		"package '%s', version '%s', runtime '%s' not found in any registry",
		name,
		opts.Version,
		opts.Runtime,
	)
	return packages.Package{}, err
}

// Search implements the PackageSearcher interface for Registry.
// It iterates through all contained registries, calls their Search method, and then aggregates and de-duplicates the results.
// Filters can be used to specify
func (r *Registry) Search(
	name string,
	filters map[string]string,
	opt ...options.SearchOption,
) ([]packages.Package, error) {
	// Handle name
	name = filter.NormalizeString(name)
	if name == "" {
		return []packages.Package{}, fmt.Errorf("name is required")
	}

	// Handle filters.
	fs, err := options.PrepareFilters(filters, name, nil)
	if err != nil {
		// Since the registry doesn't attempt to mutate the returned filters, we don't expect any errors.
		return []packages.Package{}, fmt.Errorf("unexpected error preparing filters for %s: %w", r.ID(), err)
	}

	// Handle options.
	opts, err := options.NewSearchOptions(opt...)
	if err != nil {
		return nil, err
	}

	var allResults []packages.Package

	// If a specific source registry was requested, only check that one for packages.
	if opts.Source != "" {
		reg, ok := r.registries[opts.Source]
		if !ok {
			return nil, fmt.Errorf("required source registry not found: %s", opts.Source)
		}
		results, err := reg.Search(name, fs, opt...)
		if err != nil {
			r.logger.Error("Error searching registry", "registry", reg.ID(), "error", err)
			return nil, err
		}

		return results, nil
	}

	// Search all registries for packages.
	for _, reg := range r.registries {
		results, err := reg.Search(name, fs, opt...)
		if err != nil {
			r.logger.Warn("Error searching registry ... continuing", "registry", reg.ID(), "error", err)
			continue // Continue searching other registries even if one fails.
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}
