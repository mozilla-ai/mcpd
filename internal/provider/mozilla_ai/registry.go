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
	// RegistryName is the name of this registry that will appear as the Package.Source, and in logs/errors.
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
func (r *Registry) Resolve(name string, opt ...options.ResolveOption) (packages.Package, error) {
	// Handle name.
	name = filter.NormalizeString(name)
	if name == "" {
		return packages.Package{}, fmt.Errorf("name must not be empty")
	}

	// Handle options.
	opts, err := options.NewResolveOptions(opt...)
	if err != nil {
		return packages.Package{}, err
	}

	// Handle creation of filters.
	fs, err := options.PrepareFilters(options.ResolveFilters(opts), name, nil)
	if err != nil {
		return packages.Package{}, fmt.Errorf("invalid filters for %s: %w", r.ID(), err)
	}

	r.logger.Debug(
		"Resolving package",
		"name", name,
		"version", opts.Version,
		"runtime", opts.Runtime,
		"source", opts.Source,
		"filters", fs,
	)

	result, transformed := r.buildPackageResult(name)
	if !transformed {
		return packages.Package{}, fmt.Errorf("failed to build package result for '%s'", name)
	}

	combinedMatchOpts := append(slices.Clone(r.filterOptions), options.WithDefaultMatchers())
	matches, err := options.Match(result, fs, combinedMatchOpts...)
	if err != nil {
		return packages.Package{}, err
	}
	if !matches {
		return packages.Package{}, fmt.Errorf("package with name '%s' does not match requested filters", name)
	}

	return result, nil
}

// Search implements the PackageSearcher interface for Registry.
func (r *Registry) Search(
	name string,
	filters map[string]string,
	opt ...options.SearchOption,
) ([]packages.Package, error) {
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
	var results []packages.Package
	for id := range r.mcpServers {
		result, transformed := r.buildPackageResult(id)
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

// buildPackageResult attempts to convert the ServerDetail associated with the specified ID,
// into a Package.
func (r *Registry) buildPackageResult(pkgKey string) (packages.Package, bool) {
	// Sanity check to ensure things work when a random ID gets supplied.
	sd, foundServer := r.mcpServers[pkgKey]
	if !foundServer {
		r.logger.Warn("cannot transform package, unknown key", "pkgKey", pkgKey)
		return packages.Package{}, false
	}

	runtimesAndPackages, err := r.supportedRuntimePackageNames(sd.Installations)
	if err != nil || len(runtimesAndPackages) == 0 {
		r.logger.Debug(
			"no supported runtime packages found in registry",
			"pkgKey", pkgKey,
			"error", err,
		)
		return packages.Package{}, false
	}

	tools, err := sd.Tools.ToDomainType()
	if err != nil {
		r.logger.Error(
			"unable to convert tools to domain type",
			"name", pkgKey,
			"error", err,
		)
		return packages.Package{}, false
	}

	// Convert arguments to ArgumentMetadata format
	arguments, err := sd.Arguments.ToDomainType()
	if err != nil {
		r.logger.Error(
			"unable to convert arguments to domain type",
			"name", pkgKey,
			"error", err,
		)
		return packages.Package{}, false
	}

	// Determine transports - if specified in ServerDetail, use those, otherwise default to stdio
	transports := packages.DefaultTransports()
	if len(sd.Transports) > 0 {
		transports = packages.FromStrings(sd.Transports)
	}

	// Check if deprecated based on all installations
	installations := convertInstallations(sd.Installations, r.supportedRuntimes)
	deprecated := installations.AllDeprecated()

	return packages.Package{
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
		Transports:    transports,
		IsOfficial:    sd.IsOfficial,
		Deprecated:    deprecated || sd.Deprecated,
	}, true
}

// supportedRuntimePackageNames extracts runtime-specific package names for a given MCP server.
func (r *Registry) supportedRuntimePackageNames(
	installations map[string]Installation,
) (map[runtime.Runtime]string, error) {
	result := make(map[runtime.Runtime]string)

	specs := runtime.Specs()

	for name, inst := range installations {
		rt := runtime.Runtime(inst.Command)
		if _, ok := r.supportedRuntimes[rt]; !ok {
			continue
		}

		if !inst.isValid(name) {
			continue
		}

		spec, ok := specs[rt]
		if !ok || spec.ExtractPackageName == nil {
			r.logger.Debug("no package extractor for runtime", "runtime", rt)
			continue
		}

		pkg, err := spec.ExtractPackageName(inst.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to extract package for runtime %q: %w", rt, err)
		}

		result[rt] = pkg
	}

	return result, nil
}

func convertInstallations(
	src map[string]Installation,
	supported map[runtime.Runtime]struct{},
) packages.Installations {
	if src == nil {
		return nil
	}

	specs := runtime.Specs()
	details := make(packages.Installations, len(src))

	for name, install := range src {
		rt := runtime.Runtime(install.Command)
		if _, ok := supported[rt]; !ok {
			continue
		}

		if !install.isValid(name) {
			continue
		}

		pkg := ""
		if spec, ok := specs[rt]; ok && spec.ExtractPackageName != nil {
			if packageName, err := spec.ExtractPackageName(install.Args); err == nil {
				pkg = packageName
			}
		}

		details[rt] = packages.Installation{
			Command:     install.Command,
			Args:        slices.Clone(install.Args),
			Package:     pkg,
			Version:     install.Version,
			Env:         maps.Clone(install.Env),
			Description: install.Description,
			Recommended: install.Recommended,
			Deprecated:  install.Deprecated,
			Transports:  packages.DefaultTransports(), // mozilla-ai defaults to stdio, could be extended per-installation later
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
		InputSchema: packages.JSONSchema{
			Type:       t.InputSchema.Type,
			Properties: t.InputSchema.Properties,
			Required:   t.InputSchema.Required,
		},
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

// isValid checks an installation, and its name to ensure that the runtime,
// name and type align with expected values.
func (i *Installation) isValid(name string) bool {
	name = filter.NormalizeString(name)

	switch runtime.Runtime(i.Command) {
	case runtime.UVX:
		uvx := string(runtime.UVX)
		return name == uvx && i.Type == Runtime(uvx)
	case runtime.NPX:
		npx := string(runtime.NPX)
		npm := "npm"
		return (name == npm || name == npx) && (i.Type == Runtime(npm) || i.Type == Runtime(npx))
	case runtime.Docker:
		docker := string(runtime.Docker)
		return name == docker && i.Type == Runtime(docker)
	default:
		return false
	}
}
