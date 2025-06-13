// internal/registry/filter.go

package registry

import (
	"fmt"
	"strings"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"
)

const WildcardNameQuery = "*"

// FilterPackagePredicate defines a function type that evaluates whether a given
// PackageResult satisfies a specific condition based on a string parameter.
// It returns true if the PackageResult matches the condition, false otherwise.
//
// This type is intended for use as a predicate function in filtering operations,
// allowing flexible and reusable filtering logic to be passed as arguments.
type FilterPackagePredicate func(pkg types.PackageResult, filterValue string) bool

type FilterOption func(*filterOptions) error

type filterOptions struct {
	matchers    map[string]FilterPackagePredicate
	unsupported map[string]struct{}
	logFn       func(key, val string)
}

func getDefaultFilterOptions() filterOptions {
	return filterOptions{
		matchers: map[string]FilterPackagePredicate{
			"runtime": sliceMatchPredicate(func(pkg types.PackageResult) []string { return pkg.Runtimes }),
			"tool":    sliceContainsAllPredicate(func(pkg types.PackageResult) []string { return pkg.Tools }),
			"name":    FilterByName, // Reserved filter used to check name
		},
		unsupported: map[string]struct{}{},    // nothing unsupported by default
		logFn:       func(key, val string) {}, // no-op by default
	}
}

func getFilterOpts(opts ...FilterOption) (filterOptions, error) {
	fo := getDefaultFilterOptions()
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&fo); err != nil {
			return filterOptions{}, err
		}
	}
	return fo, nil
}

func sliceContainsAllPredicate(extract func(types.PackageResult) []string) FilterPackagePredicate {
	return func(pkg types.PackageResult, val string) bool {
		required := strings.Split(val, ",")
		have := extract(pkg)
		set := make(map[string]struct{}, len(have))
		for _, v := range have {
			set[strings.ToLower(v)] = struct{}{}
		}
		for _, req := range required {
			if _, ok := set[strings.ToLower(strings.TrimSpace(req))]; !ok {
				return false
			}
		}
		return true
	}
}

func sliceMatchPredicate(extract func(types.PackageResult) []string) FilterPackagePredicate {
	return func(pkg types.PackageResult, val string) bool {
		for _, item := range extract(pkg) {
			if strings.EqualFold(item, val) {
				return true
			}
		}
		return false
	}
}

// MatchWithFilters applies filters to a PackageResult using configured matchers and unsupported filters.
// It returns false immediately if any unsupported filter is found or any matcher returns false.
func MatchWithFilters(result types.PackageResult, filters map[string]string, opts ...FilterOption) (bool, error) {
	if filters == nil {
		return true, nil
	}

	filterOpts, err := getFilterOpts(opts...)
	if err != nil {
		return false, err
	}

	for key, val := range filters {
		keyCaseInsensitive := strings.ToLower(key)
		if _, skip := filterOpts.unsupported[keyCaseInsensitive]; skip {
			filterOpts.logFn(keyCaseInsensitive, val)
			return false, nil
		}

		matcher, ok := filterOpts.matchers[keyCaseInsensitive]
		if !ok {
			continue
		}
		if !matcher(result, val) {
			return false, nil
		}
	}
	return true, nil
}

func MatchWithGetterOptions(pkg types.PackageResult, opts types.GetterOptions, filterOpts ...FilterOption) (bool, error) {
	filters := make(map[string]string)

	if opts.Runtime != "" {
		filters["runtime"] = string(opts.Runtime)
	}
	if len(opts.Tools) > 0 {
		// Assuming tools are AND-ed for matching
		filters["tools"] = strings.Join(opts.Tools, ",")
	}
	if opts.Version != "" {
		filters["version"] = opts.Version
	}

	return MatchWithFilters(pkg, filters, filterOpts...)
}

func FilterByName(pkg types.PackageResult, val string) bool {
	q := strings.ToLower(strings.TrimSpace(val))
	if q == WildcardNameQuery {
		return true
	}
	return strings.Contains(strings.ToLower(pkg.Name), q) ||
		strings.Contains(strings.ToLower(pkg.DisplayName), q) ||
		strings.Contains(strings.ToLower(pkg.ID), q)
}

func WithMatchers(m map[string]FilterPackagePredicate) FilterOption {
	return func(o *filterOptions) error {
		for k, v := range m {
			normalized := strings.ToLower(k)
			if normalized == "name" {
				return fmt.Errorf("matcher key 'name' is reserved")
			}
			o.matchers[normalized] = v
		}
		return nil
	}
}

func WithUnsupportedFilters(unsupportedKeys ...string) FilterOption {
	return func(o *filterOptions) error {
		for _, key := range unsupportedKeys {
			o.unsupported[key] = struct{}{}
		}
		return nil
	}
}

func WithLogFunc(log func(key, val string)) FilterOption {
	return func(o *filterOptions) error {
		if log == nil {
			return nil
		}
		o.logFn = log
		return nil
	}
}
