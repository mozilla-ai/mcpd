package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testYAMLSample type for testing
type testYAMLSample struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

func TestNewYAMLHandler_Writer(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewYAMLHandler[testYAMLSample](buf, 3)
	require.Equal(t, buf, h.Writer())
}

func TestYAMLHandler_HandleResults(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewYAMLHandler[testYAMLSample](buf, 2)

	samples := []testYAMLSample{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	err := h.HandleResults(samples...)
	require.NoError(t, err)

	// Expect YAML with top-level 'results' sequence
	expected := "results:\n" +
		"  - id: 1\n" +
		"    name: Alice\n" +
		"  - id: 2\n" +
		"    name: Bob\n"
	require.Equal(t, expected, buf.String())
}

func TestYAMLHandler_HandleResults_Empty(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewYAMLHandler[testYAMLSample](buf, 0)

	err := h.HandleResults(nil...)
	require.NoError(t, err)

	// Nil slice yields null results
	requiredOut := buf.String()
	require.Contains(t, requiredOut, "results:")
	// Check for null or empty array
	require.True(t, strings.Contains(requiredOut, "null") || strings.Contains(requiredOut, "[]"))
}

func TestYAMLHandler_HandleError(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewYAMLHandler[testYAMLSample](buf, 4)

	testErr := errors.New("something went wrong")
	err := h.HandleError(testErr)
	require.NoError(t, err)

	// Expect YAML with top-level 'error' key
	expected := "error: something went wrong\n"
	require.Equal(t, expected, buf.String())
}

func TestYAMLHandler_HandleError_EmptyMessage(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	h := NewYAMLHandler[testYAMLSample](buf, 0)

	err := h.HandleError(errors.New(""))
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "error:")
	// Empty message should result in empty string after key
	require.Regexp(t, `^error:\s""`, out)
	expected := "error: \"\"\n"
	require.Equal(t, expected, buf.String())
}
