package mcpm

import (
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

const (
	// RegistryName is the name of this registry that will appear as the Server.Source, and in logs/errors.
	RegistryName = "mcpm"

	// ManifestURL is the URL at which the servers JSON file for the registry can be found for MCPM.
	ManifestURL = "https://mcpm.sh/api/servers.json"
)

// Ensure Registry implements PackageProvider
var _ registry.PackageProvider = (*Registry)(nil)

// Registry implements the PackageRegistry interface for the MCPM server JSON format.
type Registry struct {
	mcpServers        MCPServers
	logger            hclog.Logger
	supportedRuntimes map[runtime.Runtime]struct{}
	filterOptions     []options.Option
}

// NewRegistry creates a new Registry instance by fetching its data from the provided URL.
// The url is the URL of the JSON manifest for this registry.
func NewRegistry(logger hclog.Logger, url string, opt ...runtime.Option) (*Registry, error) {
	// Handle URL.
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty MCPM registry URL is invalid")
	}

	// Handle all options.
	runtimeOpts, err := runtime.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	appSupported := runtimeOpts.SupportedRuntimes
	registrySupported := registrySupportedRuntimes()
	if !runtime.AnyIntersection(slices.Collect(maps.Keys(appSupported)), registrySupported) {
		return nil, fmt.Errorf(
			"no supported runtimes for mcpm registry: requires at least one of: %s",
			runtime.Join(registrySupported, ", "),
		)
	}

	// Handle retrieving the JSON data to bootstrap the registry.
	servers, err := runtime.LoadFromURL[MCPServers](url, RegistryName)
	if err != nil {
		return nil, err
	}

	// Configure 'standard' filtering options that should always be included.
	// e.g. for unsupported 'version'.
	l := logger.Named(RegistryName)
	filterOpts := []options.Option{
		options.WithUnsupportedKeys(options.FilterKeyVersion),
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
// It retrieves a specific package by its name.
// The 'version' parameter is not supported for filtering.
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

	// Handle unsupported filters.
	if opts.Version != "" {
		return packages.Server{}, fmt.Errorf("version is not supported by '%s' registry", r.ID())
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

	// Handle unsupported filters.
	if v, ok := filters[options.FilterKeyVersion]; ok && v != "" {
		return nil, fmt.Errorf("version is not supported by '%s' registry", r.ID())
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

// serverForID attempts to convert the MCPServer associated with the specified ID,
// into a Server.
// Returns the transformed result, and a flag to indicate if the transformation was successful.
// If the server cannot be transformed due to unsupported or malformed runtime installations, false is returned.
func (r *Registry) serverForID(pkgKey string) (packages.Server, bool) {
	// Sanity check to ensure things work when a random ID gets supplied.
	sd, foundServer := r.mcpServers[pkgKey]
	if !foundServer {
		r.logger.Warn("cannot transform package, unknown key", "pkgKey", pkgKey)
		return packages.Server{}, false
	}

	runtimesAndPackages, err := r.supportedRuntimePackageNames(sd.Installations)
	if err != nil || len(runtimesAndPackages) == 0 {
		r.logger.Debug(
			"no supported runtime packages found in registry",
			"pkgKey", pkgKey,
			"error", err,
		)
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

	// Analyze actual runtime variables and convert to ArgumentMetadata format
	arguments := extractArgumentMetadata(sd, r.supportedRuntimes)
	installations := convertInstallations(sd, r.supportedRuntimes)

	return packages.Server{
		Arguments:     arguments,
		Categories:    sd.Categories,
		Deprecated:    false, // MCPM doesn't support deprecated packages
		Description:   sd.Description,
		DisplayName:   sd.DisplayName,
		Homepage:      sd.Homepage,
		ID:            pkgKey,
		Installations: installations,
		IsOfficial:    sd.IsOfficial,
		License:       sd.License,
		Name:          pkgKey,
		Publisher: packages.Publisher{
			Name: sd.Author.Name,
		},
		Source: RegistryName,
		Tags:   sd.Tags,
		Tools:  tools,
	}, true
}

// isValid checks an installation, and it's name to ensure that the runtime,
// name and type align with expected values.
func (i *Installation) isValid(name string) bool {
	name = filter.NormalizeString(name)

	switch runtime.Runtime(i.Command) {
	case runtime.UVX:
		uvx := string(runtime.UVX)
		return name == uvx && i.Type == uvx
	case runtime.NPX:
		npx := string(runtime.NPX)
		npm := "npm"
		return (name == npm || name == npx) && (i.Type == npm || i.Type == npx)
	case runtime.Docker:
		docker := string(runtime.Docker)
		return name == docker && i.Type == docker
	default:
		return false
	}
}

// extractArgumentMetadata extracts and consolidates argument metadata from server installations.
// It processes both environment variables and CLI arguments, with environment variables taking precedence.
func extractArgumentMetadata(
	server MCPServer,
	supported map[runtime.Runtime]struct{},
) map[string]packages.ArgumentMetadata {
	schema := server.Arguments
	out := make(map[string]packages.ArgumentMetadata)

	for name, inst := range server.Installations {
		rt := runtime.Runtime(inst.Command)
		if _, ok := supported[rt]; !ok {
			continue
		}
		if !inst.isValid(name) {
			continue
		}

		spec := runtime.Specs()[rt]

		// Extract environment variables metadata
		envMeta := extractEnvMetadata(inst.Env, schema)
		for k, v := range envMeta {
			out[k] = v
		}

		// Extract CLI arguments metadata
		parser := NewCLIArgParser(schema, spec)
		cliMeta := parser.Parse(inst.Args)
		for k, v := range cliMeta {
			// Only add CLI metadata if not already present as an env var
			// Environment variables take precedence over CLI arguments
			if _, exists := out[k]; !exists {
				out[k] = v
			}
		}
	}

	return out
}

// extractEnvMetadata extracts environment variable metadata from an installation's environment settings.
// It maps environment variable names to their schema definitions.
func extractEnvMetadata(env map[string]string, schema Arguments) map[string]packages.ArgumentMetadata {
	out := make(map[string]packages.ArgumentMetadata)

	for envName, envVal := range env {
		metaKey := envName
		if placeholder := extractPlaceholder(envVal); placeholder != "" {
			if _, ok := schema[placeholder]; ok {
				metaKey = placeholder
			}
		}
		m := schema[metaKey] // zero-value if missing
		out[envName] = packages.ArgumentMetadata{
			Name:         envName,
			VariableType: packages.VariableTypeEnv,
			Required:     m.Required,
			Description:  m.Description,
			Example:      m.Example,
		}
	}

	return out
}

// isFlag checks if an argument is a flag (starts with --)
func isFlag(arg string) bool {
	return strings.HasPrefix(arg, "--")
}

// extractPlaceholder extracts a placeholder variable name from a string like ${VAR_NAME}.
// Returns the variable name without the ${} wrapper, or empty string if no placeholder found.
func extractPlaceholder(s string) string {
	if matches := packages.EnvVarPlaceholderRegex.FindStringSubmatch(s); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// shouldIgnoreFlag determines if a flag should be ignored for a given runtime.
// Returns true for runtime-specific flags that shouldn't be exposed as user arguments.
func shouldIgnoreFlag(rt runtime.Runtime, flag string) bool {
	switch rt {
	case runtime.Docker:
		switch flag {
		case "--rm", "--name", "--volume", "-v", "--network", "--detach", "-d", "-i":
			return true
		}
	case runtime.Python:
		if flag == "-m" {
			return true
		}
	case runtime.NPX:
		if flag == "-y" {
			return true
		}
	}
	return false
}

// convertInstallations converts MCPM installation data to internal package format.
// Only includes installations for supported runtimes with valid configurations.
func convertInstallations(
	srv MCPServer,
	supported map[runtime.Runtime]struct{},
) map[runtime.Runtime]packages.Installation {
	if srv.Installations == nil {
		return nil
	}

	src := srv.Installations

	specs := runtime.Specs()
	details := make(map[runtime.Runtime]packages.Installation, len(src))

	for name, install := range src {
		if !install.isValid(name) {
			continue
		}

		rt := runtime.Runtime(install.Command)
		if _, ok := supported[rt]; !ok {
			continue
		}

		pkg := ""
		if spec, ok := specs[rt]; ok && spec.ExtractPackageName != nil {
			if name, err := spec.ExtractPackageName(install.Args); err == nil {
				pkg = name
			}
		}

		details[rt] = packages.Installation{
			Runtime:     runtime.Runtime(install.Command),
			Package:     pkg,
			Description: install.Description,
			Recommended: install.Recommended,
			Deprecated:  false, // MCPM doesn't support deprecated installations
			Transports:  packages.DefaultTransports().ToStrings(),
			Repository: &packages.Repository{
				Type: srv.Repository.Type,
				URL:  srv.Repository.URL,
			},
		}
	}

	return details
}

// ToDomainType converts a Tool into the internal domain representation (packages.Tool).
func (t Tool) ToDomainType() (packages.Tool, error) {
	return packages.Tool{
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
	}, nil
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

// supportedRuntimePackageNames extracts runtime-specific package names for a given MCP server.
// It returns a map where keys are supported runtime identifiers (e.g., "npx", "uvx") and values are
// the corresponding plain package names used to execute the server.
//
// Only installations where the command matches a supported runtime may be included.
// An error is returned if a supported runtime is found but a valid package name cannot be extracted.
func (r *Registry) supportedRuntimePackageNames(
	installations map[string]Installation,
) (map[runtime.Runtime]string, error) {
	result := make(map[runtime.Runtime]string)

	specs := runtime.Specs()

	for _, inst := range installations {
		rt := runtime.Runtime(inst.Command)
		if _, ok := r.supportedRuntimes[rt]; !ok {
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

// extractFlagName extracts the flag name from arguments like "--flag" or "--flag=value"
func extractFlagName(arg string) string {
	if !strings.HasPrefix(arg, "--") {
		return ""
	}

	if idx := strings.Index(arg, "="); idx > 0 {
		return arg[:idx]
	}
	return arg
}

// extractFlagValue extracts the value portion from "--flag=value" style arguments
func extractFlagValue(arg string) string {
	if idx := strings.Index(arg, "="); idx > 0 && idx < len(arg)-1 {
		return arg[idx+1:]
	}
	return ""
}
