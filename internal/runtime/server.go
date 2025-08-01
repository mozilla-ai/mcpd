package runtime

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

type Servers []Server

// Server composes static config with runtime overrides.
type Server struct {
	config.ServerEntry
	context.ServerExecutionContext
}

func (s *Server) Name() string {
	return s.ServerEntry.Name
}

// Runtime returns the runtime (e.g. python, node) portion of the package string.
func (s *Server) Runtime() string {
	parts := strings.Split(s.Package, "::")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (s *Server) ResolvedArgs() []string {
	return expandEnvSlice(s.Args)
}

// Environ returns the server's effective environment with overrides applied,
// irrelevant variables stripped, and any ${VAR} references expanded.
func (s *Server) Environ() []string {
	baseEnvs := os.Environ()

	overrideEnvs := make([]string, 0, len(s.Env))
	for k, v := range s.Env {
		overrideEnvs = append(overrideEnvs, fmt.Sprintf("%s=%s", k, v))
	}

	// Merge the server's environment variables on top of the existing environment.
	mergedEnvs := mergeEnvs(baseEnvs, overrideEnvs)

	// Filter the environment to remove vars for other MCP servers or mcpd itself.
	filteredEnvs := filterEnv(mergedEnvs, s.Name())

	// Expand any variables that use templating ${}.
	expandedEnvs := expandEnvSlice(filteredEnvs)

	return expandedEnvs
}

// validateRequiredEnvVars checks that all required environment variables are set and non-empty.
func (s *Server) validateRequiredEnvVars() error {
	var errs error

	for _, key := range s.RequiredEnvVars {
		// TODO: Verify if we need to check the value (and what is valid for a value).
		if v, ok := s.Env[key]; !ok || v == "" {
			errs = errors.Join(errs, fmt.Errorf("required env var %s not set or empty", key))
		}
	}

	return errs
}

// validateRequiredValueArgs verifies all required arguments that must have values are present with values.
func (s *Server) validateRequiredValueArgs() error {
	var errs error

	// Validate required value args (must have an associated value)
	for _, key := range s.RequiredValueArgs {
		found := false
		// Using counter to allow us to look ahead to check for values that are supplied separately.
		// e.g. ["--foo=bar"] vs. ["--foo", "bar"] which are both valid.
		for i := 0; i < len(s.Args); i++ {
			arg := s.Args[i]

			// Validate --foo=bar
			if strings.HasPrefix(arg, key+"=") {
				found = true
				break
			}

			// Validate --foo bar
			// NOTE: Doesn't support validating short flags being the next value.
			if arg == key && i+1 < len(s.Args) && !strings.HasPrefix(s.Args[i+1], "--") {
				found = true
				break
			}
		}
		if !found {
			errs = errors.Join(errs, fmt.Errorf("required argument %s with value missing", key))
		}
	}

	return errs
}

// validateRequiredBoolArgs ensures all required boolean flags are present in the arguments.
func (s *Server) validateRequiredBoolArgs() error {
	var errs error

	// Validate required bool args (must be present, no value needed)
	for _, key := range s.RequiredBoolArgs {
		found := false

		for _, arg := range s.Args {
			if arg == key {
				found = true
				break
			}
		}
		if !found {
			errs = errors.Join(errs, fmt.Errorf("required boolean flag %s missing", key))
		}
	}

	return errs
}

// Validate can be used to ensure that any required env vars and args declared in config, are present in the runtime config.
func (s *Server) Validate() error {
	var errs error

	if envErrs := s.validateRequiredEnvVars(); envErrs != nil {
		errs = errors.Join(errs, envErrs)
	}

	if valueArgsErrs := s.validateRequiredValueArgs(); valueArgsErrs != nil {
		errs = errors.Join(errs, valueArgsErrs)
	}

	if boolArgsErrs := s.validateRequiredBoolArgs(); boolArgsErrs != nil {
		errs = errors.Join(errs, boolArgsErrs)
	}

	return errs
}

func (s *Server) exportArgs(appName string, recordContractFunc func(k, v string)) []string {
	args := make([]string, 0, len(s.RequiredArguments()))

	// Add all required bool args (flags).
	args = append(args, s.RequiredBoolArgs...)

	// Transform and add the required args that need values.
	for _, v := range s.RequiredValueArgs {
		t := transformValueArg(appName, s.Name(), v)
		args = append(args, t.FormattedArg)                 // Track for portable execution context export.
		recordContractFunc(t.EnvVarName, t.EnvVarReference) // Track for contract export (e.g. '.env' file).
	}

	// Capture the required args we've now seen.
	seen := make(map[string]struct{}, len(args))
	for _, k := range args {
		k = extractArgNameWithPrefix(k)
		seen[k] = struct{}{}
	}

	// Include any additional args that were set in the runtime config.
	args = append(args, s.exportRuntimeArgs(appName, seen, recordContractFunc)...)

	return args
}

// envVarsToContract converts environment variable mappings to contract format.
// Takes a map of env var name → placeholder reference (e.g., "API_KEY" → "${MCPD__SERVER__API_KEY}")
// Returns a map of placeholder name → placeholder reference (e.g., "MCPD__SERVER__API_KEY" → "${MCPD__SERVER__API_KEY}")
func envVarsToContract(envs map[string]string) map[string]string {
	contract := make(map[string]string, len(envs))

	for _, placeholderRef := range envs {
		// Extract placeholder name from reference: "${MCPD__SERVER__VAR}" → "MCPD__SERVER__VAR"
		if strings.HasPrefix(placeholderRef, "${") && strings.HasSuffix(placeholderRef, "}") {
			placeholderName := strings.TrimSpace(placeholderRef[2 : len(placeholderRef)-1])
			// Skip empty placeholder names
			if placeholderName != "" {
				contract[placeholderName] = placeholderRef
			}
		}
	}

	return contract
}

// exportEnvVars generates environment variable placeholders for both required and runtime env vars.
// Returns a map where keys are env var names (e.g., "API_KEY") and values are placeholder references (e.g., "${MCPD__SERVER__API_KEY}").
func (s *Server) exportEnvVars(appName string) map[string]string {
	envs := map[string]string{}

	// Any required env names from config should be included.
	for _, k := range s.RequiredEnvVars {
		envVarName := buildEnvVarName(appName, s.Name(), k)
		envs[k] = fmt.Sprintf("${%s}", envVarName)
	}

	// Update with any vars that were set in runtime execution context config.
	for k := range s.Env {
		// Skip if we've already captured this variable via required env vars.
		if _, ok := envs[k]; !ok {
			envVarName := buildEnvVarName(appName, s.Name(), k)
			envs[k] = fmt.Sprintf("${%s}", envVarName)
		}
	}

	return envs
}

func (s *Server) exportRuntimeArgs(
	appName string,
	seen map[string]struct{},
	recordContractFunc func(k, v string),
) []string {
	var args []string

	for i := 0; i < len(s.Args); i++ {
		rawArg := s.Args[i]

		// Sanity check for arg.
		if !strings.HasPrefix(rawArg, "--") {
			continue // value
		}

		arg := extractArgNameWithPrefix(rawArg)
		if _, ok := seen[arg]; ok {
			continue // Already handled.
		}

		// --arg=val case
		if strings.HasPrefix(rawArg, arg+"=") {
			t := transformValueArg(appName, s.Name(), rawArg)
			args = append(args, t.FormattedArg)
			recordContractFunc(t.EnvVarName, t.EnvVarReference)
			seen[arg] = struct{}{}
			continue
		}

		// --arg val case
		if rawArg == arg && i+1 < len(s.Args) && !strings.HasPrefix(s.Args[i+1], "--") {
			t := transformValueArg(appName, s.Name(), rawArg)
			args = append(args, t.FormattedArg)
			recordContractFunc(t.EnvVarName, t.EnvVarReference)
			seen[arg] = struct{}{}
			i++ // Skip the next item since it's an actual value.
			continue
		}

		// bool flag
		if rawArg == arg {
			args = append(args, arg)
			seen[arg] = struct{}{}
			continue
		}
	}

	return args
}

func (s *Servers) Export(path string) (map[string]string, error) {
	servers := *s
	if len(servers) == 0 {
		return nil, fmt.Errorf("export error, no servers defined in runtime config")
	}

	const appName = "mcpd" // TODO: Reference shared app name from somewhere without cyclic import issues.

	contract := make(map[string]string)
	pec := context.NewExecutionContextConfig(path)

	for _, srv := range servers {
		// Export env vars.
		envs := srv.exportEnvVars(appName)
		maps.Copy(contract, envVarsToContract(envs))

		// Export args.
		args := srv.exportArgs(appName, func(k, v string) {
			contract[k] = v
		})

		// Store the parsed and sanitized data in the new portable execution context.
		pec.Servers[srv.Name()] = context.ServerExecutionContext{
			Name: srv.Name(),
			Args: args,
			Env:  envs,
		}
	}

	// Save the fully formed portable execution context.
	err := pec.SaveConfig()
	if err != nil {
		return nil, fmt.Errorf("export error, failed to save portable execution config: %v", err)
	}

	return contract, nil
}

// AggregateConfigs merges static server config with any matching execution context overrides.
// Returns (unresolved) runtime configuration for all servers.
func AggregateConfigs(
	cfg config.Modifier,
	executionContextCfg context.Modifier,
) (Servers, error) {
	var runtimeCfg []Server

	for _, s := range cfg.ListServers() {
		runtimeServer := Server{
			ServerEntry: config.ServerEntry{
				Name:              s.Name,
				Package:           s.Package,
				Tools:             s.Tools,
				RequiredEnvVars:   s.RequiredEnvVars,
				RequiredValueArgs: s.RequiredValueArgs,
				RequiredBoolArgs:  s.RequiredBoolArgs,
			},
		}

		// Update with execution context if we have any for this server.
		if executionCtx, ok := executionContextCfg.Get(s.Name); ok {
			runtimeServer.ServerExecutionContext = context.ServerExecutionContext{
				Args: executionCtx.Args,
				Env:  executionCtx.Env,
			}
		}

		runtimeCfg = append(runtimeCfg, runtimeServer)
	}

	return runtimeCfg, nil
}

// mergeEnvs combines two environment slices, applying overrides where keys overlap.
// Later values take precedence, with overrideEnvs replacing entries from baseEnvs.
func mergeEnvs(baseEnvs, overrideEnvs []string) []string {
	envMap := make(map[string]string, len(baseEnvs))

	for _, e := range baseEnvs {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for _, e := range overrideEnvs {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// containsIllegalReference reports whether the value contains references to MCP servers other than
// the specified server or to application-level (mcpd) variables.
//
// The serverName parameter **must** be already normalized (uppercase with hyphens replaced by underscores).
// For example, "mcp-discord" should be passed as "MCP_DISCORD".
//
// Returns true if the value contains variable references in any of these patterns:
//   - ${MCPD__OTHER_SERVER__VAR} - reference to different server
//   - $(MCPD__OTHER_SERVER__VAR) - reference to different server
//   - $MCPD__OTHER_SERVER__VAR   - reference to different server
//   - ${MCPD_APP_VAR}            - reference to application variable (always illegal)
//   - $(MCPD_APP_VAR)            - reference to application variable (always illegal)
//   - $MCPD_APP_VAR              - reference to application variable (always illegal)
//
// Matching is case-insensitive and somewhat permissive to err on the side of security.
func containsIllegalReference(serverName string, value string) bool {
	const spacer = "__"

	appName := "MCPD" // TODO: Fix import cycle that occurs if we use strings.ToUpper(cmd.AppName())
	// appPrefix := appName + spacer

	// Define a regex that can identify use of other server's vars AND app-level vars inside expansions:
	// ${MCPD__SERVER_2__ANYTHING}  - other server vars
	// $(MCPD__SERVER_2__ANYTHING)  - other server vars
	// $MCPD__SERVER_2__ANYTHING    - other server vars
	// ${MCPD_APP_VAR}              - app-level vars
	// $(MCPD_APP_VAR)              - app-level vars
	// $MCPD_APP_VAR                - app-level vars
	valRefRe := regexp.MustCompile(
		`(?i)\$(?:\{|\()?` +
			regexp.QuoteMeta(appName) + "_{1,2}" + // "MCPD_" or "MCPD__"
			`(?:` +
			`([A-Z0-9_]+)` + spacer + `[^{}\s)\$]+|` + // Server pattern: MCPD__SERVER__VAR
			`([A-Z0-9_]+)` + // App pattern: MCPD_VAR
			`)` +
			`(?:\}|\))?`,
	)

	matches := valRefRe.FindAllStringSubmatch(value, -1)
	if matches == nil {
		return false
	}

	for _, match := range matches {
		if len(match) > 1 {
			sn := strings.TrimSpace(match[1])
			if sn != "" && sn != serverName { // Server var for different server
				return true
			}

			if match[2] != "" { // App-level var (always illegal)
				return true
			}
		}
	}

	return false
}

// filterEnv removes environment variables that appear to be intended for other MCP servers
// or for the mcpd application itself.
//
// Environment variable formats:
//   - Application vars: MCPD_{VAR_NAME}
//   - Server vars: MCPD__{SERVER_NAME}__[{ARG}__]{VAR_NAME} (created via 'mcpd config export')
//
// Filtering rules:
//  1. Variables with keys for other servers or app vars are removed
//  2. Variables with values referencing other servers or app vars are removed
//  3. Malformed variables (missing '=') are ignored
//  4. Matching is case-insensitive and permissive for security
//
// Examples of filtered content:
//   - Key: MCPD__OTHER_SERVER__CONFIG=value
//   - Value: CONFIG=${MCPD__OTHER_SERVER__HOST}
//   - Value: CONFIG=${MCPD_APP_SECRET}
//   - Value: partial${MCPD__OTHER_SERVER__TOKEN}reference
//   - Malformed: VAR_WITHOUT_EQUALS
//
// Returns a sorted slice of allowed environment variables in "KEY=VALUE" format.
// Returns an empty slice if env is nil.
func filterEnv(env []string, serverName string) []string {
	if len(env) == 0 {
		return []string{}
	}

	appName := "MCPD" // TODO: Fix import cycle that occurs if we use strings.ToUpper(cmd.AppName())
	srvName := strings.ReplaceAll(strings.ToUpper(serverName), "-", "_")

	// MCP server specific naming
	appPrefix := fmt.Sprintf("%s__", appName)                 // "MCPD__"
	serverPrefix := fmt.Sprintf("%s%s__", appPrefix, srvName) // "MCPD__TIME__"

	// mcpd specific naming (for checking after matching on MCP servers).
	reservedAppPrefix := appName + "_" // MCPD_

	var filtered []string

	for _, kv := range env {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue // Probably a malformed entry, ignore.
		}

		key, value := strings.ToUpper(kv[:idx]), strings.ToUpper(kv[idx+1:])

		// Specifically for another server (drop).
		if strings.HasPrefix(key, appPrefix) && !strings.HasPrefix(key, serverPrefix) {
			continue // Ignored
		}

		// We don't allow any MCP server to be given access to mcpd application level variables.
		if strings.HasPrefix(key, reservedAppPrefix) && !strings.HasPrefix(key, appPrefix) {
			continue // Ignored
		}

		// Value references a different MCP server variable (drop).
		if containsIllegalReference(srvName, value) {
			continue // Ignored
		}

		filtered = append(filtered, kv)
	}

	slices.Sort(filtered)
	return filtered
}

// expandEnvSlice returns a new []string with all ${VAR} references of value, expanded using the current environment.
func expandEnvSlice(input []string) []string {
	result := make([]string, len(input))

	for i, kv := range input {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			result[i] = kv
			continue
		}

		key := kv[:idx]
		val := os.ExpandEnv(kv[idx+1:])

		result[i] = fmt.Sprintf("%s=%s", key, val)
	}

	return result
}
