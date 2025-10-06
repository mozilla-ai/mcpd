package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

const (
	// CategoryAuthentication represents authentication plugins.
	CategoryAuthentication = "authentication"

	// CategoryAuthorization represents authorization plugins.
	CategoryAuthorization = "authorization"

	// CategoryRateLimiting represents rate limiting plugins.
	CategoryRateLimiting = "rate_limiting"

	// CategoryValidation represents validation plugins.
	CategoryValidation = "validation"

	// CategoryContent represents content transformation plugins.
	CategoryContent = "content"

	// CategoryObservability represents observability plugins.
	CategoryObservability = "observability"

	// CategoryAudit represents audit/compliance logging plugins.
	CategoryAudit = "audit"
)

// Flow represents the execution phase for a plugin.
type Flow string

const (
	// FlowRequest indicates the plugin executes during the request phase.
	FlowRequest Flow = "request"

	// FlowResponse indicates the plugin executes during the response phase.
	FlowResponse Flow = "response"
)

// PluginModifier defines operations for managing plugin configuration.
type PluginModifier interface {
	// Plugin retrieves a plugin by category and name.
	Plugin(category string, name string) (PluginEntry, bool)

	// UpsertPlugin creates or updates a plugin entry.
	UpsertPlugin(category string, entry PluginEntry) (context.UpsertResult, error)

	// DeletePlugin removes a plugin entry.
	DeletePlugin(category string, name string) (context.UpsertResult, error)

	// ListPlugins returns all plugins in a category.
	ListPlugins(category string) []PluginEntry
}

// PluginConfig represents the top-level plugin configuration.
//
// NOTE: if you add/remove fields you must review the associated validation implementation.
type PluginConfig struct {
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

	// Compare Flows (order matters for array comparison).
	if len(e.Flows) != len(other.Flows) {
		return false
	}
	for i, flow := range e.Flows {
		if flow != other.Flows[i] {
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
			// Check for valid flow values.
			if flow != FlowRequest && flow != FlowResponse {
				validationErrors = append(
					validationErrors,
					fmt.Errorf("invalid flow '%s', must be '%s' or '%s'", flow, FlowRequest, FlowResponse),
				)
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
		name    string
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

	return errors.Join(validationErrors...)
}

// categorySlice returns a pointer to the category slice for the given category name.
func (p *PluginConfig) categorySlice(category string) (*[]PluginEntry, error) {
	category = normalizeKey(category)

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
func (p *PluginConfig) plugin(category string, name string) (PluginEntry, bool) {
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
func (p *PluginConfig) upsertPlugin(category string, entry PluginEntry) (context.UpsertResult, error) {
	if strings.TrimSpace(entry.Name) == "" {
		return context.Noop, fmt.Errorf("plugin name cannot be empty")
	}

	if err := entry.Validate(); err != nil {
		return context.Noop, fmt.Errorf("plugin validation failed: %w", err)
	}

	slice, err := p.categorySlice(category)
	if err != nil {
		return context.Noop, err
	}

	name := strings.TrimSpace(entry.Name)

	// Check if plugin already exists.
	for i, existing := range *slice {
		if existing.Name != name {
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
func (p *PluginConfig) deletePlugin(category string, name string) (context.UpsertResult, error) {
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

	return context.Noop, fmt.Errorf("plugin %q not found in category %s", name, category)
}

// listPlugins returns all plugins in a category.
func (p *PluginConfig) listPlugins(category string) []PluginEntry {
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
