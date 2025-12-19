package plugins

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// ValidateCmd represents the command for validating plugin configuration.
// Use NewValidateCmd to create instances of ValidateCmd.
type ValidateCmd struct {
	*internalcmd.BaseCmd

	// category restricts validation to a specific category.
	category config.Category

	// cfgLoader loads the configuration file.
	cfgLoader config.Loader

	// checkBinaries enables filesystem checks for plugin binaries.
	checkBinaries bool

	// verbose enables detailed output.
	verbose bool
}

// NewValidateCmd creates a new validate command for plugin configuration.
func NewValidateCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ValidateCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate plugin configuration",
		Long: `Validate plugin configuration structure and optionally check plugin binaries.

By default, validates config structure only (portable across environments).
Use --check-binaries to also verify binaries exist (environment-specific).`,
		RunE: c.run,
		Args: cobra.NoArgs,
		Example: `  # Validate config structure (portable)
  mcpd config plugins validate

  # Validate specific category
  mcpd config plugins validate --category=authentication

  # Check binaries exist (environment-specific)
  mcpd config plugins validate --check-binaries

  # Verbose output
  mcpd config plugins validate --verbose`,
	}

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		flagCategory,
		fmt.Sprintf("Validate only this category (one of: %s)", allowedCategories.String()),
	)

	cobraCmd.Flags().BoolVar(
		&c.checkBinaries,
		"check-binaries",
		false,
		"Check plugin binaries exist (environment-specific, NOT portable)",
	)

	cobraCmd.Flags().BoolVar(
		&c.verbose,
		"verbose",
		false,
		"Show detailed validation info",
	)

	return cobraCmd, nil
}

// run executes the validation and prints results.
func (c *ValidateCmd) run(cobraCmd *cobra.Command, _ []string) error {
	loader := c.cfgLoader
	var err error

	// Wrap with validating loader if binary checks requested.
	if c.checkBinaries {
		if loader, err = config.NewValidatingLoader(loader, config.ValidatePluginBinaries); err != nil {
			return fmt.Errorf("failed to create validating loader: %w", err)
		}
	}

	cfg, err := c.LoadConfig(loader)
	if err != nil {
		return err
	}

	// Check if plugins are configured.
	if cfg.Plugins == nil {
		_, _ = fmt.Fprintln(cobraCmd.OutOrStdout(), "No plugin configuration found")
		return nil
	}

	// Build and run validator.
	v := &pluginValidator{
		cfg:           cfg.Plugins,
		category:      c.category,
		checkBinaries: c.checkBinaries,
		verbose:       c.verbose,
	}

	result := v.validate()
	result.print(cobraCmd.OutOrStdout())

	if result.totalIssues > 0 {
		return fmt.Errorf("validation failed with %d error(s)", result.totalIssues)
	}

	return nil
}

// pluginValidator performs plugin validation.
type pluginValidator struct {
	cfg           *config.PluginConfig
	category      config.Category
	checkBinaries bool
	verbose       bool
}

// validationResult holds the results of plugin validation.
type validationResult struct {
	categories    []categoryResult
	configErrors  []string
	totalPlugins  int
	totalIssues   int
	checkBinaries bool
	verbose       bool
}

// categoryResult holds validation results for a category.
type categoryResult struct {
	name    config.Category
	plugins []pluginValidation
}

// pluginValidation holds validation results for a single plugin.
type pluginValidation struct {
	name    string
	errors  []string
	details []string
}

// validate performs validation and returns results.
func (v *pluginValidator) validate() *validationResult {
	result := &validationResult{
		checkBinaries: v.checkBinaries,
		verbose:       v.verbose,
	}

	// Validate plugin directory if binary checks requested.
	if v.checkBinaries {
		if strings.TrimSpace(v.cfg.Dir) == "" {
			result.configErrors = append(
				result.configErrors,
				"Plugin directory not configured (required for --check-binaries)",
			)
			result.totalIssues++
		} else if _, err := os.Stat(v.cfg.Dir); os.IsNotExist(err) {
			result.configErrors = append(
				result.configErrors,
				fmt.Sprintf("Plugin directory does not exist: %s", v.cfg.Dir),
			)
			result.totalIssues++
		}
	}

	// Get categories to validate.
	allCategories := v.cfg.AllCategories()

	// Filter to specific category if requested.
	if v.category != "" {
		filtered := make(map[config.Category][]config.PluginEntry)
		if plugins, ok := allCategories[v.category]; ok {
			filtered[v.category] = plugins
		}
		allCategories = filtered
	}

	// Validate each category in order.
	for _, cat := range config.OrderedCategories() {
		plugins, ok := allCategories[cat]
		if !ok {
			continue
		}

		catResult := categoryResult{name: cat}

		for _, entry := range plugins {
			pv := v.validatePlugin(entry)
			catResult.plugins = append(catResult.plugins, pv)
			result.totalPlugins++
			result.totalIssues += len(pv.errors)
		}

		result.categories = append(result.categories, catResult)
	}

	return result
}

// validatePlugin validates a single plugin entry.
func (v *pluginValidator) validatePlugin(entry config.PluginEntry) pluginValidation {
	pv := pluginValidation{name: entry.Name}

	// Validate config structure.
	if err := entry.Validate(); err != nil {
		pv.errors = append(pv.errors, err.Error())
	} else if v.verbose {
		pv.details = append(pv.details, "Config structure valid")
		pv.details = append(pv.details, fmt.Sprintf("Flows: %s", formatFlows(entry.Flows)))
		if entry.Required != nil && *entry.Required {
			pv.details = append(pv.details, "Required: true")
		}
	}

	// Validate binary existence if requested.
	if v.checkBinaries && strings.TrimSpace(v.cfg.Dir) != "" {
		binaryPath := filepath.Join(v.cfg.Dir, entry.Name)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			pv.errors = append(pv.errors, fmt.Sprintf("Binary not found: %s", binaryPath))
		} else if v.verbose {
			pv.details = append(pv.details, fmt.Sprintf("Binary exists: %s", binaryPath))
		}
	}

	return pv
}

// formatFlows formats flow names for display.
func formatFlows(flowsList []config.Flow) string {
	names := make([]string, len(flowsList))
	for i, f := range flowsList {
		names[i] = string(f)
	}
	return strings.Join(names, ", ")
}

// print outputs the validation results.
func (r *validationResult) print(w io.Writer) {
	// Print header.
	_, _ = fmt.Fprintln(w, "Validating plugin configuration...")
	if r.checkBinaries {
		_, _ = fmt.Fprintln(w, "Checking plugin binaries (environment-specific)...")
	}
	_, _ = fmt.Fprintln(w)

	// Print config-level errors.
	if len(r.configErrors) > 0 {
		_, _ = fmt.Fprintln(w, "Configuration:")
		for _, err := range r.configErrors {
			_, _ = fmt.Fprintf(w, "  \u2717 %s\n", err)
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print category results.
	for _, cat := range r.categories {
		_, _ = fmt.Fprintf(w, "Category '%s':\n", cat.name)

		for _, p := range cat.plugins {
			_, _ = fmt.Fprintf(w, "  Plugin '%s':\n", p.name)

			if len(p.errors) == 0 {
				_, _ = fmt.Fprintln(w, "    \u2713 Valid")
			}

			for _, err := range p.errors {
				_, _ = fmt.Fprintf(w, "    \u2717 %s\n", err)
			}

			if r.verbose {
				for _, detail := range p.details {
					_, _ = fmt.Fprintf(w, "    \u2713 %s\n", detail)
				}
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print summary.
	if r.totalIssues > 0 {
		_, _ = fmt.Fprintf(w, "\u2717 Validation failed with %d error(s)\n\n", r.totalIssues)
		if r.checkBinaries {
			_, _ = fmt.Fprintln(w, "NOTE: Binary checks are environment-specific. This config may work in")
			_, _ = fmt.Fprintln(w, "other environments where paths differ.")
			_, _ = fmt.Fprintln(w)
		}
	} else {
		_, _ = fmt.Fprintln(w, "\u2713 All plugins validated successfully!")
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w, "Summary:")
	_, _ = fmt.Fprintf(w, "  Categories: %d\n", len(r.categories))
	_, _ = fmt.Fprintf(w, "  Plugins: %d\n", r.totalPlugins)
	if r.checkBinaries {
		_, _ = fmt.Fprintln(w, "  Binary checks: enabled")
	}
	_, _ = fmt.Fprintf(w, "  Issues: %d\n", r.totalIssues)
}
