package plugin

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// pluginHandler is a function that executes a plugin and returns a response.
type pluginHandler func(ctx context.Context, instance *Instance) (*HTTPResponse, error)

// pluginResult holds the result of a plugin execution.
type pluginResult struct {
	instance *Instance
	response *HTTPResponse
	err      error
}

// pipeline orchestrates plugin execution across categories.
// It maintains category ordering, handles serial and parallel execution,
// and enforces plugin requirements during request/response processing.
type pipeline struct {
	logger  hclog.Logger
	mu      sync.RWMutex
	plugins map[config.Category][]*Instance
}

// newPipeline creates a new plugin pipeline.
func newPipeline(logger hclog.Logger) *pipeline {
	return &pipeline{
		logger:  logger.Named("plugin-pipeline"),
		plugins: make(map[config.Category][]*Instance),
	}
}

// HandleRequest processes a request through all plugin categories in order.
// Executes plugins serially or in parallel based on category properties.
// Returns early if a required plugin fails or if HTTPResponse.Continue is false.
// When every plugin passes, ModifiedRequest on the returned response holds the transformed request,
// and is nil when no plugin returned a modification.
func (p *pipeline) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	currentReq := req

	// Ensure we process plugins 'per category' and that category order is adhered to.
	for _, category := range orderedCategories {
		instances := p.plugins[category]
		if len(instances) == 0 {
			continue // Skip categories with no plugins registered.
		}

		// Properties will determine things like execution mode and
		// whether plugins in this category can modify a request.
		props, err := PropertiesForCategory(category)
		if err != nil {
			return nil, fmt.Errorf("getting properties for category %s: %w", category, err)
		}

		// Filter plugins that support request flow.
		requestPlugins, err := p.filterByFlow(ctx, instances, config.FlowRequest)
		if err != nil {
			return nil, fmt.Errorf("filtering plugins for category %s: %w", category, err)
		}
		if len(requestPlugins) == 0 {
			continue // this plugin doesn't support the current (request) flow.
		}

		mode := "serial"
		if props.Parallel {
			mode = "parallel"
		}

		p.logger.Debug("executing category",
			"category", category,
			"mode", mode,
			"plugin_count", len(requestPlugins),
		)

		// Execute using strategy pattern.
		// A modification is handed to the plugins that follow, so a category transforms the request in order.
		handler := func(ctx context.Context, inst *Instance) (*HTTPResponse, error) {
			pluginResp, err := inst.HandleRequest(ctx, currentReq)
			if err != nil {
				return nil, err
			}

			if props.CanModify && pluginResp != nil {
				currentReq = mergeRequestModification(currentReq, pluginResp.ModifiedRequest)
			}

			return pluginResp, nil
		}

		resp, err := p.execute(ctx, requestPlugins, handler, props)
		if err != nil {
			return nil, fmt.Errorf("category %s execution failed: %w", category, err)
		}

		// If a plugin returned a response with Continue=false, stop the pipeline.
		if resp != nil && !resp.Continue {
			p.logger.Info("pipeline stopped by plugin",
				"category", category,
				"status_code", resp.StatusCode,
			)
			return resp, nil
		}
	}

	// All plugins passed, continue to the upstream server.
	result := &HTTPResponse{Continue: true}
	if currentReq != req {
		result.ModifiedRequest = currentReq
	}

	return result, nil
}

// HandleResponse processes a response through all plugin categories in order.
// Executes plugins serially or in parallel based on category properties.
// Returns early if a required plugin fails or if Continue is false.
// The returned response carries any transformation a content plugin made, and otherwise is the response given.
func (p *pipeline) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	currentResp := resp

	for _, category := range orderedCategories {
		instances := p.plugins[category]
		if len(instances) == 0 {
			continue
		}

		props, err := PropertiesForCategory(category)
		if err != nil {
			return nil, fmt.Errorf("getting properties for category %s: %w", category, err)
		}

		// Filter plugins that support response flow.
		responsePlugins, err := p.filterByFlow(ctx, instances, config.FlowResponse)
		if err != nil {
			return nil, fmt.Errorf("filtering plugins for category %s: %w", category, err)
		}

		if len(responsePlugins) == 0 {
			continue
		}

		mode := "serial"
		if props.Parallel {
			mode = "parallel"
		}

		p.logger.Debug("executing category",
			"category", category,
			"mode", mode,
			"plugin_count", len(responsePlugins),
		)

		// Execute using strategy pattern.
		// A modification is handed to the plugins that follow, so a category transforms the response in order.
		handler := func(ctx context.Context, inst *Instance) (*HTTPResponse, error) {
			pluginResp, err := inst.HandleResponse(ctx, currentResp)
			if err != nil {
				return nil, err
			}

			if props.CanModify {
				currentResp = mergeResponseModification(currentResp, pluginResp)
			}

			return pluginResp, nil
		}

		result, err := p.execute(ctx, responsePlugins, handler, props)
		if err != nil {
			return nil, fmt.Errorf("category %s execution failed: %w", category, err)
		}

		// If a plugin returned Continue=false, return this response.
		if result != nil && !result.Continue {
			p.logger.Info("pipeline stopped by plugin",
				"category", category,
				"status_code", result.StatusCode,
			)
			return result, nil
		}
	}

	return currentResp, nil
}

// Register can be used to register a running plugin instance with a category.
// Returns an error if the category is unknown.
func (p *pipeline) Register(category config.Category, instance *Instance) error {
	if instance == nil {
		return fmt.Errorf("plugin instance cannot be nil")
	}

	if _, err := PropertiesForCategory(category); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.plugins[category] = append(p.plugins[category], instance)
	p.logger.Debug("plugin registered",
		"category", category,
		"name", instance.Name(),
		"required", instance.Required(),
	)

	return nil
}

// Shutdown gracefully stops all registered plugins.
func (p *pipeline) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error

	for category, instances := range p.plugins {
		for _, instance := range instances {
			p.logger.Debug("stopping plugin", "category", category, "name", instance.Name())
			if err := instance.Stop(ctx); err != nil {
				tmpErr := fmt.Errorf("stopping plugin %s in category %s: %w", instance.Name(), category, err)
				errors = append(errors, tmpErr)
			}
		}
	}

	p.plugins = make(map[config.Category][]*Instance)

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// execute runs plugins using the appropriate strategy (serial or parallel).
// This implements the Strategy pattern to avoid code duplication.
//
// NOTE: The returned response is the response of a plugin that stopped the pipeline, or an empty
// pass-through when every plugin passed, so a caller must not treat it as the current request or response state.
// A caller that tracks that state must fold each plugin's result in from its own handler.
// Only a serial category may do so, which PropertiesForCategory enforces.
func (p *pipeline) execute(
	ctx context.Context,
	plugins []*Instance,
	handler pluginHandler,
	props CategoryProperties,
) (*HTTPResponse, error) {
	if props.Parallel {
		return p.executeParallel(ctx, plugins, handler, props.IgnoreOptionalRejection)
	}

	return p.executeSerial(ctx, plugins, handler, props.IgnoreOptionalRejection)
}

// executeParallel runs plugins concurrently and aggregates results.
func (p *pipeline) executeParallel(
	ctx context.Context,
	plugins []*Instance,
	handler pluginHandler,
	ignoreOptionalRejection bool,
) (*HTTPResponse, error) {
	var wg sync.WaitGroup
	results := make(chan *pluginResult, len(plugins))

	for _, plugin := range plugins {
		wg.Add(1)
		go func(inst *Instance) {
			defer wg.Done()

			resp, err := handler(ctx, inst)
			results <- &pluginResult{
				instance: inst,
				response: resp,
				err:      err,
			}
		}(plugin)
	}

	wg.Wait()
	close(results)

	// Collect and analyze results.
	for result := range results {
		shouldStop, stopResp, handledErr := p.handlePluginResult(
			result.err,
			result.response,
			result.instance,
			ignoreOptionalRejection,
		)

		if handledErr != nil {
			return nil, handledErr
		}

		if shouldStop {
			return stopResp, nil
		}
	}

	return &HTTPResponse{Continue: true}, nil
}

// executeSerial runs plugins sequentially.
func (p *pipeline) executeSerial(
	ctx context.Context,
	plugins []*Instance,
	handler pluginHandler,
	ignoreOptionalRejection bool,
) (*HTTPResponse, error) {
	for _, instance := range plugins {
		p.logger.Debug("executing plugin", "name", instance.Name())

		resp, err := handler(ctx, instance)

		// Handle plugin result using common logic.
		shouldStop, stopResp, handledErr := p.handlePluginResult(
			err,
			resp,
			instance,
			ignoreOptionalRejection,
		)

		if handledErr != nil {
			return nil, handledErr
		}

		if shouldStop {
			return stopResp, nil
		}
	}

	return &HTTPResponse{Continue: true}, nil
}

// handlePluginResult processes plugin errors and rejections.
//
// Used by serial and parallel execution to consolidate logic.
//
// The following predicates are adhered to:
//   - An error for a required plugin should stop pipeline processing.
//   - A response for a required plugin which indicates that the pipeline should not continue, should stop processing.
//   - A response for an optional plugin which indicates that the pipeline should not continue, when the category does
//     not ignore optional plugin rejections, should stop processing.
//
// Other scenarios may produce a warning log message.
//
// Returns boolean indicating whether pipeline processing should be stopped, the response where available and required,
// and the error.
func (p *pipeline) handlePluginResult(
	err error,
	resp *HTTPResponse,
	inst *Instance,
	ignoreOptionalRejection bool,
) (bool, *HTTPResponse, error) {
	if err == nil && (resp == nil || resp.Continue) {
		return false, resp, nil
	}

	if err != nil {
		if inst.Required() {
			return true, nil, fmt.Errorf("%w: %s: %w", ErrRequiredPluginFailed, inst.Name(), err)
		}

		p.logger.Warn("optional plugin failed", "name", inst.Name(), "error", err)
		return false, resp, nil
	}

	if inst.Required() {
		return true, resp, nil
	}

	if ignoreOptionalRejection {
		p.logger.Warn("optional plugin rejected - but category ignores it", "name", inst.Name())
		return false, resp, nil
	}

	// Optional plugin didn't want to continue.
	return true, resp, nil
}

// filterByFlow returns plugins that both support the specified flow, and are configured to execute on that flow.
func (p *pipeline) filterByFlow(ctx context.Context, instances []*Instance, flow config.Flow) ([]*Instance, error) {
	var filtered []*Instance

	for _, instance := range instances {
		if !instance.IsFlowAllowed(flow) {
			continue // This plugin has not been configured for the specified flow.
		}

		canHandle, err := instance.IsFlowSupported(ctx, flow)
		if err != nil {
			tmpErr := fmt.Errorf(
				"unable to determine flow ('%s') support for plugin '%s': %w",
				flow,
				instance.Name(),
				err,
			)
			return nil, tmpErr
		}

		if canHandle {
			filtered = append(filtered, instance)
		}
	}

	return filtered, nil
}

// mergeHeaders returns a copy of base with every entry in overlay applied over it.
// It skips an entry in overlay that names a reserved header, and stores each name in canonical form so
// two spellings of one header cannot both survive.
func mergeHeaders(base map[string]string, overlay map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overlay))
	maps.Copy(merged, base)

	for name, value := range overlay {
		if isReservedHeader(name) {
			continue
		}

		merged[http.CanonicalHeaderKey(name)] = value
	}

	return merged
}

// mergeRequestModification returns req with the headers and body that mod sets applied to a copy of it.
// It returns req itself when mod is nil or sets neither.
//
// NOTE: The method and the path in mod are ignored, so every plugin and the handler see the ones the
// client sent. A plugin that replaced them would otherwise give the audit trail a request mcpd never served.
// A zero value in mod leaves that field unchanged, so a plugin cannot clear a header or a body.
func mergeRequestModification(req *HTTPRequest, mod *HTTPRequest) *HTTPRequest {
	if mod == nil {
		return req
	}

	if len(mod.Headers) == 0 && len(mod.Body) == 0 {
		return req
	}

	merged := *req

	if len(mod.Headers) != 0 {
		merged.Headers = mergeHeaders(req.Headers, mod.Headers)
	}

	if len(mod.Body) != 0 {
		merged.Body = mod.Body
	}

	return &merged
}

// mergeResponseModification returns resp with the headers and body that mod sets applied to a copy of it.
// It returns resp itself when mod is nil, when mod stops the pipeline, or when mod sets neither.
//
// NOTE: The status code in mod is ignored, because a plugin that continues the pipeline is adjusting a
// response rather than authoring one, and a redaction that forgets to echo the status would otherwise
// report an upstream failure as success.
// A plugin reports its own status code by stopping the pipeline with Continue=false.
// A zero value in mod leaves that field unchanged, so a plugin cannot clear a header or a body.
// An empty body cannot mean anything else, because proto3 omits an empty bytes field from the wire, so a
// plugin that sent one would be indistinguishable from a plugin that sent none.
func mergeResponseModification(resp *HTTPResponse, mod *HTTPResponse) *HTTPResponse {
	if mod == nil || !mod.Continue {
		return resp
	}

	if len(mod.Headers) == 0 && len(mod.Body) == 0 {
		return resp
	}

	merged := *resp

	if len(mod.Headers) != 0 {
		merged.Headers = mergeHeaders(resp.Headers, mod.Headers)
	}

	if len(mod.Body) != 0 {
		merged.Body = mod.Body
	}

	return &merged
}
