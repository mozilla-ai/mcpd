package config

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/files"
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
)

const (
	// CategoryAuthentication represents authentication plugins.
	CategoryAuthentication Category = "authentication"

	// CategoryAuthorization represents authorization plugins.
	CategoryAuthorization Category = "authorization"

	// CategoryRateLimiting represents rate limiting plugins.
	CategoryRateLimiting Category = "rate_limiting"

	// CategoryValidation represents validation plugins.
	CategoryValidation Category = "validation"

	// CategoryContent represents content transformation plugins.
	CategoryContent Category = "content"

	// CategoryObservability represents observability plugins.
	CategoryObservability Category = "observability"

	// CategoryAudit represents audit/compliance logging plugins.
	CategoryAudit Category = "audit"
)

const (
	// FlowRequest indicates the plugin executes during the request phase.
	FlowRequest Flow = "request"

	// FlowResponse indicates the plugin executes during the response phase.
	FlowResponse Flow = "response"
)

// orderedCategories defines the pipeline execution order.
// Categories should execute in this sequence for each request/response.
// NOTE: This variable should not be modified in other parts of the codebase.
var orderedCategories = Categories{
	CategoryObservability, // First, parallel, non-blocking.
	CategoryAuthentication,
	CategoryAuthorization,
	CategoryRateLimiting,
	CategoryValidation,
	CategoryContent,
	CategoryAudit, // Last.
}

// flows defines the set of valid flow types.
// NOTE: This variable should not be modified in other parts of the codebase.
var flows = map[Flow]struct{}{
	FlowRequest:  {},
	FlowResponse: {},
}

// PluginModifier defines operations for managing plugin configuration.
type PluginModifier interface {
	// Plugin retrieves a plugin by category and name.
	Plugin(category Category, name string) (PluginEntry, bool)

	// UpsertPlugin creates or updates a plugin entry.
	UpsertPlugin(category Category, entry PluginEntry) (context.UpsertResult, error)

	// DeletePlugin removes a plugin entry.
	DeletePlugin(category Category, name string) (context.UpsertResult, error)

	// ListPlugins returns all plugins in a category.
	ListPlugins(category Category) []PluginEntry
}

// Categories represents collection of Category types.
type Categories []Category

// Category represents a plugin category.
type Category string

// Flow represents the execution phase for a plugin.
type Flow string

// Flows returns the canonical set of allowed flows.
// Returns a clone to prevent modification of the internal map.
func Flows() map[Flow]struct{} {
	return maps.Clone(flows)
}

// IsValid returns true if the Flow is a recognized value.
func (f Flow) IsValid() bool {
	_, ok := flows[f]
	return ok
}

// ParseFlowsDistinct validates and reduces flow strings to a distinct set.
// Flow strings are normalized before validation.
// Invalid flows are silently ignored. Returns an empty map if no valid flows are found.
func ParseFlowsDistinct(flags []string) map[Flow]struct{} {
	valid := make(map[Flow]struct{}, len(flows))

	for _, s := range flags {
		f := Flow(filter.NormalizeString(s))
		if _, ok := flows[f]; ok {
			valid[f] = struct{}{}
		}
	}

	return valid
}

// PluginConfig represents the top-level plugin configuration.
//
// NOTE: if you add/remove fields you must review the associated validation implementation.
type PluginConfig struct {
	// Dir specifies the directory containing plugin binaries.
	Dir string `json:"dir,omitempty" toml:"dir,omitempty" yaml:"dir,omitempty"`

	// Authentication plugins execute first, validating identity.
	Authentication []PluginEntry `json:"authentication,omitempty" toml:"authentication,omitempty" yaml:"authentication,omitempty"`

	// Authorization plugins verify permissions after authentication.
	Authorization []PluginEntry `json:"authorization,omitempty" toml:"authorization,omitempty" yaml:"authorization,omitempty"`

	// RateLimiting plugins enforce request rate limits.
	RateLimiting []PluginEntry `json:"rateLimiting,omitempty" toml:"rate_limiting,omitempty" yaml:"rate_limiting,omitempty"`

	// Validation plugins check request/response structure and content.
	Validation []PluginEntry `json:"validation,omitempty" toml:"validation,omitempty" yaml:"validation,omitempty"`

	// Content plugins transform request/response payloads.
	Content []PluginEntry `json:"content,omitempty" toml:"content,omitempty" yaml:"content,omitempty"`

	// Observability plugins collect metrics and traces (non-blocking).
	Observability []PluginEntry `json:"observability,omitempty" toml:"observability,omitempty" yaml:"observability,omitempty"`

	// Audit plugins log compliance and security events (typically required).
	Audit []PluginEntry `json:"audit,omitempty" toml:"audit,omitempty" yaml:"audit,omitempty"`
}

// PluginEntry represents a single plugin configuration within a category.
type PluginEntry struct {
	// Name of the plugin binary in the plugins directory.
	Name string `json:"name" toml:"name" yaml:"name"`

	// CommitHash for validating plugin version against metadata.
	CommitHash *string `json:"commitHash,omitempty" toml:"commit_hash,omitempty" yaml:"commit_hash,omitempty"`

	// Required indicates if plugin failure should block the request.
	Required *bool `json:"required,omitempty" toml:"required,omitempty" yaml:"required,omitempty"`

	// Flows specifies when the plugin executes (request, response, or both).
	// Treated as a set - duplicates are rejected during validation.
	Flows []Flow `json:"flows" toml:"flows" yaml:"flows"`
}

// Equals compares two PluginEntry instances for equality.
func (e *PluginEntry) Equals(other *PluginEntry) bool {
	if other == nil {
		return false
	}

	// Compare Name.
	if e.Name != other.Name {
		return false
	}

	// Compare CommitHash.
	if (e.CommitHash == nil) != (other.CommitHash == nil) {
		return false
	}
	if e.CommitHash != nil && *e.CommitHash != *other.CommitHash {
		return false
	}

	// Compare Required.
	if (e.Required == nil) != (other.Required == nil) {
		return false
	}
	if e.Required != nil && *e.Required != *other.Required {
		return false
	}

	// Compare Flows...
	currentFlows := e.FlowsDistinct()
	otherFlows := other.FlowsDistinct()

	// Check maps are the same size.
	if len(currentFlows) != len(otherFlows) {
		return false
	}

	// Check they have the same entries.
	for k := range currentFlows {
		if _, ok := otherFlows[k]; !ok {
			return false
		}
	}

	return true
}

// FlowsDistinct converts the Flows slice to a set for efficient lookup.
func (e *PluginEntry) FlowsDistinct() map[Flow]struct{} {
	result := make(map[Flow]struct{}, len(e.Flows))
	for _, flow := range e.Flows {
		result[flow] = struct{}{}
	}
	return result
}

// HasFlow checks if the plugin is configured for the specified flow.
func (e *PluginEntry) HasFlow(flow Flow) bool {
	for _, f := range e.Flows {
		if f == flow {
			return true
		}
	}
	return false
}

func (c *Category) String() string {
	return string(*c)
}

// Validate validates a single PluginEntry.
func (e *PluginEntry) Validate() error {
	var validationErrors []error

	// Name is required.
	if strings.TrimSpace(e.Name) == "" {
		validationErrors = append(validationErrors, fmt.Errorf("plugin name is required"))
	}

	// Validate flows.
	if len(e.Flows) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("at least one flow is required"))
	} else {
		seen := make(map[Flow]struct{})
		for _, flow := range e.Flows {
			if !flow.IsValid() {
				allowedFlows := strings.Join(OrderedFlowNames(), ", ")
				err := fmt.Errorf("invalid flow '%s' (allowed: %s)", flow, allowedFlows)
				validationErrors = append(validationErrors, err)
			}

			// Check for duplicates.
			if _, exists := seen[flow]; exists {
				validationErrors = append(validationErrors, fmt.Errorf("duplicate flow: %s", flow))
			}
			seen[flow] = struct{}{}
		}
	}

	return errors.Join(validationErrors...)
}

// Validate implements Validator for PluginConfig.
// Validates all plugin entries across all categories.
func (p *PluginConfig) Validate() error {
	if p == nil {
		return nil
	}

	var validationErrors []error

	// Validate each category.
	categories := []struct {
		name    Category
		entries []PluginEntry
	}{
		{CategoryAuthentication, p.Authentication},
		{CategoryAuthorization, p.Authorization},
		{CategoryRateLimiting, p.RateLimiting},
		{CategoryValidation, p.Validation},
		{CategoryContent, p.Content},
		{CategoryObservability, p.Observability},
		{CategoryAudit, p.Audit},
	}

	for _, cat := range categories {
		for _, entry := range cat.entries {
			if err := entry.Validate(); err != nil {
				// Use plugin name if available, otherwise "unknown".
				name := "unknown"
				if strings.TrimSpace(entry.Name) != "" {
					name = entry.Name
				}
				validationErrors = append(
					validationErrors,
					fmt.Errorf("plugin '%s' in category '%s': %w", name, cat.name, err),
				)
			}
		}
	}

	// Validate directory and plugins if Dir is configured.
	if err := p.validatePluginDirectory(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return errors.Join(validationErrors...)
}

// validatePluginDirectory validates that the plugin directory exists and contains all configured plugins.
// Returns nil if Dir is empty (plugins disabled).
func (p *PluginConfig) validatePluginDirectory() error {
	if strings.TrimSpace(p.Dir) == "" {
		return nil
	}

	available, err := files.DiscoverExecutables(p.Dir)
	if err != nil {
		return fmt.Errorf("plugin directory %s: %w", p.Dir, err)
	}

	return p.validateConfiguredPluginsExist(available)
}

// validateConfiguredPluginsExist checks that all configured plugins exist in the available set.
func (p *PluginConfig) validateConfiguredPluginsExist(available map[string]struct{}) error {
	var missingPlugins []error

	for name := range p.PluginNamesDistinct() {
		if _, exists := available[name]; !exists {
			missingPlugins = append(
				missingPlugins,
				fmt.Errorf("plugin %s not found in directory %s", name, p.Dir),
			)
		}
	}

	return errors.Join(missingPlugins...)
}

// categorySlice returns a pointer to the category slice for the given category name.
func (p *PluginConfig) categorySlice(category Category) (*[]PluginEntry, error) {
	switch category {
	case CategoryAuthentication:
		return &p.Authentication, nil
	case CategoryAuthorization:
		return &p.Authorization, nil
	case CategoryRateLimiting:
		return &p.RateLimiting, nil
	case CategoryValidation:
		return &p.Validation, nil
	case CategoryContent:
		return &p.Content, nil
	case CategoryObservability:
		return &p.Observability, nil
	case CategoryAudit:
		return &p.Audit, nil
	default:
		return nil, fmt.Errorf("unknown plugin category: %s", category)
	}
}

// plugin retrieves a plugin by category and name.
func (p *PluginConfig) plugin(category Category, name string) (PluginEntry, bool) {
	if p == nil {
		return PluginEntry{}, false
	}

	slice, err := p.categorySlice(category)
	if err != nil {
		return PluginEntry{}, false
	}

	name = strings.TrimSpace(name)
	for _, entry := range *slice {
		if entry.Name == name {
			return entry, true
		}
	}

	return PluginEntry{}, false
}

// upsertPlugin creates or updates a plugin entry.
func (p *PluginConfig) upsertPlugin(category Category, entry PluginEntry) (context.UpsertResult, error) {
	// Handle sanitizing the plugin name.
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		return context.Noop, fmt.Errorf("plugin name cannot be empty")
	}

	if err := entry.Validate(); err != nil {
		return context.Noop, fmt.Errorf("plugin validation failed: %w", err)
	}

	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	// Check if plugin already exists.
	for i, existing := range *slice {
		if existing.Name != entry.Name {
			continue
		}

		// Plugin exists, update it.
		if existing.Equals(&entry) {
			return context.Noop, nil
		}

		(*slice)[i] = entry
		return context.Updated, nil
	}

	// Plugin doesn't exist, add it.
	*slice = append(*slice, entry)
	return context.Created, nil
}

// deletePlugin removes a plugin entry.
func (p *PluginConfig) deletePlugin(category Category, name string) (context.UpsertResult, error) {
	if p == nil {
		return context.Noop, fmt.Errorf("plugin config is nil")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return context.Noop, fmt.Errorf("plugin name cannot be empty")
	}

	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	// Find and remove the plugin.
	for i, entry := range *slice {
		if entry.Name != name {
			continue
		}

		// Remove by slicing around the element.
		*slice = append((*slice)[:i], (*slice)[i+1:]...)
		return context.Deleted, nil
	}

	return context.Noop, fmt.Errorf("plugin %s not found in category '%s'", name, category)
}

// ListPlugins returns all plugins in a category.
func (p *PluginConfig) ListPlugins(category Category) []PluginEntry {
	if p == nil {
		return nil
	}

	slice, err := p.categorySlice(category)
	if err != nil {
		return nil
	}

	// Return a copy to prevent external modification.
	result := make([]PluginEntry, len(*slice))
	copy(result, *slice)
	return result
}

// AllCategories returns all plugin entries organized by category.
// Only categories with configured plugins are included in the returned map.
func (p *PluginConfig) AllCategories() map[Category][]PluginEntry {
	if p == nil {
		return nil
	}

	result := make(map[Category][]PluginEntry)

	if len(p.Authentication) > 0 {
		result[CategoryAuthentication] = p.Authentication
	}
	if len(p.Authorization) > 0 {
		result[CategoryAuthorization] = p.Authorization
	}
	if len(p.RateLimiting) > 0 {
		result[CategoryRateLimiting] = p.RateLimiting
	}
	if len(p.Validation) > 0 {
		result[CategoryValidation] = p.Validation
	}
	if len(p.Content) > 0 {
		result[CategoryContent] = p.Content
	}
	if len(p.Observability) > 0 {
		result[CategoryObservability] = p.Observability
	}
	if len(p.Audit) > 0 {
		result[CategoryAudit] = p.Audit
	}

	return result
}

// PluginNamesDistinct returns the names of all distinct plugins specified in config.
func (p *PluginConfig) PluginNamesDistinct() map[string]struct{} {
	if p == nil {
		return nil
	}

	all := p.AllCategories()
	result := make(map[string]struct{})

	for _, plugins := range all {
		for _, plugin := range plugins {
			result[plugin.Name] = struct{}{}
		}
	}

	return result
}

// OrderedCategories returns the list of categories in the order they are executed in the plugin pipeline.
// This ordering is important for consistent plugin execution across the system.
func OrderedCategories() Categories {
	return slices.Clone(orderedCategories)
}

// OrderedFlowNames returns the names of allowed flows in order.
func OrderedFlowNames() []string {
	sortedFlows := slices.Sorted(maps.Keys(flows))

	flowNames := make([]string, len(sortedFlows))
	for i, f := range sortedFlows {
		flowNames[i] = string(f)
	}

	return flowNames
}

// Set is used by Cobra to set the category value from a string.
// NOTE: This is also required by Cobra as part of implementing flag.Value.
func (c *Category) Set(v string) error {
	v = filter.NormalizeString(v)
	allowed := OrderedCategories()

	for _, a := range allowed {
		if string(a) == v {
			*c = Category(v)
			return nil
		}
	}

	return fmt.Errorf("invalid category '%s', must be one of %v", v, allowed.String())
}

// Type is used by Cobra/pflag to describe the flag's underlying type.
func (c *Category) Type() string {
	return "category"
}

// String implements fmt.Stringer for a collection of plugin categories,
// converting them to a comma separated string.
func (c Categories) String() string {
	categories := slices.Clone(c)

	slices.Sort(categories)

	out := make([]string, len(categories))
	for i := range categories {
		out[i] = categories[i].String()
	}

	return strings.Join(out, ", ")
}

// MoveOption defines a functional option for configuring plugin move operations.
type MoveOption func(*moveOptions) error

// moveOptions contains configuration for moving a plugin.
type moveOptions struct {
	toCategory *Category
	before     *string
	after      *string
	position   *int
	force      bool
}

func newMoveOptions(opts ...MoveOption) (moveOptions, error) {
	o := moveOptions{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&o); err != nil {
			return moveOptions{}, err
		}
	}

	return o, nil
}

// WithToCategory moves the plugin to a different category.
func WithToCategory(category Category) MoveOption {
	return func(o *moveOptions) error {
		o.toCategory = &category
		return nil
	}
}

// WithBefore positions the plugin before the named plugin.
func WithBefore(name string) MoveOption {
	return func(o *moveOptions) error {
		o.before = &name
		return nil
	}
}

// WithAfter positions the plugin after the named plugin.
func WithAfter(name string) MoveOption {
	return func(o *moveOptions) error {
		o.after = &name
		return nil
	}
}

// WithPosition sets the absolute position (1-based).
func WithPosition(pos int) MoveOption {
	return func(o *moveOptions) error {
		o.position = &pos
		return nil
	}
}

// WithForce overwrites an existing plugin with the same name in the target category.
func WithForce(force bool) MoveOption {
	return func(o *moveOptions) error {
		o.force = force
		return nil
	}
}

// movePlugin moves a plugin based on the provided options.
func (p *PluginConfig) movePlugin(category Category, name string, opts ...MoveOption) (context.UpsertResult, error) {
	options, err := newMoveOptions(opts...)
	if err != nil {
		return context.Noop, err
	}

	// Defaults before any operations take place.
	res := context.Noop
	err = fmt.Errorf("no move operation specified")

	// Cross-category moves can occur in addition to positional moves.
	if options.toCategory != nil {
		res, err = p.moveToCategory(category, name, *options.toCategory, options.force)
		if err != nil {
			return res, err
		}

		// Position operations should now target the new category.
		category = *options.toCategory
	}

	// Positional moves (within whatever category the plugin is now in).
	switch {
	case options.before != nil:
		return p.moveBefore(category, name, *options.before)
	case options.after != nil:
		return p.moveAfter(category, name, *options.after)
	case options.position != nil:
		return p.moveToPosition(category, name, *options.position)
	default:
		return res, err
	}
}

// moveToCategory moves a plugin from one category to another (appends to end).
// If a plugin with the same name exists in the new category,
// then the operation can only succeed if the force parameter is set to true.
func (p *PluginConfig) moveToCategory(
	fromCategory Category,
	name string,
	toCategory Category,
	force bool,
) (context.UpsertResult, error) {
	srcSlice, err := p.categorySlice(fromCategory)
	if err != nil {
		return context.Noop, err
	}

	srcIdx := findPluginIndex(*srcSlice, name)
	if srcIdx == -1 {
		return context.Noop, fmt.Errorf("plugin '%s' not found in category '%s'", name, fromCategory)
	}

	plugin := (*srcSlice)[srcIdx]

	targetSlice, err := p.categorySlice(toCategory)
	if err != nil {
		return context.Noop, err
	}

	existingIdx := findPluginIndex(*targetSlice, name)
	if existingIdx != -1 && !force {
		return context.Noop, fmt.Errorf(
			"plugin '%s' already exists in category '%s', use --force to overwrite",
			name,
			toCategory,
		)
	}

	// Remove from source.
	*srcSlice = append((*srcSlice)[:srcIdx], (*srcSlice)[srcIdx+1:]...)

	// Remove existing in target if force.
	if existingIdx != -1 {
		*targetSlice = append((*targetSlice)[:existingIdx], (*targetSlice)[existingIdx+1:]...)
	}

	// Append to end of target category.
	*targetSlice = append(*targetSlice, plugin)

	return context.Updated, nil
}

// findPluginIndex returns the index of a plugin by name, or -1 if not found.
func findPluginIndex(slice []PluginEntry, name string) int {
	for i, entry := range slice {
		if entry.Name == name {
			return i
		}
	}
	return -1
}

// moveAfter moves a plugin to immediately after the target plugin within the same category.
func (p *PluginConfig) moveAfter(category Category, name string, targetName string) (context.UpsertResult, error) {
	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	items := *slice
	n := len(items)
	if n <= 1 {
		return context.Noop, nil
	}

	srcIdx := slices.IndexFunc(items, func(e PluginEntry) bool { return e.Name == name })
	if srcIdx < 0 {
		return context.Noop, fmt.Errorf("plugin '%s' not found in category '%s'", name, category)
	}

	targetIdx := slices.IndexFunc(items, func(e PluginEntry) bool { return e.Name == targetName })
	if targetIdx < 0 {
		return context.Noop, fmt.Errorf("target plugin '%s' not found in category '%s'", targetName, category)
	}

	// Already in correct position?
	if srcIdx == targetIdx+1 {
		return context.Noop, nil
	}

	// Remove the entry.
	entry := items[srcIdx]
	items = slices.Delete(items, srcIdx, srcIdx+1)

	// If removal was before target, targetIdx shifts left by one.
	if srcIdx < targetIdx {
		targetIdx--
	}

	// Insert after targetIdx.
	items = slices.Insert(items, targetIdx+1, entry)

	*slice = items
	return context.Updated, nil
}

// moveBefore moves a plugin to immediately before the target plugin within the same category.
func (p *PluginConfig) moveBefore(category Category, name string, targetName string) (context.UpsertResult, error) {
	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	items := *slice
	if len(items) <= 1 {
		return context.Noop, nil
	}

	srcIdx := slices.IndexFunc(items, func(e PluginEntry) bool { return e.Name == name })
	if srcIdx < 0 {
		return context.Noop, fmt.Errorf("plugin '%s' not found in category '%s'", name, category)
	}

	targetIdx := slices.IndexFunc(items, func(e PluginEntry) bool { return e.Name == targetName })
	if targetIdx < 0 {
		return context.Noop, fmt.Errorf("target plugin '%s' not found in category '%s'", targetName, category)
	}

	// Already in correct position?
	if srcIdx == targetIdx-1 {
		return context.Noop, nil
	}

	// Remove the entry.
	entry := items[srcIdx]
	items = slices.Delete(items, srcIdx, srcIdx+1)

	// If removal was before target, targetIdx shifts left by one.
	if srcIdx < targetIdx {
		targetIdx--
	}

	// Insert before targetIdx.
	items = slices.Insert(items, targetIdx, entry)

	*slice = items
	return context.Updated, nil
}

// moveToPosition moves a plugin to an absolute position within the same category.
// Position is 1-based. Position -1 means "move to end".
func (p *PluginConfig) moveToPosition(category Category, name string, position int) (context.UpsertResult, error) {
	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	items := *slice
	n := len(items)
	if n <= 1 {
		return context.Noop, nil
	}

	srcIdx := slices.IndexFunc(items, func(e PluginEntry) bool { return e.Name == name })
	if srcIdx < 0 {
		return context.Noop, fmt.Errorf("plugin '%s' not found in category '%s'", name, category)
	}

	// Normalize target index (1-based to 0-based, -1 means end).
	var targetIdx int
	switch {
	case position == -1:
		targetIdx = n - 1
	case position < 1:
		targetIdx = 0
	case position > n:
		targetIdx = n - 1
	default:
		targetIdx = position - 1
	}

	if srcIdx == targetIdx {
		return context.Noop, nil
	}

	// Remove the entry.
	entry := items[srcIdx]
	items = slices.Delete(items, srcIdx, srcIdx+1)

	// Recalculate target position after removal.
	switch {
	case position == -1 || position >= n:
		targetIdx = len(items) // End of shortened list.
	case srcIdx < targetIdx:
		targetIdx--
	}

	// Insert at target position.
	items = slices.Insert(items, targetIdx, entry)

	*slice = items
	return context.Updated, nil
}
