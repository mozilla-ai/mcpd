package plugin

import (
	"context"
	"fmt"
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
// ModifiedRequest on the returned response holds the transformed request, and is nil when no plugin changed it.
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

			if props.CanModify && pluginResp != nil && pluginResp.ModifiedRequest != nil {
				currentReq = pluginResp.ModifiedRequest
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

			if props.CanModify && isResponseModification(pluginResp) {
				currentResp = pluginResp
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
// NOTE: The returned response belongs to a plugin that stopped the pipeline.
// It carries no content when every plugin passed, so a caller must not treat it as the current state.
// A caller that tracks request or response state must fold each plugin's result in from its own handler.
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

// isResponseModification reports whether resp carries a replacement response rather than a pass-through.
//
// NOTE: A plugin that continues without setting a status code, headers or a body makes no change.
// Zero is not a valid HTTP status, so an echo of Continue alone cannot blank the response.
// A plugin that wants to return an empty response must reject it with Continue=false.
func isResponseModification(resp *HTTPResponse) bool {
	if resp == nil || !resp.Continue {
		return false
	}

	return resp.StatusCode != 0 || len(resp.Headers) != 0 || len(resp.Body) != 0
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
