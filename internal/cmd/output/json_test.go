package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// testJSONSample type for testing
type testJSONSample struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestNewJSONHandler_Writer(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewJSONHandler[testJSONSample](buf, 2)
	require.Equal(t, buf, h.Writer())
}

func TestJSONHandler_HandleResults(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewJSONHandler[testJSONSample](buf, 2)

	samples := []testJSONSample{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	err := h.HandleResults(samples...)
	require.NoError(t, err)

	// The output should be valid JSON with "results" key containing the array
	expected := `{
  "results": [
    {
      "id": 1,
      "name": "Alice"
    },
    {
      "id": 2,
      "name": "Bob"
    }
  ]
}` + "\n"
	require.Equal(t, expected, buf.String())
}

func TestJSONHandler_HandleResults_Empty(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewJSONHandler[testJSONSample](buf, 0)

	err := h.HandleResults(nil...)
	require.NoError(t, err)

	// With zero indent, expect compact JSON
	expected := `{"results":null}` + "\n"
	require.Equal(t, expected, buf.String())
}

func TestJSONHandler_HandleError(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewJSONHandler[testJSONSample](buf, 4)

	testErr := errors.New("something went wrong")
	err := h.HandleError(testErr)
	require.NoError(t, err)

	// Check that the JSON contains the error message under "error"
	expected := `{
    "error": "something went wrong"
}` + "\n"
	require.Equal(t, expected, buf.String())
}

func TestJSONHandler_HandleError_EmptyMessage(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	h := NewJSONHandler[testJSONSample](buf, 0)

	err := h.HandleError(errors.New(""))
	require.NoError(t, err)

	// Even empty error string should be marshaled
	expected := `{"error":""}` + "\n"
	require.Equal(t, expected, buf.String())
}
