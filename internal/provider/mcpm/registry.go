package mcpm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/filter"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/packages"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/runtime"
)

const registryName = "mcpm"

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

	// Configure 'standard' filtering options that should always be included.
	// e.g. for unsupported 'version'.
	l := logger.Named(registryName)
	filterOpts := []options.Option{
		options.WithUnsupportedKeys("version"),
		options.WithLogFunc(func(key, val string) {
			l.Warn("Unsupported filter/key", "filter", key, "value", val)
		}),
	}

	// Handle retrieving the JSON data to bootstrap the registry.
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch '%s' registry data from URL '%s': %w", registryName, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status from '%s' registry for URL '%s': %d", registryName, url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read '%s' registry response body from '%s': %w", registryName, url, err)
	}

	var servers MCPServers
	if err := json.Unmarshal(body, &servers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal '%s' registry JSON from '%s': %w", registryName, url, err)
	}

	return &Registry{
		mcpServers:        servers,
		logger:            l,
		supportedRuntimes: runtimeOpts.SupportedRuntimes,
		filterOptions:     filterOpts,
	}, nil
}

func (r *Registry) ID() string {
	return registryName
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
				"version", v)
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
func (r *Registry) Search(name string, filters map[string]string, opt ...options.SearchOption) ([]packages.Package, error) {
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
				"version", v)
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
		r.logger.Warn(
			"transformation to server details failed, MCP server package not found",
			"pkgKey", pkgKey,
		)
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
	pkgName := runtimesAndPackages[runtimes[0]]

	tools := make([]string, 0, len(sd.Tools))
	for _, tool := range sd.Tools {
		tools = append(tools, tool.Name)
	}

	// Analyze actual runtime variables
	runtimeVariables := r.analyzeServerArguments(sd)

	// Convert runtime variables to ArgumentMetadata format
	arguments := make(map[string]packages.ArgumentMetadata, len(runtimeVariables))
	for varName, rtVar := range runtimeVariables {
		arguments[varName] = packages.ArgumentMetadata{
			VariableType: rtVar.VariableType,
			Required:     rtVar.Required,
			Description:  rtVar.Description,
		}
	}

	// Extract just the environment variables for backward compatibility
	configurableEnvVars := make([]string, 0)
	for varName, rtVar := range runtimeVariables {
		if rtVar.VariableType == packages.VariableTypeEnv {
			configurableEnvVars = append(configurableEnvVars, varName)
		}
	}
	slices.Sort(configurableEnvVars)

	return packages.Package{
		ID:                  pkgKey,
		Source:              registryName,
		Name:                pkgName,
		DisplayName:         sd.DisplayName,
		Description:         sd.Description,
		License:             sd.License,
		Tools:               tools,
		Runtimes:            runtimes,
		InstallationDetails: convertInstallations(sd.Installations),
		Arguments:           arguments,
		ConfigurableEnvVars: configurableEnvVars,
	}, true
}

func convertInstallations(src map[string]Installation) map[runtime.Runtime]packages.Installation {
	if src == nil {
		return nil
	}

	details := make(map[runtime.Runtime]packages.Installation, len(src))
	for _, install := range src {
		details[runtime.Runtime(install.Command)] = packages.Installation{
			Args:        slices.Clone(install.Args),
			Package:     install.Package,
			Env:         maps.Clone(install.Env),
			Description: install.Description,
			Recommended: install.Recommended,
		}
	}
	return details
}

// supportedRuntimePackageNames extracts runtime-specific package names for a given MCP server.
// It returns a map where keys are supported runtime identifiers (e.g., "npx", "uvx") and values are
// the corresponding plain package names used to execute the server.
//
// Only installations where the command matches a supported runtime are included.
// Additionally, the installation must use a standard execution format—plain package names like "some-package" or
// "some-package@version". Non-standard forms such as git URLs, Python scripts, or direct file paths
// are ignored.
// An error is returned if a supported runtime is found but a valid package name cannot be extracted.
func (r *Registry) supportedRuntimePackageNames(installations map[string]Installation) (map[runtime.Runtime]string, error) {
	result := make(map[runtime.Runtime]string)

	for _, inst := range installations {
		// MCPM's registry is a bit inconsistent around npm/npx.
		// Sometimes an installation key is npm, sometimes npx.
		// Sometimes, the key is npx, but the type is npm, etc. it seems the only consistent thing to
		// index on is the actual 'command' which shows npx consistently.
		rt := runtime.Runtime(inst.Command)
		if _, ok := r.supportedRuntimes[rt]; !ok {
			continue
		}

		pkg, err := extractPlainPackage(rt, inst.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to extract package for runtime %q: %w", rt, err)
		}
		result[rt] = pkg
	}

	return result, nil
}

// extractPlainPackage scans a slice of command-line arguments and returns the first valid
// package identifier. It skips flags (e.g., "-y"), interpolated env vars ("${FOO}"),
// URLs, git references, and script files (".py").
//
// This function enforces a strict format, ensuring only plain package names are accepted.
// Returns an error if no suitable package name is found.
func extractPlainPackage(rt runtime.Runtime, args []string) (string, error) {
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-"), strings.HasPrefix(arg, "${"):
			continue
		case strings.HasPrefix(arg, "git+"), strings.HasSuffix(arg, ".py"):
			continue
		case rt == runtime.UVX && strings.HasPrefix(arg, "https://"),
			rt == runtime.UVX && strings.HasPrefix(arg, "http://"):
			continue

		default:
			return arg, nil
		}
	}
	return "", errors.New("no valid plain package name found in args")
}

// TODO: Relocate?
type RuntimeVariable struct {
	Name         string                       `json:"name"`          // Actual name used at runtime
	VariableType packages.VariableType        `json:"variable_type"` // "environment" or "argument"
	Runtimes     map[runtime.Runtime]struct{} `json:"runtimes"`      // Which runtimes use this variable
	Required     bool                         `json:"required"`      // If we can determine this
	Description  string                       `json:"description"`   // If available from Arguments map
}

// extractActualCommandLineFlag extracts the actual flag name from a command line argument
// e.g., "--local-timezone=${TZ}" returns "--local-timezone"
// e.g., "--verbose" returns "--verbose"
func extractActualCommandLineFlag(arg string) string {
	if strings.Contains(arg, "=") {
		parts := strings.Split(arg, "=")
		if len(parts) > 0 && strings.HasPrefix(parts[0], "--") {
			return parts[0]
		}
	} else if strings.HasPrefix(arg, "--") {
		// Simple flag without assignment
		return arg
	}
	return ""
}

func (r *Registry) analyzeServerArguments(server MCPServer) map[string]RuntimeVariable {
	// Map from schema argument names to their metadata (for descriptions, etc.)
	schemaArgs := server.Arguments
	runtimeVars := make(map[string]RuntimeVariable)

	for installType, installation := range server.Installations {
		rt := runtime.Runtime(installType) // TODO: Check key vs. the installation.Command
		// Skip args for runtimes we don't support.
		if _, ok := r.supportedRuntimes[rt]; !ok {
			continue
		}
		// Extract actual environment variables (left side of env mappings)
		for envName, envValue := range installation.Env {
			if rtv, exists := runtimeVars[envName]; exists {
				rtv.Runtimes[rt] = struct{}{}
				continue
			}

			// Create new runtime variable
			rtVar := RuntimeVariable{
				Name:         envName,
				VariableType: packages.VariableTypeEnv,
				Runtimes:     map[runtime.Runtime]struct{}{rt: {}},
			}

			// Try to find description from schema arguments
			// Check if env value is a placeholder that matches a schema arg
			if matches := packages.EnvVarPlaceholderRegex.FindStringSubmatch(envValue); len(matches) > 1 {
				schemaArgName := matches[1]
				if schemaArg, exists := schemaArgs[schemaArgName]; exists {
					rtVar.Description = schemaArg.Description
					rtVar.Required = schemaArg.Required
				}
			}

			// Also check if the actual env name matches a schema arg name
			if schemaArg, exists := schemaArgs[envName]; exists {
				rtVar.Description = schemaArg.Description
				rtVar.Required = schemaArg.Required
			}

			runtimeVars[envName] = rtVar
		}

		// Extract actual command line arguments
		for _, arg := range installation.Args {
			if strings.HasPrefix(arg, "--") {
				actualArgName := extractActualCommandLineFlag(arg)
				if actualArgName != "" {
					if rtv, exists := runtimeVars[actualArgName]; exists {
						rtv.Runtimes[rt] = struct{}{}
						continue
					}

					// Create new runtime variable
					rtVar := RuntimeVariable{
						Name:         actualArgName,
						VariableType: packages.VariableTypeArg,
						Runtimes:     map[runtime.Runtime]struct{}{rt: {}},
					}

					// Try to find description from schema arguments by looking for placeholders
					if matches := packages.EnvVarPlaceholderRegex.FindStringSubmatch(arg); len(matches) > 1 {
						schemaArgName := matches[1]
						if schemaArg, exists := schemaArgs[schemaArgName]; exists {
							rtVar.Description = schemaArg.Description
							rtVar.Required = schemaArg.Required
						}
					}

					runtimeVars[actualArgName] = rtVar
				}
			}
		}
	}

	return runtimeVars
}
