package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
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
func (p *pipeline) HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	currentReq := req

	// Ensure we process plugins 'per category' and that category order is adhered to.
	for _, category := range OrderedCategories() {
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

		var resp *HTTPResponse

		// Execute using strategy pattern.
		handler := func(ctx context.Context, inst *Instance) (*HTTPResponse, error) {
			return inst.HandleRequest(ctx, currentReq)
		}

		resp, err = p.execute(ctx, requestPlugins, handler, props)
		if err != nil {
			return nil, fmt.Errorf("category %s execution failed: %w", category, err)
		}

		// If plugin modified the request and modification is allowed, use the modified version.
		if props.CanModify && resp != nil && resp.ModifiedRequest != nil {
			currentReq = resp.ModifiedRequest
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
	return &HTTPResponse{Continue: true}, nil
}

// HandleResponse processes a response through all plugin categories in order.
// Executes plugins serially or in parallel based on category properties.
// Returns early if a required plugin fails or if Continue is false.
func (p *pipeline) HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	currentResp := resp

	for _, category := range OrderedCategories() {
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
		handler := func(ctx context.Context, inst *Instance) (*HTTPResponse, error) {
			return inst.HandleResponse(ctx, currentResp)
		}

		result, err := p.execute(ctx, responsePlugins, handler, props)
		if err != nil {
			return nil, fmt.Errorf("category %s execution failed: %w", category, err)
		}

		// Update current response if modified.
		if props.CanModify && result != nil {
			currentResp = result
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
// Returns (shouldStop, response, error).
// This centralizes the logic for handling required vs optional plugins and rejection policies.
func (p *pipeline) handlePluginResult(
	err error,
	response *HTTPResponse,
	instance *Instance,
	ignoreOptionalRejection bool,
) (bool, *HTTPResponse, error) {
	// Handle plugin errors.
	if err != nil {
		if instance.Required() {
			tmpErr := fmt.Errorf("%w: %s: %w", ErrRequiredPluginFailed, instance.Name(), err)
			return false, nil, tmpErr
		}

		// Optional plugin error, log and continue.
		p.logger.Warn("optional plugin failed",
			"name", instance.Name(),
			"error", err,
		)
		return false, nil, nil
	}

	// Handle plugin rejection (Continue=false).
	if response != nil && !response.Continue {
		// Required plugins always cause rejection.
		// Optional plugins cause rejection only if ignoreOptionalRejection is false.
		if instance.Required() || !ignoreOptionalRejection {
			return true, response, nil
		}

		// Optional plugin in non-rejecting category, log and continue.
		p.logger.Warn("optional plugin rejected but category ignores optional rejection",
			"name", instance.Name(),
		)
		return false, nil, nil
	}

	return false, nil, nil
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
