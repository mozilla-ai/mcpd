package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/flags"
	"github.com/mozilla-ai/mcpd/internal/perms"
)

// Init creates the base skeleton configuration file for the mcpd project.
func (d *DefaultLoader) Init(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	content := `servers = []`

	if err := os.WriteFile(path, []byte(content), perms.RegularFile); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func (d *DefaultLoader) Load(path string) (Modifier, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("%w: path cannot be empty", ErrConfigLoadFailed)
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: config file cannot be found, run: 'mcpd init'", ErrConfigLoadFailed)
		}
		return nil, fmt.Errorf("%w: failed to stat config file (%s): %w", ErrConfigLoadFailed, path, err)
	}

	var cfg *Config
	_, err = toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to decode config from file (%s): %w",
			ErrConfigLoadFailed,
			flags.DefaultConfigFile,
			err,
		)
	}
	if cfg == nil {
		return nil, fmt.Errorf("%w: config file is empty (%s)", ErrConfigLoadFailed, path)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%w: failed to validate existing config (%s): %w", ErrConfigLoadFailed, path, err)
	}

	// Update the path that loaded this file to track it.
	cfg.configFilePath = path

	return cfg, nil
}

// AddServer attempts to persist a new MCP Server to the configuration file (.mcpd.toml).
func (c *Config) AddServer(entry ServerEntry) error {
	// Add server
	c.Servers = append(c.Servers, entry)

	// Validate servers
	if err := c.validate(); err != nil {
		return err
	}

	// Save
	if err := c.saveConfig(); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	return nil
}

// RemoveServer removes a server entry by name from the configuration file (.mcpd.toml).
func (c *Config) RemoveServer(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Filter out servers matching the given name
	filtered := make([]ServerEntry, 0, len(c.Servers))
	for _, s := range c.Servers {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == len(c.Servers) {
		return fmt.Errorf("server '%s' not found in config", name)
	}

	c.Servers = filtered

	if err := c.validate(); err != nil {
		return err
	}

	if err := c.saveConfig(); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	return nil
}

// ListServers returns a copy of the currently configured server entries.
// This provides read-only access to the internal configuration without exposing direct mutation of the underlying slice.
func (c *Config) ListServers() []ServerEntry {
	return slices.Clone(c.Servers)
}

// SaveConfig saves the current configuration to the config file.
func (c *Config) SaveConfig() error {
	return c.saveConfig()
}

// Plugin retrieves a plugin by category and name.
func (c *Config) Plugin(category Category, name string) (PluginEntry, bool) {
	if c.Plugins == nil {
		return PluginEntry{}, false
	}

	return c.Plugins.plugin(category, name)
}

// UpsertPlugin creates or updates a plugin entry and saves the configuration.
func (c *Config) UpsertPlugin(category Category, entry PluginEntry) (context.UpsertResult, error) {
	if c.Plugins == nil {
		c.Plugins = &PluginConfig{}
	}

	result, err := c.Plugins.upsertPlugin(category, entry)
	if err != nil {
		return result, err
	}

	if result == context.Noop {
		return result, nil
	}

	if err := c.validate(); err != nil {
		return context.Noop, err
	}

	if err := c.saveConfig(); err != nil {
		return context.Noop, fmt.Errorf("failed to save updated config: %w", err)
	}

	return result, nil
}

// DeletePlugin removes a plugin entry and saves the configuration.
func (c *Config) DeletePlugin(category Category, name string) (context.UpsertResult, error) {
	if c.Plugins == nil {
		return context.Noop, fmt.Errorf("no plugins configured")
	}

	result, err := c.Plugins.deletePlugin(category, name)
	if err != nil {
		return result, err
	}

	if result == context.Noop {
		return result, err
	}

	if err := c.validate(); err != nil {
		return context.Noop, err
	}

	if err := c.saveConfig(); err != nil {
		return context.Noop, fmt.Errorf("failed to save updated config: %w", err)
	}

	return result, nil
}

// MovePlugin moves a plugin within or between categories.
// Use MoveOption functions to specify the operation:
//   - WithToCategory: move to a different category
//   - WithBefore/WithAfter: position relative to another plugin
//   - WithPosition: move to absolute position (1-based)
//   - WithForce: overwrite existing plugin in target category
func (c *Config) MovePlugin(category Category, name string, opts ...MoveOption) (context.UpsertResult, error) {
	if c.Plugins == nil {
		return context.Noop, fmt.Errorf("no plugins configured")
	}

	result, err := c.Plugins.movePlugin(category, name, opts...)
	if err != nil {
		return result, err
	}

	if result == context.Noop {
		return result, nil
	}

	if err := c.validate(); err != nil {
		return context.Noop, err
	}

	if err := c.saveConfig(); err != nil {
		return context.Noop, fmt.Errorf("failed to save updated config: %w", err)
	}

	return result, nil
}

// keyFor generates a temporary version of the ServerEntry to be used as a composite key.
// It consists of the name of the server and the package without version information.
func keyFor(entry ServerEntry) serverKey {
	return serverKey{
		Name:    entry.Name,
		Package: stripVersion(entry.Package),
	}
}

// stripVersion removes any version information present at the end of the package.
func stripVersion(pkg string) string {
	// Find the last @ symbol (version separator)
	if idx := strings.LastIndex(pkg, "@"); idx != -1 {
		return pkg[:idx]
	}
	return pkg
}

func stripPrefix(pkg string) string {
	prefixDelim := "::"
	if idx := strings.Index(pkg, prefixDelim); idx != -1 {
		return pkg[idx+len(prefixDelim):]
	}
	return pkg
}

func (c *Config) saveConfig() error {
	if c.configFilePath == "" {
		return fmt.Errorf("config file path not present")
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(c.configFilePath, data, perms.RegularFile)
}

// validate orchestrates validation of configuration structure.
func (c *Config) validate() error {
	if err := c.validateServers(); err != nil {
		return err
	}

	if c.Daemon != nil {
		if err := c.Daemon.Validate(); err != nil {
			return fmt.Errorf("daemon configuration error: %w", err)
		}
	}

	if c.Plugins != nil {
		if err := c.Plugins.Validate(); err != nil {
			return fmt.Errorf("plugin configuration error: %w", err)
		}
	}

	return nil
}

// validateServers checks the server config section to ensure there are no errors.
func (c *Config) validateServers() error {
	if err := c.validateFields(); err != nil {
		return err
	}
	if err := c.validateDistinct(); err != nil {
		return err
	}
	return nil
}

// validateFields ensures that all ServerEntry's in Config have a name and package.
func (c *Config) validateFields() error {
	seen := map[string]struct{}{}

	for _, entry := range c.Servers {
		if _, ok := seen[entry.Name]; ok {
			return fmt.Errorf("duplicate server name '%s'", entry.Name)
		}
		seen[entry.Name] = struct{}{}
		if strings.TrimSpace(entry.Name) == "" {
			return fmt.Errorf("server entry has empty name")
		}
		if strings.TrimSpace(entry.Package) == "" {
			return fmt.Errorf("server entry has empty package")
		}
	}
	return nil
}

// validateDistinct ensures that all ServerEntry's in Config are distinct (no duplicate servers allowed).
func (c *Config) validateDistinct() error {
	seen := map[serverKey]struct{}{}

	for _, entry := range c.Servers {
		k := keyFor(entry)
		if _, exists := seen[k]; exists {
			return fmt.Errorf("duplicate server entry: name: '%s' package: '%s'", k.Name, k.Package)
		}
		seen[k] = struct{}{}
	}
	return nil
}
