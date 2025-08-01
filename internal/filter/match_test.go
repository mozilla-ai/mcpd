package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name       string
	Category   string
	Tags       []string
	IsOfficial bool
}

func TestNormalizeString(t *testing.T) {
	assert.Equal(t, "hello", NormalizeString("  Hello "))
	assert.Equal(t, "world", NormalizeString("WORLD"))
	assert.Equal(t, "", NormalizeString("  "))
}

func TestNormalizeSlice(t *testing.T) {
	input := []string{"  A ", "b", " C"}
	expected := []string{"a", "b", "c"}
	assert.Equal(t, expected, NormalizeSlice(input))
}

func TestEquals(t *testing.T) {
	p := Equals(func(m testItem) string { return m.Name })
	assert.True(t, p(testItem{Name: "ToolA"}, "toola"))
	assert.False(t, p(testItem{Name: "ToolB"}, "toola"))
}

func TestContains(t *testing.T) {
	p := Partial(func(m testItem) string { return m.Category })
	assert.True(t, p(testItem{Category: "devtools"}, "tool"))
	assert.False(t, p(testItem{Category: "runtime"}, "tool"))
}

func TestHasOnly(t *testing.T) {
	p := HasOnly(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"A", "B"}}, "a,b"))
	assert.False(t, p(testItem{Tags: []string{"A", "C"}}, "a,b"))
}

func TestHasAll(t *testing.T) {
	p := HasAll(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"X", "Y", "Z"}}, "x,y"))
	assert.False(t, p(testItem{Tags: []string{"X", "Y"}}, "x,y,z"))
}

func TestHasAny(t *testing.T) {
	p := HasAny(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"alpha", "beta"}}, "beta,gamma"))
	assert.False(t, p(testItem{Tags: []string{"alpha"}}, "beta,gamma"))
}

func TestPartialAll(t *testing.T) {
	p := PartialAll(func(m testItem) []string { return m.Tags })
	// foo is a substring of foo2, bar is a substring of bar3, so both (all) match.
	assert.True(t, p(testItem{Name: "a", Tags: []string{"foo2", "bar3"}}, "foo,bar"))
	// foo is a substring of foo2, filter was only a single value so all have matched.
	assert.True(t, p(testItem{Name: "a", Tags: []string{"foo2", "bar3"}}, "foo"))
	// foo doesn't match any of the values even as substrings.
	assert.False(t, p(testItem{Name: "b", Tags: []string{"baz", "bar"}}, "foo"))
}

func TestEqualsAny(t *testing.T) {
	p := EqualsAny(
		func(m testItem) string { return m.Name },
		func(m testItem) string { return m.Category },
	)
	assert.True(t, p(testItem{Name: "foo", Category: "tools"}, "tools"))
	assert.False(t, p(testItem{Name: "foo", Category: "bar"}, "baz"))
}

func TestMatchRequestedSlice(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		available []string
		want      []string
		wantErr   string
	}{
		{
			name:      "all matched",
			requested: []string{" A ", "b"},
			available: []string{"a", "b", "c"},
			want:      []string{"a", "b"},
		},
		{
			name:      "partial match with missing",
			requested: []string{"A", "X"},
			available: []string{"a", "b"},
			wantErr:   "missing values: X",
		},
		{
			name:      "none matched",
			requested: []string{"x", "y"},
			available: []string{"a", "b"},
			wantErr:   "none of the requested values were found",
		},
		{
			name:      "empty requested returns all available",
			requested: []string{},
			available: []string{"A", "B"},
			want:      []string{"a", "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchRequestedSlice(tc.requested, tc.available)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.want, got)
			}
		})
	}
}

func TestMatch_BasicEquals(t *testing.T) {
	matchers := map[string]Predicate[testItem]{
		"name": Equals(func(i testItem) string { return i.Name }),
	}

	m := testItem{Name: "abc123"}
	match, err := Match(m, map[string]string{"id": "ABC123"}, WithMatchers(matchers))
	require.NoError(t, err)
	assert.True(t, match)
}

func TestMatch_EqualsBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		itemVal       bool
		filterValue   string
		expectedMatch bool
	}{
		{
			name:          "true",
			itemVal:       true,
			filterValue:   "true",
			expectedMatch: true,
		},
		{
			name:          "true whitespace",
			itemVal:       true,
			filterValue:   "  true  ",
			expectedMatch: true,
		},
		{
			name:          "true upper",
			itemVal:       true,
			filterValue:   "TRUE",
			expectedMatch: true,
		},
		{
			name:          "true mixed",
			itemVal:       true,
			filterValue:   "TruE",
			expectedMatch: true,
		},
		{
			name:          "true all",
			itemVal:       true,
			filterValue:   "  trUE  ",
			expectedMatch: true,
		},
		{
			name:          "false",
			itemVal:       false,
			filterValue:   "false",
			expectedMatch: true,
		},
		{
			name:          "false whitespace",
			itemVal:       false,
			filterValue:   "  false  ",
			expectedMatch: true,
		},
		{
			name:          "false upper",
			itemVal:       false,
			filterValue:   "FALSE",
			expectedMatch: true,
		},
		{
			name:          "false mixed",
			itemVal:       false,
			filterValue:   "FalsE",
			expectedMatch: true,
		},
		{
			name:          "false all",
			itemVal:       false,
			filterValue:   "  FAlse  ",
			expectedMatch: true,
		},
		{
			name:          "no match when filter is non-bool value",
			itemVal:       false,
			filterValue:   "hello",
			expectedMatch: false,
		},
		{
			name:          "no match when filter true, but item false",
			itemVal:       false,
			filterValue:   "true",
			expectedMatch: false,
		},
		{
			name:          "no match when filter false, but item true",
			itemVal:       true,
			filterValue:   "false",
			expectedMatch: false,
		},
	}

	key := "isOfficial"
	matcher := WithMatcher(key, EqualsBool(func(i testItem) bool { return i.IsOfficial }))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := testItem{IsOfficial: tc.itemVal}
			filters := map[string]string{key: tc.filterValue}
			match, err := Match(item, filters, matcher)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedMatch, match)
		})
	}
}

func TestMatch_FailsOnMismatch(t *testing.T) {
	matchers := map[string]Predicate[testItem]{
		"group": Equals(func(i testItem) string { return i.Category }),
	}

	m := testItem{Category: "core"}
	match, err := Match(m, map[string]string{"group": "tools"}, WithMatchers(matchers))
	require.NoError(t, err)
	assert.False(t, match)
}

func TestMatch_IgnoresUnknownKeys(t *testing.T) {
	matchers := map[string]Predicate[testItem]{
		"group": Equals(func(i testItem) string { return i.Category }),
	}

	m := testItem{Category: "infra"}
	match, err := Match(m, map[string]string{"group": "infra", "unknown": "x"}, WithMatchers(matchers))
	require.NoError(t, err)
	assert.True(t, match)
}

func TestMatch_WithUnsupportedKey(t *testing.T) {
	var loggedKey, loggedVal string
	logFunc := func(k, v string) {
		loggedKey = k
		loggedVal = v
	}

	m := testItem{Name: "abc"}
	match, err := Match(
		m,
		map[string]string{"name": "abc"},
		WithUnsupportedKeys[testItem]("name"),
		WithLogFunc[testItem](logFunc),
	)
	require.NoError(t, err)
	assert.False(t, match)
	assert.Equal(t, "name", loggedKey)
	assert.Equal(t, "abc", loggedVal)
}

func TestMatch_NoFilters(t *testing.T) {
	m := testItem{Name: "anything"}
	match, err := Match(m, nil)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestMatch_EmptyKeySkips(t *testing.T) {
	m := testItem{Name: "abc123"}
	match, err := Match(m, map[string]string{"": "skip"})
	require.NoError(t, err)
	assert.True(t, match)
}
