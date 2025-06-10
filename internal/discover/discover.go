package discover

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Discoverer interface defines methods for package discovery
type Discoverer interface {
	DiscoverPackage(packageName, version string) (DiscoveryResult, error)
}

// DefaultDiscoverer implements the Discoverer interface with the standard implementation
type DefaultDiscoverer struct{}

// DiscoverPackage gets information about a package from PyPI
func (d *DefaultDiscoverer) DiscoverPackage(packageName, version string) (DiscoveryResult, error) {
	reEnvVar := regexp.MustCompile(`(?i)environment variable`)

	info, err := getPyPIInfo(packageName, version)
	if err != nil {
		return DiscoveryResult{}, err
	}

	result := DiscoveryResult{
		PackageName:      packageName,
		Version:          info.Version,
		FoundTools:       parseTools(info.Description),
		FoundStartupArgs: parseArgs(info.Description),
		FoundEnvVars:     reEnvVar.MatchString(info.Description),
	}

	return result, nil
}

// For backward compatibility
func DiscoverPackage(packageName, version string) (DiscoveryResult, error) {
	d := &DefaultDiscoverer{}
	return d.DiscoverPackage(packageName, version)
}

// PyPIResponse is used to decode the top-level JSON from the PyPI API.
type PyPIResponse struct {
	Info PyPIInfo `json:"info"`
}

// PyPIInfo is used to decode the 'info' object within the PyPI API response.
type PyPIInfo struct {
	Description string `json:"description"`
	Version     string `json:"version"`
}

// DiscoveryResult holds the findings from parsing a package's description.
type DiscoveryResult struct {
	PackageName      string
	Version          string
	FoundStartupArgs []string
	FoundEnvVars     bool
	FoundTools       []string
	Error            error
}

// stopWords is a list of common words found in descriptions that are not tool names.
var stopWords = map[string]bool{
	"Input":     true,
	"Inputs":    true,
	"Returns":   true,
	"Required":  true,
	"Optional":  true,
	"Arguments": true,
	"Example":   true,
	"Examples":  true,
	"Response":  true,
	"Prompts":   true,
}

// ValidateTools checks if requested tools exist in the discovered tools.
func ValidateTools(requestedTools []string, discoveredTools []string) []string {
	if len(requestedTools) == 0 {
		return nil
	}

	discoveredToolsMap := make(map[string]bool)
	for _, tool := range discoveredTools {
		discoveredToolsMap[tool] = true
	}

	var missingTools []string
	for _, tool := range requestedTools {
		if !discoveredToolsMap[tool] {
			missingTools = append(missingTools, tool)
		}
	}

	return missingTools
}

// getPyPIInfo fetches the package description.
func getPyPIInfo(packageName string, packageVersion string) (PyPIInfo, error) {
	apiURL := "https://pypi.org/pypi/%s/json"
	packageSlug := packageName
	if packageVersion != "latest" {
		packageSlug = fmt.Sprintf("%s/%s", packageName, packageVersion)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(fmt.Sprintf(apiURL, packageSlug))
	if err != nil {
		return PyPIInfo{}, fmt.Errorf("failed to connect to PyPI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PyPIInfo{}, fmt.Errorf("package '%s' not found on PyPI (HTTP %d)", packageName, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PyPIInfo{}, fmt.Errorf("failed to read API response: %w", err)
	}

	var pypiResp PyPIResponse
	if err := json.Unmarshal(body, &pypiResp); err != nil {
		return PyPIInfo{}, fmt.Errorf("failed to parse PyPI JSON: %w", err)
	}

	if packageVersion != "latest" && pypiResp.Info.Version != packageVersion {
		return PyPIInfo{}, fmt.Errorf(
			"pypi package version mismatch (expected %s) (got %s)",
			packageVersion,
			pypiResp.Info.Version,
		)
	}

	return pypiResp.Info, nil
}

// getSectionText extracts the text between a start heading and the next major heading.
func getSectionText(desc string, startHeadingPattern *regexp.Regexp) string {
	reNextSection := regexp.MustCompile(`(?im)^\s*#*\s*(installation|configuration|customization|debugging|development|license|contributing|prompts|usage|overview)\s*#*\s*$`)

	headingMatch := startHeadingPattern.FindStringIndex(desc)
	if headingMatch == nil {
		return ""
	}

	searchArea := desc[headingMatch[1]:]

	endMatch := reNextSection.FindStringIndex(searchArea)
	if endMatch != nil {
		searchArea = searchArea[:endMatch[0]]
	}
	return searchArea
}

// parseTools extracts potential tool names from the package description
func parseTools(desc string) []string {
	reToolsHeading := regexp.MustCompile(`(?im)^\s*#*\s*(available\s)?tools\s*#*\s*$`)
	toolsSection := getSectionText(desc, reToolsHeading)
	if toolsSection == "" {
		return nil
	}

	uniqueTools := make(map[string]bool)
	// Regex to find the first snake_case word on a line.
	reFirstWord := regexp.MustCompile(`[a-z_][a-z_0-9]*`)
	// Regex to check if a line is indented.
	reIsIndented := regexp.MustCompile(`^\s+`)

	lines := strings.Split(toolsSection, "\n")

	for _, line := range lines {
		// Skip empty lines and indented lines
		if strings.TrimSpace(line) == "" || reIsIndented.MatchString(line) {
			continue
		}

		// Find the first potential tool name on the line
		toolName := reFirstWord.FindString(line)

		// Check if it's a valid candidate
		if toolName != "" && !stopWords[toolName] {
			uniqueTools[toolName] = true
		}
	}

	// Convert map keys to slice
	finalTools := make([]string, 0, len(uniqueTools))
	for tool := range uniqueTools {
		finalTools = append(finalTools, tool)
	}

	sort.Strings(finalTools)
	return finalTools
}

// parseArgs finds --arguments mentioned in code examples
func parseArgs(desc string) []string {
	uniqueArgs := make(map[string]bool)
	// multi-line code blocks.
	reCodeBlock := regexp.MustCompile("(?s)```.*?```")
	// --flags.
	reArgFlag := regexp.MustCompile(`--\w+[\w-]*`)

	// Find all code blocks in the description.
	codeBlocks := reCodeBlock.FindAllString(desc, -1)
	for _, block := range codeBlocks {
		// Only look for args in blocks that contain "uvx".
		if strings.Contains(block, `"uvx"`) || strings.Contains(block, "uvx mcp-server") {
			foundArgs := reArgFlag.FindAllString(block, -1)
			for _, arg := range foundArgs {
				uniqueArgs[arg] = true
			}
		}
	}

	// Also, run the simple line-based check for prose descriptions.
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "argument") {
			foundArgs := reArgFlag.FindAllString(line, -1)
			for _, arg := range foundArgs {
				uniqueArgs[arg] = true
			}
		}
	}

	finalArgs := make([]string, 0, len(uniqueArgs))
	for arg := range uniqueArgs {
		finalArgs = append(finalArgs, arg)
	}

	sort.Strings(finalArgs)
	return finalArgs
}
