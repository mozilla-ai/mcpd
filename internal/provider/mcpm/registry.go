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
	// RegistryName is the name of this registry that will appear as the Package.Source, and in logs/errors.
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
	fs, err := options.PrepareFilters(options.ResolveFilters(opts), name, func(fs map[string]string) error {
		// Handle lack of 'version' support in mcpm.
		if v, ok := fs[options.FilterKeyVersion]; ok {
			r.logger.Warn(
				"'version' not supported on resolve operation, returning latest known definition",
				"name", name,
				options.FilterKeyVersion, v)
			// Clear 'version' for mcpm as it cannot be used.
			delete(fs, options.FilterKeyVersion)
		}
		return nil
	})
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

	fs, err := options.PrepareFilters(filters, name, func(fs map[string]string) error {
		// Handle lack of 'version' support in mcpm.
		if v, ok := fs[options.FilterKeyVersion]; ok {
			r.logger.Warn(
				"'version' not supported on search operation, returning latest known definition",
				"name", name,
				options.FilterKeyVersion, v,
			)
			// Clear 'version' for mcpm as it cannot be used.
			delete(fs, options.FilterKeyVersion)
		}
		return nil
	})
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

// buildPackageResult attempts to convert the MCPServer associated with the specified ID,
// into a Package.
// Returns the transformed result, and a flag to indicate if the transformation was successful.
// If the server cannot be transformed due to unsupported or malformed runtime installations, false is returned.
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

	var runtimes []runtime.Runtime
	for rt := range runtimesAndPackages {
		runtimes = append(runtimes, rt)
	}
	slices.Sort(runtimes)

	tools, err := sd.Tools.ToDomainType()
	if err != nil {
		r.logger.Error(
			"unable to convert tools to domain type",
			"name", pkgKey,
			"error", err,
		)
		return packages.Package{}, false
	}

	// Analyze actual runtime variables and convert to ArgumentMetadata format
	arguments := extractArgumentMetadata(sd, r.supportedRuntimes)

	return packages.Package{
		Source:              RegistryName,
		ID:                  pkgKey,
		Name:                pkgKey,
		DisplayName:         sd.DisplayName,
		Description:         sd.Description,
		License:             sd.License,
		Tools:               tools,
		Tags:                sd.Tags,
		Categories:          sd.Categories,
		Runtimes:            runtimes,
		InstallationDetails: convertInstallations(sd.Installations, r.supportedRuntimes),
		Arguments:           arguments,
		IsOfficial:          sd.IsOfficial,
	}, true
}

type RuntimeSpec struct {
	ShouldIgnoreFlag   func(string) bool
	ExtractPackageName func([]string) (string, error)
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
		cliMeta := extractCLIArgMetadata(inst.Args, schema, spec)
		for k, v := range cliMeta {
			out[k] = v
		}
	}

	return out
}

func extractEnvMetadata(
	env map[string]string,
	schema map[string]Argument,
) map[string]packages.ArgumentMetadata {
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
			VariableType: packages.VariableTypeEnv,
			Required:     m.Required,
			Description:  m.Description,
		}
	}

	return out
}

func extractCLIArgMetadata(
	args []string,
	schema map[string]Argument,
	spec runtime.Spec,
) map[string]packages.ArgumentMetadata {
	out := make(map[string]packages.ArgumentMetadata)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		flag := extractActualCommandLineFlag(arg)
		if flag == "" || (spec.ShouldIgnoreFlag != nil && spec.ShouldIgnoreFlag(flag)) {
			continue
		}

		metaKey := flag

		// Check placeholder in current arg
		if placeholder := extractPlaceholder(arg); placeholder != "" {
			if _, ok := schema[placeholder]; ok {
				metaKey = placeholder
				m := schema[metaKey]
				out[flag] = packages.ArgumentMetadata{
					VariableType: packages.VariableTypeArg,
					Required:     m.Required,
					Description:  m.Description,
				}
				continue
			}
		}

		// Determine if boolean flag (no value expected)
		nextArgIndex := i + 1
		isBoolFlag := nextArgIndex >= len(args) || strings.HasPrefix(args[nextArgIndex], "--")

		// Check placeholder in next arg if not bool
		if !isBoolFlag && nextArgIndex < len(args) {
			if placeholder := extractPlaceholder(args[nextArgIndex]); placeholder != "" {
				if _, ok := schema[placeholder]; ok {
					metaKey = placeholder
				}
			}
		}

		m := schema[metaKey]
		var vt packages.VariableType
		if isBoolFlag {
			vt = packages.VariableTypeArgBool
		} else {
			vt = packages.VariableTypeArg
		}

		out[flag] = packages.ArgumentMetadata{
			VariableType: vt,
			Required:     m.Required,
			Description:  m.Description,
		}
	}

	return out
}

func extractPlaceholder(s string) string {
	if matches := packages.EnvVarPlaceholderRegex.FindStringSubmatch(s); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

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

func convertInstallations(
	src map[string]Installation,
	supported map[runtime.Runtime]struct{},
) map[runtime.Runtime]packages.Installation {
	if src == nil {
		return nil
	}

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
			Command:     install.Command,
			Args:        slices.Clone(install.Args),
			Package:     pkg,
			Env:         maps.Clone(install.Env),
			Description: install.Description,
			Recommended: install.Recommended,
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
		InputSchema: packages.JSONSchema{
			Type:       t.InputSchema.Type,
			Properties: t.InputSchema.Properties,
			Required:   t.InputSchema.Required,
		},
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

// extractActualCommandLineFlag extracts the actual flag name from a command line argument
// e.g., "--local-timezone=${TZ}" returns "--local-timezone"
// e.g., "--verbose" returns "--verbose"
// NOTE: Only supports long flag style at present
func extractActualCommandLineFlag(arg string) string {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) > 0 && strings.HasPrefix(parts[0], "--") {
			return parts[0]
		}
	} else if strings.HasPrefix(arg, "--") {
		// Simple flag without assignment
		return arg
	}
	return ""
}
