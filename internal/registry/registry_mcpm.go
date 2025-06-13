package registry

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

	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/mcpm"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"
)

const registryNameMCPM = "mcpm"

// MCPMRegistry implements the PackageRegistry interface for the MCPM server JSON format.
type MCPMRegistry struct {
	servers           mcpm.ServerMap
	logger            hclog.Logger
	supportedRuntimes map[types.Runtime]struct{}
	filterOptions     []FilterOption
}

// NewMCPMRegistry creates a new MCPMRegistry instance by fetching its data from the provided URL.
func NewMCPMRegistry(logger hclog.Logger, url string, opts ...Option) (*MCPMRegistry, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty MCPM registry URL is invalid")
	}

	opt, err := getOpts(opts...)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCPM registry URL '%s': %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status from MCPM registry '%s': %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCPM registry response body from '%s': %w", url, err)
	}

	var servers mcpm.ServerMap
	if err := json.Unmarshal(body, &servers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCPM registry JSON from '%s': %w", url, err)
	}

	l := logger.Named(registryNameMCPM)

	// Configure filter options with 'version' unsupported and logging.
	filterOpts := []FilterOption{
		WithUnsupportedFilters("version"),
		WithLogFunc(func(key, val string) {
			l.Warn("Unsupported filter for search operation", "filter", key, "value", val)
		}),
	}

	return &MCPMRegistry{
		servers:           servers,
		logger:            l,
		supportedRuntimes: opt.supportedRuntimes,
		filterOptions:     filterOpts,
	}, nil
}

func (r *MCPMRegistry) ID() string {
	return registryNameMCPM
}

// Search implements the PackageSearcher interface for MCPMRegistry.
func (r *MCPMRegistry) Search(name string, filters map[string]string) ([]types.PackageResult, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}

	// TODO: Sanity check to make sure we have name in the filters.
	if filters == nil {
		filters = make(map[string]string)
	}
	if _, ok := filters["name"]; !ok {
		filters["name"] = name
	}

	var results []types.PackageResult

	for id := range r.servers {
		result, transformed := r.buildPackageResult(id)
		if !transformed {
			continue
		}

		matches, err := MatchWithFilters(result, filters, r.filterOptions...)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}

		results = append(results, result)
	}
	return results, nil
}

// Get implements the PackageGetter interface for MCPMRegistry.
// It retrieves a specific package by its unique ID.
// For MCPMRegistry, the 'version' parameter is not supported for filtering,
// and it will always return the definition for the given 'id' if found.
func (r *MCPMRegistry) Get(id string, opts ...types.GetterOption) (types.PackageResult, error) {
	options, err := types.GetGetterOpts(opts...)
	if err != nil {
		return types.PackageResult{}, err
	}

	if options.Version != "" {
		r.logger.Warn(
			"'version' not supported on get operation, returning latest known definition",
			"id", id,
			"version", options.Version)

		// Clear 'version' for MCPM.
		options.Version = ""
	}

	r.logger.Debug("Getting package", "id", id, "version", options.Version)

	result, transformed := r.buildPackageResult(id)
	if !transformed {
		return types.PackageResult{}, fmt.Errorf("failed to build package result for '%s'", id)
	}

	matches, err := MatchWithGetterOptions(result, options, r.filterOptions...)
	if err != nil {
		return types.PackageResult{}, err
	}
	if !matches {
		return types.PackageResult{}, fmt.Errorf("package with ID '%s' does not match requested filters", id)
	}

	return result, nil
}

// buildPackageResult attempts to convert the ServerDetails associated with the specified ID,
// into a PackageResult.
// Returns the transformed result, and a flag to indicate if the transformation was successful.
// If the server cannot be transformed due to unsupported or malformed runtime installations, false is returned.
func (r *MCPMRegistry) buildPackageResult(id string) (types.PackageResult, bool) {
	// Sanity check to ensure things work when a random ID gets supplied.
	sd, foundServer := r.servers[id]
	if !foundServer {
		r.logger.Warn(
			"transformation to server details failed, server not found",
			"id", id,
		)
		return types.PackageResult{}, false
	}

	runtimesAndPackages, err := r.supportedRuntimePackageNames(sd)
	if err != nil || len(runtimesAndPackages) == 0 {
		r.logger.Debug(
			"no supported runtime packages found in registry",
			"id", id,
			"error", err,
		)
		return types.PackageResult{}, false
	}

	var runtimes []string
	for rt := range runtimesAndPackages {
		runtimes = append(runtimes, string(rt))
	}
	slices.Sort(runtimes)
	pkgName := runtimesAndPackages[types.Runtime(runtimes[0])]

	tools := make([]string, 0, len(sd.Tools))
	for _, tool := range sd.Tools {
		tools = append(tools, tool.Name)
	}

	configurableEnvVars := make([]string, 0)
	seenEnvVars := make(map[string]struct{})
	// TODO: Refactor...
	for _, install := range sd.Installations {
		for envVar := range install.Env {
			if _, seen := seenEnvVars[envVar]; !seen {
				configurableEnvVars = append(configurableEnvVars, envVar)
				seenEnvVars[envVar] = struct{}{}
			}
		}
		for _, arg := range install.Args {
			matches := types.EnvVarPlaceholderRegex.FindAllStringSubmatch(arg, -1)
			for _, match := range matches {
				if len(match) > 1 {
					envVar := match[1]
					if _, seen := seenEnvVars[envVar]; !seen {
						configurableEnvVars = append(configurableEnvVars, envVar)
						seenEnvVars[envVar] = struct{}{}
					}
				}
			}
		}
	}

	return types.PackageResult{
		ID:                  id,
		Source:              registryNameMCPM,
		Name:                pkgName,
		DisplayName:         sd.DisplayName,
		Description:         sd.Description,
		License:             sd.License,
		Tools:               tools,
		Runtimes:            runtimes,
		InstallationDetails: convertInstallations(sd.Installations),
		Arguments:           convertArguments(sd.Arguments),
		ConfigurableEnvVars: configurableEnvVars,
	}, true
}

func convertInstallations(src map[string]mcpm.Installation) map[string]types.Installation {
	if src == nil {
		return nil
	}

	details := make(map[string]types.Installation, len(src))
	for _, install := range src {
		details[install.Command] = types.Installation{
			Args:        slices.Clone(install.Args),
			Package:     install.Package,
			Env:         maps.Clone(install.Env),
			Description: install.Description,
			Recommended: install.Recommended,
		}
	}
	return details
}

func convertArguments(src map[string]mcpm.Argument) map[string]types.ArgumentMetadata {
	if src == nil {
		return nil
	}

	details := make(map[string]types.ArgumentMetadata, len(src))
	for key, value := range src {
		details[key] = types.ArgumentMetadata{
			Description: value.Description,
			Required:    value.Required,
			Example:     value.Example,
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
func (r *MCPMRegistry) supportedRuntimePackageNames(server mcpm.ServerDetails) (map[types.Runtime]string, error) {
	result := make(map[types.Runtime]string)

	for _, inst := range server.Installations {
		// MCPM's registry is a bit inconsistent around npm/npx.
		// Sometimes an installation key is npm, sometimes npx.
		// Sometimes, the key is npx, but the type is npm, etc. it seems the only consistent thing to
		// index on is the actual 'command' which shows npx consistently.
		runtime := types.Runtime(inst.Command)
		if _, ok := r.supportedRuntimes[runtime]; !ok {
			continue
		}

		pkg, err := extractPlainPackage(runtime, inst.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to extract package for runtime %q: %w", runtime, err)
		}
		result[runtime] = pkg
	}

	return result, nil
}

// extractPlainPackage scans a slice of command-line arguments and returns the first valid
// package identifier. It skips flags (e.g., "-y"), interpolated env vars ("${FOO}"),
// URLs, git references, and script files (".py").
//
// This function enforces a strict format, ensuring only plain package names are accepted.
// Returns an error if no suitable package name is found.
func extractPlainPackage(runtime types.Runtime, args []string) (string, error) {
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-"), strings.HasPrefix(arg, "${"):
			continue
		case strings.HasPrefix(arg, "git+"), strings.HasSuffix(arg, ".py"):
			continue
		case runtime == types.RuntimeUvx && strings.HasPrefix(arg, "https://"),
			runtime == types.RuntimeUvx && strings.HasPrefix(arg, "http://"):
			continue

		default:
			return arg, nil
		}
	}
	return "", errors.New("no valid plain package name found in args")
}
