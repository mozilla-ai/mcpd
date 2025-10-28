package cmd

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestBaseCmd_RequireTogether(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flagNames   []string
		setFlags    []string
		expectError bool
	}{
		{
			name:        "all flags provided",
			flagNames:   []string{"flag1", "flag2"},
			setFlags:    []string{"flag1", "flag2"},
			expectError: false,
		},
		{
			name:        "no flags provided",
			flagNames:   []string{"flag1", "flag2"},
			setFlags:    []string{},
			expectError: false,
		},
		{
			name:        "only first flag provided",
			flagNames:   []string{"flag1", "flag2"},
			setFlags:    []string{"flag1"},
			expectError: true,
		},
		{
			name:        "only second flag provided",
			flagNames:   []string{"flag1", "flag2"},
			setFlags:    []string{"flag2"},
			expectError: true,
		},
		{
			name:        "three flags all provided",
			flagNames:   []string{"flag1", "flag2", "flag3"},
			setFlags:    []string{"flag1", "flag2", "flag3"},
			expectError: false,
		},
		{
			name:        "three flags none provided",
			flagNames:   []string{"flag1", "flag2", "flag3"},
			setFlags:    []string{},
			expectError: false,
		},
		{
			name:        "three flags only one provided",
			flagNames:   []string{"flag1", "flag2", "flag3"},
			setFlags:    []string{"flag1"},
			expectError: true,
		},
		{
			name:        "three flags only two provided",
			flagNames:   []string{"flag1", "flag2", "flag3"},
			setFlags:    []string{"flag1", "flag3"},
			expectError: true,
		},
		{
			name:        "three flags only two provided - test sorting",
			flagNames:   []string{"flag1", "flag2", "flag3"},
			setFlags:    []string{"flag3", "flag1"},
			expectError: true,
		},
		{
			name:        "single flag not provided",
			flagNames:   []string{"flag1"},
			setFlags:    []string{},
			expectError: false,
		},
		{
			name:        "single flag provided",
			flagNames:   []string{"flag1"},
			setFlags:    []string{"flag1"},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{
				Use: "test",
			}

			for _, flagName := range tc.flagNames {
				cmd.Flags().String(flagName, "", "test flag")
			}

			for _, flagName := range tc.setFlags {
				err := cmd.Flags().Set(flagName, "value")
				require.NoError(t, err)
			}

			baseCmd := &BaseCmd{}
			err := baseCmd.RequireTogether(cmd, tc.flagNames...)

			if tc.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, "must be provided together or not at all")
				names := slices.Clone(tc.flagNames)
				slices.Sort(names)
				sortedNames := strings.Join(names, ", ")
				require.ErrorContains(t, err, fmt.Sprintf("(%s)", sortedNames))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
