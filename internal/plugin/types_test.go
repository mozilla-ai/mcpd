package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestCategoryProperties_AllCategoriesDefined(t *testing.T) {
	t.Parallel()

	// Ensure all 7 categories are defined.
	require.Len(t, categoryProps, 7, "expected exactly 7 plugin categories")

	// Verify each category exists with correct properties.
	authProps, err := PropertiesForCategory(config.CategoryAuthentication)
	require.NoError(t, err)
	require.False(t, authProps.Parallel)
	require.False(t, authProps.IgnoreOptionalRejection)
	require.False(t, authProps.CanModify)

	authzProps, err := PropertiesForCategory(config.CategoryAuthorization)
	require.NoError(t, err)
	require.False(t, authzProps.Parallel)
	require.False(t, authzProps.IgnoreOptionalRejection)
	require.False(t, authzProps.CanModify)

	rateLimitProps, err := PropertiesForCategory(config.CategoryRateLimiting)
	require.NoError(t, err)
	require.False(t, rateLimitProps.Parallel)
	require.False(t, rateLimitProps.IgnoreOptionalRejection)
	require.False(t, rateLimitProps.CanModify)

	validationProps, err := PropertiesForCategory(config.CategoryValidation)
	require.NoError(t, err)
	require.False(t, validationProps.Parallel)
	require.False(t, validationProps.IgnoreOptionalRejection)
	require.False(t, validationProps.CanModify)

	contentProps, err := PropertiesForCategory(config.CategoryContent)
	require.NoError(t, err)
	require.False(t, contentProps.Parallel)
	require.False(t, contentProps.IgnoreOptionalRejection)
	require.True(t, contentProps.CanModify)

	observabilityProps, err := PropertiesForCategory(config.CategoryObservability)
	require.NoError(t, err)
	require.True(t, observabilityProps.Parallel)
	require.True(t, observabilityProps.IgnoreOptionalRejection)
	require.False(t, observabilityProps.CanModify)

	auditProps, err := PropertiesForCategory(config.CategoryAudit)
	require.NoError(t, err)
	require.False(t, auditProps.Parallel)
	require.False(t, auditProps.IgnoreOptionalRejection)
	require.False(t, auditProps.CanModify)
}

func TestCategoryProperties_ParallelCannotModify(t *testing.T) {
	t.Parallel()

	for category, props := range categoryProps {
		if props.Parallel {
			require.False(t, props.CanModify,
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
