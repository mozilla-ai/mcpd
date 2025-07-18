package filter

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Predicate defines a function that returns true if the given item matches a condition.
type Predicate[T any] func(item T, filterValue string) bool

// Options holds configuration for filtering behavior.
type Options[T any] struct {
	matchers    map[string]Predicate[T]
	unsupported map[string]struct{}
	logFunc     func(key string, val string)
}

// Option configures filter Options.
type Option[T any] func(*Options[T]) error

// defaultOptions returns the default filter Options.
func defaultOptions[T any]() Options[T] {
	return Options[T]{
		matchers:    make(map[string]Predicate[T]),
		unsupported: make(map[string]struct{}),
		logFunc:     func(key, val string) {}, // no-op
	}
}

// NormalizeString can be used to normalize a string value for filtering/comparison.
// The value is made lowercase and has any leading and/or trailing whitespace removed.
func NormalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeSlice can be used to normalize all values of a slice, returning a new slice.
// The values are normalized with the same behavior as NormalizeString.
func NormalizeSlice(s []string) []string {
	s2 := make([]string, len(s))
	for i := range s {
		s2[i] = NormalizeString(s[i])
	}
	return s2
}

// NewOptions creates a FilterOptions with defaults and applies given options.
func NewOptions[T any](opt ...Option[T]) (Options[T], error) {
	opts := defaultOptions[T]()

	for _, o := range opt {
		if o == nil {
			continue
		}
		if err := o(&opts); err != nil {
			return Options[T]{}, err
		}
	}
	return opts, nil
}

// Provider is a generic function type that encapsulates the logic for extracting
// a value of type V from an filterValue of type T. It provides a flexible way to
// retrieve specific data from various types of structures or sources.
type Provider[T any, V any] func(T) V

// BoolValueProvider extracts a single boolean value from an item of type T.
type BoolValueProvider[T any] Provider[T, bool]

// StringValueProvider extracts a single string value from an item of type T.
type StringValueProvider[T any] Provider[T, string]

// StringValuesProvider extracts a slice of string values from an item of type T.
// type ValuesProvider[T any] func(T) []string
type StringValuesProvider[T any] Provider[T, []string]

// Equals returns a Predicate that checks if the value extracted by the provider
// exactly matches the filter value (case-insensitive, normalized).
//
// Example:
//
// predicate := Equals(options.SourceProvider),
// result := predicate(pkg, "github") // true if pkg.Source equals "github"
func Equals[T any](provider StringValueProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		actual := NormalizeString(provider(item))
		expected := NormalizeString(val)
		return actual == expected
	}
}

// EqualsBool returns a Predicate that checks if the value extracted by the provider
// matches the parsed boolean representation of the filter value.
//
// Example:
//
// predicate := EqualsBool(options.IsOfficialProvider),
// result := predicate(pkg, "true") // true if pkg.IsOfficial is true
func EqualsBool[T any](provider BoolValueProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		parsedVal, err := strconv.ParseBool(NormalizeString(val))
		if err != nil {
			return false
		}
		return provider(item) == parsedVal
	}
}

// Partial returns a Predicate that checks if the value extracted by the provider
// contains the filter value as a substring (case-insensitive, normalized).
//
// Example:
//
// predicate := Partial(options.VersionProvider),
// result := predicate(pkg, "1.2") // true if pkg.Version contains "1.2"
func Partial[T any](provider StringValueProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		actual := NormalizeString(provider(item))
		expected := NormalizeString(val)
		return strings.Contains(actual, expected)
	}
}

// PartialAll returns a Predicate that checks if *ALL* comma-separated values in the filter string are found
// as substrings within provided values (case-insensitive, normalized).
// Functionally similar to Partial, but operates on a ValuesProvider, and expects the filter to be comma-separated.
//
// Example:
//
// predicate := PartialAll(options.ToolsProvider),
// result := predicate(pkg, "get_current_time,convert_time") // true if pkg.Tools contains values with "get_current_time" and "convert_time" as substrings
func PartialAll[T any](provider StringValuesProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		required := NormalizeSlice(strings.Split(val, ","))
		actual := NormalizeSlice(provider(item))

		for _, v := range required {
			found := false
			for _, a := range actual {
				if strings.Contains(a, v) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
}

// EqualsAny returns a Predicate that checks if *ANY* of the values from the supplied providers are equal to the
// filter value (case-insensitive, normalized).
// Functionally similar to Equals, but operates on one or more StringValueProvider.
//
// Example:
//
// predicate := EqualsAny(options.ToolsProvider),
// result := predicate(pkg, "get_current_time,convert_time") // true if pkg.Tools contains values "get_current_time" or "convert_time"
func EqualsAny[T any](providers ...StringValueProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		q := NormalizeString(val)
		for _, p := range providers {
			actual := NormalizeString(p(item))
			if strings.Contains(actual, q) {
				return true
			}
		}
		return false
	}
}

// HasOnly returns a Predicate that checks if the values extracted by the provider are a subset of
// the comma-separated values in the filter string (case-insensitive, normalized).
// Returns true only if *ALL* extracted values are present in the filter list.
//
// Example:
//
// predicate := HasOnly(options.ToolsProvider),
// result := predicate(pkg, "get_current_time,convert_time") // true if pkg.Tools only contains tools from the list
func HasOnly[T any](provider StringValuesProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		required := strings.Split(val, ",")
		expected := make(map[string]struct{}, len(required))

		for _, v := range required {
			expected[NormalizeString(v)] = struct{}{}
		}

		for _, v := range provider(item) {
			if _, ok := expected[NormalizeString(v)]; !ok {
				return false
			}
		}
		return true
	}
}

// HasAll returns a Predicate that checks if the values extracted by the provider include *ALL*
// of the comma-separated values in the filter string (case-insensitive, normalized)..
// Returns true only if *ALL* required values are present in the extracted values.
//
// Example:
//
// predicate := HasAll(options.ToolsProvider),
// result := predicate(pkg, "get_current_time,convert_time") // true if pkg.Tools contains both "get_current_time" and "convert_time"
func HasAll[T any](provider StringValuesProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		required := NormalizeSlice(strings.Split(val, ","))
		actual := provider(item)
		allowed := make(map[string]struct{}, len(actual))

		for _, v := range actual {
			allowed[NormalizeString(v)] = struct{}{}
		}

		for _, r := range required {
			if _, ok := allowed[r]; !ok {
				return false
			}
		}
		return true
	}
}

// HasAny returns a Predicate that checks if the values extracted by the provider include *ANY* of
// the comma-separated values in the filter string (case-insensitive, normalized).
// Returns true if at least one required value is present in the extracted values.
//
// Example:
//
// predicate := HasAny(options.ToolsProvider),
// result := predicate(pkg, "get_current_time,convert_time") // true if pkg.Tools contains either "get_current_time" or "convert_time"
func HasAny[T any](provider StringValuesProvider[T]) Predicate[T] {
	return func(item T, val string) bool {
		required := NormalizeSlice(strings.Split(val, ","))
		allowed := make(map[string]struct{}, len(required))

		for _, v := range required {
			allowed[v] = struct{}{}
		}

		for _, v := range provider(item) {
			if _, ok := allowed[NormalizeString(v)]; ok {
				return true
			}
		}
		return false
	}
}

// WithMatchers adds or overrides matchers.
func WithMatchers[T any](m map[string]Predicate[T]) Option[T] {
	return func(o *Options[T]) error {
		for k, v := range m {
			o.matchers[NormalizeString(k)] = v
		}
		return nil
	}
}

// WithMatcher adds or overrides a matcher.
func WithMatcher[T any](key string, value Predicate[T]) Option[T] {
	return func(o *Options[T]) error {
		o.matchers[NormalizeString(key)] = value
		return nil
	}
}

// WithUnsupportedKeys marks specific keys as unsupported when used for filtering.
func WithUnsupportedKeys[T any](keys ...string) Option[T] {
	return func(o *Options[T]) error {
		for _, key := range keys {
			k := NormalizeString(key)
			o.unsupported[k] = struct{}{}
		}
		return nil
	}
}

// WithLogFunc sets a log function which will be used to log info if unsupported keys are encountered.
func WithLogFunc[T any](logFunc func(key string, val string)) Option[T] {
	return func(o *Options[T]) error {
		if logFunc != nil {
			o.logFunc = logFunc
		}
		return nil
	}
}

// Match applies the provided filters to an item of type T using any configured Option matchers.
// It returns false if any unsupported filter key is encountered or if any matcher fails to validate
// the corresponding field.
func Match[T any](item T, filters map[string]string, opts ...Option[T]) (bool, error) {
	if filters == nil {
		return true, nil
	}

	filterOpts, err := NewOptions(opts...)
	if err != nil {
		return false, err
	}

	for key, val := range filters {
		k := NormalizeString(key)
		if k == "" {
			continue
		}

		// Check unsupported
		if _, unsupported := filterOpts.unsupported[k]; unsupported {
			filterOpts.logFunc(k, val)
			return false, nil // TODO: Should unsupported filters just log and continue?
		}

		// Check for an associated matcher, and try to match.
		matcher, ok := filterOpts.matchers[k]
		if !ok {
			continue
		}
		if !matcher(item, val) {
			return false, nil
		}
	}
	return true, nil
}

// MatchRequestedSlice returns normalized values from `requested` that are found in `available`.
// It returns an error if any requested value is not found in the available set.
func MatchRequestedSlice(requested []string, available []string) ([]string, error) {
	availableSet := make(map[string]struct{}, len(available))
	for _, v := range available {
		availableSet[NormalizeString(v)] = struct{}{}
	}

	if len(requested) == 0 {
		return slices.Collect(maps.Keys(availableSet)), nil
	}

	requestedSet := make(map[string]struct{}, len(requested))
	missing := make([]string, 0)

	for _, v := range requested {
		n := NormalizeString(v)
		requestedSet[n] = struct{}{}
		if _, ok := availableSet[n]; !ok {
			missing = append(missing, v)
		}
	}

	switch len(missing) {
	case 0:
		return slices.Collect(maps.Keys(requestedSet)), nil
	case len(requestedSet):
		return nil, fmt.Errorf("none of the requested values were found")
	default:
		sort.Strings(missing)
		return nil, fmt.Errorf("missing values: %s", strings.Join(missing, ", "))
	}
}
