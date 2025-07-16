package cmd

import (
	"fmt"
	"slices"
	"strings"
)

// ExportFormat represents an enum for the supported formats we can export to.
type ExportFormat string

// ExportFormats is a wrapper which allows 'helper' receivers to be declared,
// such as String().
type ExportFormats []ExportFormat

const (
	// FormatDotEnv for contract files that should be dotenv (.env) format.
	FormatDotEnv ExportFormat = "dotenv"

	// FormatGitHubActions for contract files that should be GitHub Actions format.
	FormatGitHubActions ExportFormat = "github"

	// FormatKubernetesSecret for contract files that should be Kubernetes Secret format.
	FormatKubernetesSecret ExportFormat = "k8s"
)

// AllowedExportFormats returns the allowed formats for the export command.
func AllowedExportFormats() ExportFormats {
	formats := ExportFormats{
		FormatDotEnv,
		// TODO: Uncomment to enable, as we add support for each.
		// FormatGitHubActions,
		// FormatKubernetesSecret,
	}

	slices.Sort(formats)

	return formats
}

// String implements fmt.Stringer for a collection of export formats,
// converting them to a comma separated string.
func (f *ExportFormats) String() string {
	efs := *f
	out := make([]string, len(efs))
	for i := range efs {
		out[i] = efs[i].String()
	}
	return strings.Join(out, ", ")
}

// String implements fmt.Stringer for an export format.
// This is also required by Cobra as part of implementing flag.Value.
func (f *ExportFormat) String() string {
	return strings.ToLower(string(*f))
}

// Set is used by Cobra to set the export format value from a string.
// This is also required by Cobra as part of implementing flag.Value.
func (f *ExportFormat) Set(v string) error {
	allowed := AllowedExportFormats()

	for _, a := range allowed {
		if string(a) == v {
			*f = ExportFormat(v)
			return nil
		}
	}

	return fmt.Errorf("invalid format '%s', must be one of %v", v, allowed.String())
}

// Type is used by Cobra to get the 'type' of an export format for display purposes.
// This is also required by Cobra as part of implementing flag.Value.
func (f *ExportFormat) Type() string {
	return "format"
}
