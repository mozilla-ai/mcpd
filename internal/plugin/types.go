package plugin

import (
	"fmt"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// NOTE: The order of these constants is deliberate.
// ModeSerial is zero so the zero value of CategoryProperties is the most restrictive mode.
const (
	// ModeSerial runs a category's plugins one at a time, in configured order, and applies no modification.
	ModeSerial Mode = iota

	// ModeParallel runs a category's plugins concurrently and applies no modification.
	ModeParallel

	// ModeSerialModifying runs a category's plugins one at a time, and hands each plugin what the plugin
	// before it produced.
	ModeSerialModifying
)

// categoryProps maps each category to its execution properties.
// The pipeline enforces these constraints during request/response processing.
// The zero value is the common case: serial execution, no modification, and optional rejection honored.
var categoryProps = map[config.Category]CategoryProperties{
	config.CategoryAuthentication: {},
	config.CategoryAuthorization:  {},
	config.CategoryRateLimiting:   {},
	config.CategoryValidation:     {},
	config.CategoryContent:        {Mode: ModeSerialModifying},
	config.CategoryObservability:  {Mode: ModeParallel, IgnoreOptionalRejection: true},
	config.CategoryAudit:          {},
}

// orderedCategories stores a copy of the slice that determines the order categories should be executed in the pipeline.
// The order is essentially a constant in the system, and does not change during runtime.
// As the pipeline runs for every request/response we take a copy here and re-use it for efficiency.
// NOTE: This variable must not be mutated within the package.
var orderedCategories = config.OrderedCategories()

// CategoryProperties defines execution semantics for each plugin category.
type CategoryProperties struct {
	// IgnoreOptionalRejection when true ignores Continue=false from optional plugins.
	// Required plugins always cause rejection regardless of this flag.
	// Default (false): honor optional plugin rejection.
	IgnoreOptionalRejection bool

	// Mode determines how the category's plugins run, and whether a modification they return is applied.
	// Default (ModeSerial): sequential execution, no modification.
	Mode Mode
}

// Mode describes how the pipeline runs the plugins in one category.
// It is a single choice rather than a set of flags, so a category cannot ask both to run its plugins
// concurrently and to fold each plugin's modification into the next, which has no meaning without an order.
type Mode int

// PropertiesForCategory returns execution properties for a category.
// Returns an error if the category is unknown.
func PropertiesForCategory(category config.Category) (CategoryProperties, error) {
	props, ok := categoryProps[category]
	if !ok {
		return CategoryProperties{}, fmt.Errorf("unknown plugin category: %s", category)
	}

	return props, nil
}

// CanModify reports whether the pipeline applies a modification returned by a plugin running in this mode.
func (m Mode) CanModify() bool {
	return m == ModeSerialModifying
}

// IsParallel reports whether the pipeline runs this mode's plugins concurrently.
func (m Mode) IsParallel() bool {
	return m == ModeParallel
}

// String returns the name of the mode.
func (m Mode) String() string {
	switch m {
	case ModeParallel:
		return "parallel"
	case ModeSerial:
		return "serial"
	case ModeSerialModifying:
		return "serial-modifying"
	default:
		return fmt.Sprintf("unknown mode: %d", int(m))
	}
}
