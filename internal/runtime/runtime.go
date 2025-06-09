package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/context"
)

// RuntimeServer composes static config with runtime overrides.
type RuntimeServer struct {
	config.ServerEntry // import from internal/config
	context.ServerExecutionContext
}

// Runtime returns the runtime (e.g. python, node) portion of the package string.
func (s *RuntimeServer) Runtime() string {
	parts := strings.Split(s.Package, "::")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func AggregateConfigs(
	cfg config.Config,
	executionContextCfg context.ExecutionContextConfig,
) ([]RuntimeServer, error) {
	var runtimeCfg []RuntimeServer

	for _, s := range cfg.Servers {
		runtimeServer := RuntimeServer{
			ServerEntry: config.ServerEntry{
				Name:    s.Name,
				Package: s.Package,
				Tools:   s.Tools,
			},
		}

		// Update with execution context if we have any for this server.
		if executionCtx, ok := executionContextCfg.Servers[s.Name]; ok {
			runtimeServer.ServerExecutionContext = context.ServerExecutionContext{
				Args: executionCtx.Args,
				Env:  executionCtx.Env,
			}
		}

		runtimeCfg = append(runtimeCfg, runtimeServer)
	}

	return runtimeCfg, nil
}

func (s *RuntimeServer) Environ() []string {
	baseEnvs := os.Environ()
	overrideEnvs := make([]string, 0, len(s.Env))
	for k, v := range s.Env {
		overrideEnvs = append(overrideEnvs, fmt.Sprintf("%s=%s", k, v))
	}
	return mergeEnvs(baseEnvs, overrideEnvs)
}

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
