package plugin

import (
	"fmt"
	"slices"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// categoryProps maps each category to its execution properties.
// The pipeline enforces these constraints during request/response processing.
// Default behavior: serial execution, respect optional rejection, no modification.
// True values mark special cases (Observability: parallel and non-blocking, Content: can modify).
var categoryProps = map[config.Category]CategoryProperties{
	config.CategoryAuthentication: {},
	config.CategoryAuthorization:  {},
	config.CategoryRateLimiting:   {},
	config.CategoryValidation:     {},
	config.CategoryContent:        {CanModify: true},
	config.CategoryObservability:  {Parallel: true, IgnoreOptionalRejection: true},
	config.CategoryAudit:          {},
}

// orderedCategories defines the pipeline execution order.
// Categories execute in this sequence for each request/response.
var orderedCategories = []config.Category{
	config.CategoryObservability, // First, parallel, non-blocking.
	config.CategoryAuthentication,
	config.CategoryAuthorization,
	config.CategoryRateLimiting,
	config.CategoryValidation,
	config.CategoryContent,
	config.CategoryAudit, // Last.
}

// CategoryProperties defines execution semantics for each plugin category.
// Defaults (zero values) represent the common case: serial, blocking, no modification.
type CategoryProperties struct {
	// Parallel when true executes plugins concurrently.
	// Default (false): sequential execution.
	Parallel bool

	// IgnoreOptionalRejection when true ignores Continue=false from optional plugins.
	// Required plugins always cause rejection regardless of this flag.
	// Default (false): honor optional plugin rejection.
	IgnoreOptionalRejection bool

	// CanModify when true allows plugins to mutate the request/response object.
	// Default (false): no modification allowed.
	CanModify bool
}

// OrderedCategories returns the list of categories in execution order.
func OrderedCategories() []config.Category {
	return slices.Clone(orderedCategories)
}

// PropertiesForCategory returns execution properties for a category.
// Returns an error if the category is unknown.
func PropertiesForCategory(category config.Category) (CategoryProperties, error) {
	if props, ok := categoryProps[category]; ok {
		return props, nil
	}

	return CategoryProperties{}, fmt.Errorf("unknown plugin category: %s", category)
}
