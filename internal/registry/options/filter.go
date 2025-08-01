package options

import (
	"maps"

	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

const (
	// WildcardCharacter represents the character that can be supplied
	// to searches to find 'all' results (before filtering).
	// It is explicitly intended to be used in relation to querying a name.
	WildcardCharacter = "*"

	// FilterKeyName is the key to use for filtering 'name'.
	FilterKeyName = "name"

	// FilterKeyRuntime is the key to use for filtering 'runtime'.
	FilterKeyRuntime = "runtime"

	// FilterKeyTools is the key to use for filtering 'tools'.
	FilterKeyTools = "tools"

	// FilterKeyTags is the key to use for filtering 'tags'.
	FilterKeyTags = "tags"

	// FilterKeyCategories is the key to use for filtering 'categories'.
	FilterKeyCategories = "categories"

	// FilterKeyVersion is the key to use for filtering 'version'.
	FilterKeyVersion = "version"

	// FilterKeyLicense is the key to use for filtering 'license'.
	FilterKeyLicense = "license"

	// FilterKeySource is the key to use for filtering 'source'.
	FilterKeySource = "source"

	// FilterKeyIsOfficial is the key to use for filtering 'isOfficial'.
	FilterKeyIsOfficial = "isOfficial"
)

// Predicate for matching a packages.Package.
type Predicate = filter.Predicate[packages.Package]

// Option for providing a packages.Package.
type Option = filter.Option[packages.Package]

// BoolValueProvider is used to provide a specific boolean value from a packages.Package.
type BoolValueProvider = filter.BoolValueProvider[packages.Package]

// StringValueProvider is used to provide a specific string value from a packages.Package.
type StringValueProvider = filter.StringValueProvider[packages.Package]

// StringValuesProvider is used to provide a specific string slice value from a packages.Package.
type StringValuesProvider = filter.StringValuesProvider[packages.Package]

// Match filters a packages.Package using optional predicate matchers defined by the supplied filter keys.
// Each filter key corresponds to a predicate that (if supplied) evaluates a specific field in the packages.Package.
func Match(pkg packages.Package, filters map[string]string, opts ...Option) (bool, error) {
	return filter.Match(pkg, filters, opts...)
}

// PrepareFilters returns a new map based on the provided filters, ensuring that the FilterKeyName is injected.
// If a registry-specific mutation function is provided, it is applied after the name injection.
// If the mutation function returns an error, PrepareFilters returns that error.
func PrepareFilters(
	filters map[string]string,
	name string,
	mutateFunc func(map[string]string) error,
) (map[string]string, error) {
	var fs map[string]string

	// Ensure name filter is present.
	if _, ok := filters[FilterKeyName]; ok {
		fs = maps.Clone(filters) // Clone if name already present
	} else {
		fs = make(map[string]string, len(filters)+1)
		maps.Copy(fs, filters)
		fs[FilterKeyName] = name
	}

	// Allow for optional mutation and side effects for the new filters.
	if mutateFunc != nil {
		if err := mutateFunc(fs); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// WithLogFunc sets a log function which will be used to log info if unsupported keys are encountered.
func WithLogFunc(fn func(key, val string)) Option {
	return filter.WithLogFunc[packages.Package](fn)
}

// WithUnsupportedKeys marks specific keys as unsupported when used for filtering.
func WithUnsupportedKeys(keys ...string) Option {
	return filter.WithUnsupportedKeys[packages.Package](keys...)
}

// WithNameMatcher returns a filter.Option with a matcher configured for the "name" filter key.
// The matcher is applied during Match only if the name filter key is present in the filters map.
// This matcher performs case-insensitive substring matching across Name, DisplayName, and ID fields.
// If the filter value is "*", all packages match unconditionally.
func WithNameMatcher() Option {
	return filter.WithMatcher(FilterKeyName, withWildcardMatcher(NameProvider, DisplayNameProvider, IDProvider))
}

// WithRuntimeMatcher returns a filter.Option with a matcher configured for the "runtime" filter key.
// The matcher is applied during Match only if the runtime filter key is present in the filters map.
// Matching is case-insensitive and uses normalized values.
func WithRuntimeMatcher(provider StringValuesProvider) Option {
	return filter.WithMatcher(FilterKeyRuntime, filter.HasAny(provider))
}

// WithToolsMatcher returns a filter.Option with a matcher configured for the "tools" filter key.
// The matcher is applied during Match only if the tools filter key is present in the filters map.
// This matcher returns true only if all filter values are found in the package's tools.
// Matching is case-insensitive and uses normalized values.
func WithToolsMatcher(provider StringValuesProvider) Option {
	return filter.WithMatcher(FilterKeyTools, filter.HasAll(provider))
}

// WithTagsMatcher returns a filter.Option with a matcher configured for the "tags" filter key.
// The matcher is applied during Match only if the tags filter key is present in the filters map.
// This matcher returns true if all the filter values are found in the package's tag as substrings.
// Matching is case-insensitive and uses normalized values.
func WithTagsMatcher(provider StringValuesProvider) Option {
	return filter.WithMatcher(FilterKeyTags, filter.PartialAll(provider))
}

// WithCategoriesMatcher returns a filter.Option with a matcher configured for the "categories" filter key.
// The matcher is applied during Match only if the categories filter key is present in the filters map.
// This matcher returns true if all the filter values are found in the package's categories as substrings.
// Matching is case-insensitive and uses normalized values.
func WithCategoriesMatcher(provider StringValuesProvider) Option {
	return filter.WithMatcher(FilterKeyCategories, filter.PartialAll(provider))
}

// WithVersionMatcher returns a filter.Option with a matcher configured for the "version" filter key.
// The matcher is applied during Match only if the version filter key is present in the filters map.
// This matcher performs case-insensitive equality matching on the version field.
func WithVersionMatcher(provider StringValueProvider) Option {
	return filter.WithMatcher(FilterKeyVersion, filter.Equals(provider))
}

// WithLicenseMatcher returns a filter.Option with a matcher configured for the "license" filter key.
// The matcher is applied during Match only if the license filter key is present in the filters map.
// This matcher performs case-insensitive substring matching on the license field.
func WithLicenseMatcher(provider StringValueProvider) Option {
	return filter.WithMatcher(FilterKeyLicense, filter.Partial(provider))
}

// WithSourceMatcher returns a filter.Option with a matcher configured for the "source" filter key.
// The matcher is applied during Match only if the source filter key is present in the filters map.
// This matcher performs case-insensitive equality matching on the source field.
func WithSourceMatcher(provider StringValueProvider) Option {
	return filter.WithMatcher(FilterKeySource, filter.Equals(provider))
}

// WithIsOfficialMatcher returns a filter.Option with a matcher configured for the 'is Official' filter key.
// The matcher is applied during Match only if the 'is_official' filter key is present in the filters map.
// This matcher performs boolean comparison on a parsed input string value, with the provided field.
func WithIsOfficialMatcher(provider BoolValueProvider) Option {
	return filter.WithMatcher(FilterKeyIsOfficial, filter.EqualsBool(provider))
}

func WithDefaultMatchers() Option {
	return filter.WithMatchers(DefaultMatchers())
}

// withWildcardMatcher returns a Predicate that checks if any of the provided
// value providers contain the input string, or matches the WildcardCharacter.
func withWildcardMatcher(providers ...StringValueProvider) Predicate {
	return func(pkg packages.Package, val string) bool {
		q := filter.NormalizeString(val)
		if q == WildcardCharacter {
			return true
		}
		return filter.EqualsAny(providers...)(pkg, q)
	}
}

func DefaultMatchers() map[string]Predicate {
	return map[string]Predicate{
		FilterKeyName:       withWildcardMatcher(NameProvider, DisplayNameProvider, IDProvider),
		FilterKeyRuntime:    filter.HasAny(RuntimesProvider),
		FilterKeyTools:      filter.HasAll(ToolsProvider),
		FilterKeyTags:       filter.PartialAll(TagsProvider),
		FilterKeyCategories: filter.PartialAll(CategoriesProvider),
		FilterKeyVersion:    filter.Equals(VersionProvider),
		FilterKeyLicense:    filter.Partial(LicenseProvider),
		FilterKeySource:     filter.Equals(SourceProvider),
		FilterKeyIsOfficial: filter.EqualsBool(IsOfficialProvider),
	}
}

func DisplayNameProvider(pkg packages.Package) string {
	return pkg.DisplayName
}

func IDProvider(pkg packages.Package) string {
	return pkg.ID
}

func LicenseProvider(pkg packages.Package) string {
	return pkg.License
}

func NameProvider(pkg packages.Package) string {
	return pkg.Name
}

func RuntimesProvider(pkg packages.Package) []string {
	rts := make([]string, len(pkg.Runtimes))
	for i, rt := range pkg.Runtimes {
		rts[i] = string(rt)
	}
	return rts
}

func SourceProvider(pkg packages.Package) string {
	return pkg.Source
}

func TagsProvider(pkg packages.Package) []string {
	return pkg.Tags
}

func CategoriesProvider(pkg packages.Package) []string {
	return pkg.Categories
}

func ToolsProvider(pkg packages.Package) []string {
	return pkg.Tools.Names()
}

func VersionProvider(pkg packages.Package) string {
	return pkg.Version
}

func IsOfficialProvider(pkg packages.Package) bool {
	return pkg.IsOfficial
}
