//go:build docsgen_nav
// +build docsgen_nav

package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CommandInfo represents information about a command file
type CommandInfo struct {
	Filename string
	Title    string
	Path     string
}

// updateMkDocsNav reads the mkdocs.yaml file, updates the CLI Reference navigation,
// and writes it back to the file while preserving structure and ordering
func updateMkDocsNav(mkdocsPath, commandsDir string) error {
	// Read the existing mkdocs.yaml
	data, err := os.ReadFile(mkdocsPath)
	if err != nil {
		return fmt.Errorf("failed to read mkdocs.yaml: %w", err)
	}

	// Parse as yaml.Node to preserve structure
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to unmarshal mkdocs.yaml: %w", err)
	}

	// Get command files from the commands directory
	commands, err := commandFiles(commandsDir)
	if err != nil {
		return fmt.Errorf("failed to get command files: %w", err)
	}

	// Update the navigation in the YAML node
	if err := updateNavigationNode(&root, commands); err != nil {
		return fmt.Errorf("failed to update navigation: %w", err)
	}

	// Configure encoder to preserve formatting
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(&root); err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	encoder.Close()

	// Write back to file
	if err := os.WriteFile(mkdocsPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write updated mkdocs.yaml: %w", err)
	}

	fmt.Printf("Successfully updated mkdocs.yaml with %d command references\n", len(commands))
	return nil
}

// commandFiles scans the commands directory and returns command information
func commandFiles(commandsDir string) ([]CommandInfo, error) {
	var commands []CommandInfo
	overviewTitle := "Overview"

	err := filepath.WalkDir(commandsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		markdownExt := ".md"
		filename := d.Name()
		extension := filepath.Ext(filename)

		if d.IsDir() || extension != markdownExt {
			return nil
		}

		nameWithoutExt := filename[:len(filename)-len(extension)]

		// Determine title and relative path
		var title string
		relativePath := filepath.Join("commands", filename)

		if nameWithoutExt == "mcpd" {
			// Root command gets "Overview" title
			title = overviewTitle
		} else {
			// For subcommands, extract the command part after "mcpd_"
			if strings.HasPrefix(nameWithoutExt, "mcpd_") {
				commandPart := strings.TrimPrefix(nameWithoutExt, "mcpd_")
				// Replace underscores with spaces and title case
				title = strings.ReplaceAll(commandPart, "_", " ")
				title = strings.TrimSpace(strings.ToLower(title))
			} else {
				// Fallback to filename without extension
				title = nameWithoutExt
			}
		}

		commands = append(commands, CommandInfo{
			Filename: filename,
			Title:    title,
			Path:     relativePath,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort commands: Overview first, then alphabetically by title
	sort.Slice(commands, func(i, j int) bool {
		if commands[i].Title == overviewTitle {
			return true
		}
		if commands[j].Title == overviewTitle {
			return false
		}
		return commands[i].Title < commands[j].Title
	})

	return commands, nil
}

// updateNavigationNode finds and updates the "CLI Reference" section in the YAML node tree
func updateNavigationNode(root *yaml.Node, commands []CommandInfo) error {
	// Find the document node
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("invalid YAML structure")
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping at root level")
	}

	// Find the nav section
	for i := 0; i < len(doc.Content); i += 2 {
		key := doc.Content[i]
		value := doc.Content[i+1]

		if key.Value == "nav" && value.Kind == yaml.SequenceNode {
			// Found nav section, now find CLI Reference
			if err := updateCLIReference(value, commands); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// updateCLIReference finds and updates the CLI Reference section within the nav
func updateCLIReference(navNode *yaml.Node, commands []CommandInfo) error {
	cliReferenceTitle := "CLI Reference" // NOTE: This needs to match the name of the section in MkDocs
	for _, item := range navNode.Content {
		if item.Kind == yaml.MappingNode {
			// Look for CLI Reference key
			for i := 0; i < len(item.Content); i += 2 {
				key := item.Content[i]
				value := item.Content[i+1]

				if key.Value == cliReferenceTitle && value.Kind == yaml.SequenceNode {
					// Found CLI Reference, update its content
					updateCLIReferenceContent(value, commands)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("CLI Reference section not found")
}

// updateCLIReferenceContent replaces the CLI Reference content with generated commands
func updateCLIReferenceContent(cliRefNode *yaml.Node, commands []CommandInfo) {
	var newContent []*yaml.Node

	// Add all commands (Overview will be first due to sorting)
	for _, cmd := range commands {
		// Create a new mapping node for each command
		mappingNode := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: cmd.Title},
				{Kind: yaml.ScalarNode, Value: cmd.Path},
			},
		}
		newContent = append(newContent, mappingNode)
	}

	// Replace the content
	cliRefNode.Content = newContent
}

func main() {
	mkdocsPath := "mkdocs.yaml"
	commandsDir := "./docs/commands/"

	// Check if mkdocs.yaml exists
	if _, err := os.Stat(mkdocsPath); os.IsNotExist(err) {
		log.Fatalf("MkDocs config file '%s' not found: %v", mkdocsPath, err)
	}

	// Check if commands docs directory exists
	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		log.Fatalf("Commands docs directory '%s' does not exist", commandsDir)
	}

	// Update the navigation
	if err := updateMkDocsNav(mkdocsPath, commandsDir); err != nil {
		log.Fatalf("Failed to update MkDocs navigation: %v", err)
	}
}
