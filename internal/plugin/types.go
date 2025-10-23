package plugin

import (
	"fmt"

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

// orderedCategories stores a copy of the slice that determines the order categories should be executed in the pipeline.
// The order is essentially a constant in the system, and does not change during runtime.
// As the pipeline runs for every request/response we take a copy here and re-use it for efficiency.
// NOTE: This variable must not be mutated within the package.
var orderedCategories = config.OrderedCategories()

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

// PropertiesForCategory returns execution properties for a category.
// Returns an error if the category is unknown.
func PropertiesForCategory(category config.Category) (CategoryProperties, error) {
	if props, ok := categoryProps[category]; ok {
		return props, nil
	}

	return CategoryProperties{}, fmt.Errorf("unknown plugin category: %s", category)
}
