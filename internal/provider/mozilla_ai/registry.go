package mozilla_ai

import (
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

//go:embed data/registry.json
var embeddedRegistryData embed.FS

const (
	// RegistryName is the name of this registry that will appear as the Server.Source, and in logs/errors.
	RegistryName = "mozilla-ai"
)

// Ensure Registry implements PackageProvider
var _ registry.PackageProvider = (*Registry)(nil)

// Registry implements the PackageRegistry interface for the Mozilla AI enriched server format.
type Registry struct {
	mcpServers        MCPRegistry
	logger            hclog.Logger
	supportedRuntimes map[runtime.Runtime]struct{}
	filterOptions     []options.Option
}

// NewRegistry creates a new Registry instance by loading data from the provided URL or embedded data.
// If url is empty, uses the embedded enriched servers data.
func NewRegistry(logger hclog.Logger, url string, opt ...runtime.Option) (*Registry, error) {
	// Handle all options.
	runtimeOpts, err := runtime.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	appSupported := runtimeOpts.SupportedRuntimes
	registrySupported := registrySupportedRuntimes()
	if !runtime.AnyIntersection(slices.Collect(maps.Keys(appSupported)), registrySupported) {
		return nil, fmt.Errorf(
			"no supported runtimes for %s registry: requires at least one of: %s",
			RegistryName,
			runtime.Join(registrySupported, ", "),
		)
	}

	// Handle retrieving the JSON data to bootstrap the registry.
	var servers MCPRegistry
	url = strings.TrimSpace(url)

	if url == "" {
		// Use embedded data
		data, err := embeddedRegistryData.ReadFile("data/registry.json")
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded registry data: %w", err)
		}
		err = json.Unmarshal(data, &servers)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedded servers data: %w", err)
		}
	} else {
		// Load from URL
		servers, err = runtime.LoadFromURL[MCPRegistry](url, RegistryName)
		if err != nil {
			return nil, err
		}
	}

	// Configure 'standard' filtering options that should always be included.
	l := logger.Named(RegistryName)

	// TODO: Is this needed?
	filterOpts := []options.Option{
		options.WithLogFunc(func(key, val string) {
			l.Warn("Unsupported filter/key", "filter", key, "value", val)
		}),
	}

	return &Registry{
		mcpServers:        servers,
		logger:            l,
		supportedRuntimes: runtimeOpts.SupportedRuntimes,
		filterOptions:     filterOpts,
	}, nil
}

// registrySupportedRuntimes declares the runtimes that this registry supports.
func registrySupportedRuntimes() []runtime.Runtime {
	return []runtime.Runtime{
		runtime.NPX,
		runtime.UVX,
	}
}

func (r *Registry) ID() string {
	return RegistryName
}

// Resolve implements the PackageGetter interface for Registry.
func (r *Registry) Resolve(name string, opt ...options.ResolveOption) (packages.Server, error) {
	// Handle name.
	name = filter.NormalizeString(name)
	if name == "" {
		return packages.Server{}, fmt.Errorf("name must not be empty")
	}

	// Handle options.
	opts, err := options.NewResolveOptions(opt...)
	if err != nil {
		return packages.Server{}, err
	}

	// Handle creation of filters.
	fs, err := options.PrepareFilters(options.ResolveFilters(opts), name, nil)
	if err != nil {
		return packages.Server{}, fmt.Errorf("invalid filters for %s: %w", r.ID(), err)
	}

	r.logger.Debug(
		"Resolving package",
		"name", name,
		"version", opts.Version,
		"runtime", opts.Runtime,
		"source", opts.Source,
		"filters", fs,
	)

	result, transformed := r.serverForID(name)
	if !transformed {
		return packages.Server{}, fmt.Errorf("failed to build package result for '%s'", name)
	}

	combinedMatchOpts := append(slices.Clone(r.filterOptions), options.WithDefaultMatchers())
	matches, err := options.Match(result, fs, combinedMatchOpts...)
	if err != nil {
		return packages.Server{}, err
	}
	if !matches {
		return packages.Server{}, fmt.Errorf("server with name '%s' does not match requested filters", name)
	}

	return result, nil
}

// Search implements the PackageSearcher interface for Registry.
func (r *Registry) Search(
	name string,
	filters map[string]string,
	opt ...options.SearchOption,
) ([]packages.Server, error) {
	name = filter.NormalizeString(name)
	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}

	opts, err := options.NewSearchOptions(opt...)
	if err != nil {
		return nil, err
	}

	fs, err := options.PrepareFilters(filters, name, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid filters for %s: %w", r.ID(), err)
	}

	r.logger.Debug("Searching for package", "name", name, "filters", fs, "source", opts.Source)
	var results []packages.Server
	for id := range r.mcpServers {
		result, transformed := r.serverForID(id)
		if !transformed {
			continue
		}
		combinedMatchOpts := append(slices.Clone(r.filterOptions), options.WithDefaultMatchers())
		matches, err := options.Match(result, fs, combinedMatchOpts...)
		if err != nil {
			return nil, err
		}
		if !matches {
			r.logger.Debug(
				"no match",
				"id", result.ID,
				"name", result.Name,
				"display-name", result.DisplayName,
				"filters", fs,
			)
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

// serverForID attempts to convert the Server associated with the specified ID, into a packages.Server.
// Returns the packages.Server and a boolean value indicating success.
func (r *Registry) serverForID(pkgKey string) (packages.Server, bool) {
	// Sanity check to ensure things work when a random ID gets supplied.
	sd, foundServer := r.mcpServers[pkgKey]
	if !foundServer {
		r.logger.Warn("cannot transform package, unknown key", "pkgKey", pkgKey)
		return packages.Server{}, false
	}

	tools, err := sd.Tools.ToDomainType()
	if err != nil {
		r.logger.Error(
			"unable to convert tools to domain type",
			"name", pkgKey,
			"error", err,
		)
		return packages.Server{}, false
	}

	// Convert arguments to ArgumentMetadata format
	arguments, err := sd.Arguments.ToDomainType()
	if err != nil {
		r.logger.Error(
			"unable to convert arguments to domain type",
			"name", pkgKey,
			"error", err,
		)
		return packages.Server{}, false
	}

	// Check if deprecated based on all installations
	installations := convertInstallations(sd.Installations, r.supportedRuntimes)
	deprecated := installations.AllDeprecated()

	return packages.Server{
		Source:        RegistryName,
		ID:            pkgKey,
		Name:          pkgKey,
		DisplayName:   sd.DisplayName,
		Description:   sd.Description,
		License:       sd.License,
		Tools:         tools,
		Tags:          sd.Tags,
		Categories:    sd.Categories,
		Installations: installations,
		Arguments:     arguments,
		IsOfficial:    sd.IsOfficial,
		Deprecated:    deprecated || sd.Deprecated,
	}, true
}

func convertInstallations(
	src map[string]Installation,
	supported map[runtime.Runtime]struct{},
) packages.Installations {
	if src == nil {
		return nil
	}

	details := make(packages.Installations, len(src))

	for _, install := range src {
		rt := runtime.Runtime(install.Runtime)
		if _, ok := supported[rt]; !ok {
			continue
		}

		transports := install.Transports
		if len(transports) == 0 {
			// TODO: mozilla-ai defaults to stdio, could be extended per-installation later.
			transports = packages.DefaultTransports().ToStrings()
		}

		details[rt] = packages.Installation{
			Runtime:     runtime.Runtime(install.Runtime),
			Package:     install.Package,
			Version:     install.Version,
			Description: install.Description,
			Recommended: install.Recommended,
			Deprecated:  install.Deprecated,
			Transports:  transports,
		}
	}

	return details
}

// ToDomainType converts Tools into the internal domain representation (packages.Tools).
func (t Tools) ToDomainType() (packages.Tools, error) {
	tools := make(packages.Tools, len(t))
	for i, tool := range t {
		data, err := tool.ToDomainType()
		if err != nil {
			return nil, err
		}
		tools[i] = data
	}

	return tools, nil
}

// ToDomainType converts a Tool into the internal domain representation (packages.Tool).
func (t Tool) ToDomainType() (packages.Tool, error) {
	return packages.Tool{
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
	}, nil
}

func (a Arguments) ToDomainType() (packages.Arguments, error) {
	args := make(packages.Arguments, len(a))
	for key, arg := range a {
		data, err := arg.ToDomainType()
		if err != nil {
			return nil, err
		}
		args[key] = data
	}

	return args, nil
}

func (a Argument) ToDomainType() (packages.ArgumentMetadata, error) {
	return packages.ArgumentMetadata{
		Name:         a.Name,
		Description:  a.Description,
		Required:     a.Required,
		VariableType: packages.VariableType(a.Type),
		Example:      a.Example,
		Position:     a.Position,
	}, nil
}
