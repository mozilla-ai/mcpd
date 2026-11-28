package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestCategoryProperties_AllCategoriesDefined(t *testing.T) {
	t.Parallel()

	require.Len(t, categoryProps, 7, "expected exactly 7 plugin categories")

	tests := []struct {
		name                    string
		category                config.Category
		mode                    Mode
		ignoreOptionalRejection bool
	}{
		{
			name:                    "audit",
			category:                config.CategoryAudit,
			mode:                    ModeSerial,
			ignoreOptionalRejection: false,
		},
		{
			name:                    "authentication",
			category:                config.CategoryAuthentication,
			mode:                    ModeSerial,
			ignoreOptionalRejection: false,
		},
		{
			name:                    "authorization",
			category:                config.CategoryAuthorization,
			mode:                    ModeSerial,
			ignoreOptionalRejection: false,
		},
		{
			name:                    "content",
			category:                config.CategoryContent,
			mode:                    ModeSerialModifying,
			ignoreOptionalRejection: false,
		},
		{
			name:                    "observability",
			category:                config.CategoryObservability,
			mode:                    ModeParallel,
			ignoreOptionalRejection: true,
		},
		{
			name:                    "rate limiting",
			category:                config.CategoryRateLimiting,
			mode:                    ModeSerial,
			ignoreOptionalRejection: false,
		},
		{
			name:                    "validation",
			category:                config.CategoryValidation,
			mode:                    ModeSerial,
			ignoreOptionalRejection: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			props, err := PropertiesForCategory(tc.category)
			require.NoError(t, err)
			require.Equal(t, tc.mode, props.Mode)
			require.Equal(t, tc.ignoreOptionalRejection, props.IgnoreOptionalRejection)
		})
	}
}

// TestCategoryProperties_ParallelCannotModify pins the table against a mode that would hand a plugin
// the output of a plugin that has no defined position before it.
func TestCategoryProperties_ParallelCannotModify(t *testing.T) {
	t.Parallel()

	for category, props := range categoryProps {
		if props.Mode.IsParallel() {
			require.False(t, props.Mode.CanModify(),
				"category %s: parallel execution must not allow modification", category)
		}
	}
}

func TestCategoryProperties_UnknownCategory(t *testing.T) {
	t.Parallel()

	_, err := PropertiesForCategory("unknown-category")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown plugin category")
}

// TestCategoryProperties_ZeroValueIsMostRestrictive pins ModeSerial to zero, so a category added to the
// table without naming a mode runs serially and applies no modification.
func TestCategoryProperties_ZeroValueIsMostRestrictive(t *testing.T) {
	t.Parallel()

	var props CategoryProperties

	require.Equal(t, ModeSerial, props.Mode)
	require.False(t, props.Mode.CanModify())
	require.False(t, props.Mode.IsParallel())
	require.False(t, props.IgnoreOptionalRejection)
}

func TestMode_CanModify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     Mode
		expected bool
	}{
		{
			name:     "parallel",
			mode:     ModeParallel,
			expected: false,
		},
		{
			name:     "serial",
			mode:     ModeSerial,
			expected: false,
		},
		{
			name:     "serial modifying",
			mode:     ModeSerialModifying,
			expected: true,
		},
		{
			name:     "unrecognized",
			mode:     Mode(99),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.mode.CanModify())
		})
	}
}

func TestMode_IsParallel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     Mode
		expected bool
	}{
		{
			name:     "parallel",
			mode:     ModeParallel,
			expected: true,
		},
		{
			name:     "serial",
			mode:     ModeSerial,
			expected: false,
		},
		{
			name:     "serial modifying",
			mode:     ModeSerialModifying,
			expected: false,
		},
		{
			name:     "unrecognized",
			mode:     Mode(99),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.mode.IsParallel())
		})
	}
}

func TestMode_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     Mode
		expected string
	}{
		{
			name:     "parallel",
			mode:     ModeParallel,
			expected: "parallel",
		},
		{
			name:     "serial",
			mode:     ModeSerial,
			expected: "serial",
		},
		{
			name:     "serial modifying",
			mode:     ModeSerialModifying,
			expected: "serial-modifying",
		},
		{
			name:     "unrecognized",
			mode:     Mode(99),
			expected: "unknown mode: 99",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.mode.String())
		})
	}
}

func TestOrderedCategories_ReturnsCorrectOrder(t *testing.T) {
	t.Parallel()

	require.Len(t, orderedCategories, 7)
	require.Equal(t, config.CategoryObservability, orderedCategories[0])
	require.Equal(t, config.CategoryAuthentication, orderedCategories[1])
	require.Equal(t, config.CategoryAuthorization, orderedCategories[2])
	require.Equal(t, config.CategoryRateLimiting, orderedCategories[3])
	require.Equal(t, config.CategoryValidation, orderedCategories[4])
	require.Equal(t, config.CategoryContent, orderedCategories[5])
	require.Equal(t, config.CategoryAudit, orderedCategories[6])
}
