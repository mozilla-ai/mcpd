package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name     string
	Category string
	Tags     []string
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
	p := Contains(func(m testItem) string { return m.Category })
	assert.True(t, p(testItem{Category: "devtools"}, "tool"))
	assert.False(t, p(testItem{Category: "runtime"}, "tool"))
}

func TestContainsOnly(t *testing.T) {
	p := ContainsOnly(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"A", "B"}}, "a,b"))
	assert.False(t, p(testItem{Tags: []string{"A", "C"}}, "a,b"))
}

func TestContainsAll(t *testing.T) {
	p := ContainsAll(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"X", "Y", "Z"}}, "x,y"))
	assert.False(t, p(testItem{Tags: []string{"X", "Y"}}, "x,y,z"))
}

func TestContainsAny(t *testing.T) {
	p := ContainsAny(func(m testItem) []string { return m.Tags })
	assert.True(t, p(testItem{Tags: []string{"alpha", "beta"}}, "beta,gamma"))
	assert.False(t, p(testItem{Tags: []string{"alpha"}}, "beta,gamma"))
}

func TestOrContains(t *testing.T) {
	p := OrContains(
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
