package context

import (
	"fmt"
	"strings"
)

// DockerConfig represents Docker-specific configuration for an MCP server
type DockerConfig struct {
	// Network & Connectivity
	Network string   `toml:"network,omitempty"` // --network flag (default: host)
	Ports   []string `toml:"ports,omitempty"`   // -p/--publish flags

	// Storage
	Volumes  []string `toml:"volumes,omitempty"`  // -v/--volume flags
	Tmpfs    []string `toml:"tmpfs,omitempty"`    // --tmpfs flags
	ReadOnly bool     `toml:"read_only,omitempty"` // --read-only flag

	// Resources
	Memory     string `toml:"memory,omitempty"`      // --memory flag
	MemorySwap string `toml:"memory_swap,omitempty"` // --memory-swap flag
	CPUs       string `toml:"cpus,omitempty"`        // --cpus flag

	// Security & Environment
	User    string `toml:"user,omitempty"`     // --user flag
	WorkDir string `toml:"workdir,omitempty"`  // --workdir flag
	EnvFile string `toml:"env_file,omitempty"` // --env-file flag

	// Lifecycle
	Remove *bool `toml:"remove,omitempty"` // --rm flag (default: true)
}

// ParseDockerConfig parses docker.* prefixed environment variables into DockerConfig
func ParseDockerConfig(envMap map[string]string) (*DockerConfig, map[string]string) {
	if envMap == nil {
		return nil, nil
	}

	config := &DockerConfig{}
	regularEnv := make(map[string]string)

	// Track volumes and ports by suffix for multiple entries
	volumes := make(map[string]string)
	ports := make(map[string]string)

	for key, value := range envMap {
		if !strings.HasPrefix(strings.ToLower(key), "docker.") {
			regularEnv[key] = value
			continue
		}

		// Remove docker. prefix and convert to lowercase for matching
		dockerKey := strings.ToLower(strings.TrimPrefix(strings.ToLower(key), "docker."))

		switch {
		// Volume handling (docker.volume or docker.volume.*)
		case dockerKey == "volume" || strings.HasPrefix(dockerKey, "volume."):
			suffix := "default"
			if strings.HasPrefix(dockerKey, "volume.") {
				suffix = strings.TrimPrefix(dockerKey, "volume.")
			}
			volumes[suffix] = value

		// Port handling (docker.port or docker.port.*)
		case dockerKey == "port" || strings.HasPrefix(dockerKey, "port."):
			suffix := "default"
			if strings.HasPrefix(dockerKey, "port.") {
				suffix = strings.TrimPrefix(dockerKey, "port.")
			}
			ports[suffix] = value

		// Network configuration
		case dockerKey == "network":
			config.Network = value

		// Tmpfs handling (docker.tmpfs or docker.tmpfs.*)
		case dockerKey == "tmpfs" || strings.HasPrefix(dockerKey, "tmpfs."):
			config.Tmpfs = append(config.Tmpfs, value)

		// Read-only filesystem
		case dockerKey == "read-only" || dockerKey == "readonly":
			if strings.ToLower(value) == "true" {
				config.ReadOnly = true
			}

		// Resource limits
		case dockerKey == "memory":
			config.Memory = value
		case dockerKey == "memory-swap" || dockerKey == "memoryswap":
			config.MemorySwap = value
		case dockerKey == "cpus":
			config.CPUs = value

		// Security & Environment
		case dockerKey == "user":
			config.User = value
		case dockerKey == "workdir" || dockerKey == "working-dir":
			config.WorkDir = value
		case dockerKey == "env-file" || dockerKey == "envfile":
			config.EnvFile = value

		// Lifecycle
		case dockerKey == "remove" || dockerKey == "rm":
			remove := strings.ToLower(value) == "true"
			config.Remove = &remove

		default:
			// Unknown docker.* key - treat as regular env var
			regularEnv[key] = value
		}
	}

	// Convert volume and port maps to slices
	for _, v := range volumes {
		config.Volumes = append(config.Volumes, v)
	}
	for _, p := range ports {
		config.Ports = append(config.Ports, p)
	}

	// Return nil if no Docker config was actually set
	if isEmptyDockerConfig(config) {
		return nil, envMap
	}

	return config, regularEnv
}

// BuildDockerArgs builds Docker command arguments from the configuration
func (c *DockerConfig) BuildDockerArgs() []string {
	if c == nil {
		// Return default args if no config
		return []string{"run", "-i", "--rm", "--network", "host"}
	}

	args := []string{"run", "-i"}

	// Handle --rm flag (default true if not specified)
	if c.Remove == nil || *c.Remove {
		args = append(args, "--rm")
	}

	// Network configuration (default to host if not specified)
	network := c.Network
	if network == "" {
		network = "host"
	}
	args = append(args, "--network", network)

	// Add volumes
	for _, volume := range c.Volumes {
		args = append(args, "-v", volume)
	}

	// Add ports (only makes sense with non-host network)
	if network != "host" {
		for _, port := range c.Ports {
			args = append(args, "-p", port)
		}
	}

	// Add tmpfs mounts
	for _, tmpfs := range c.Tmpfs {
		args = append(args, "--tmpfs", tmpfs)
	}

	// Add read-only flag
	if c.ReadOnly {
		args = append(args, "--read-only")
	}

	// Resource limits
	if c.Memory != "" {
		args = append(args, "--memory", c.Memory)
	}
	if c.MemorySwap != "" {
		args = append(args, "--memory-swap", c.MemorySwap)
	}
	if c.CPUs != "" {
		args = append(args, "--cpus", c.CPUs)
	}

	// Security & Environment
	if c.User != "" {
		args = append(args, "--user", c.User)
	}
	if c.WorkDir != "" {
		args = append(args, "--workdir", c.WorkDir)
	}
	if c.EnvFile != "" {
		args = append(args, "--env-file", c.EnvFile)
	}

	return args
}

// Validate checks if the DockerConfig has valid values
func (c *DockerConfig) Validate() error {
	if c == nil {
		return nil
	}

	// Validate volumes don't mount host root
	for _, volume := range c.Volumes {
		parts := strings.SplitN(volume, ":", 2)
		if len(parts) > 0 {
			hostPath := strings.TrimSpace(parts[0])
			// Check for dangerous mounts
			if hostPath == "/" || hostPath == "/etc" || hostPath == "/usr" || hostPath == "/bin" || hostPath == "/sbin" {
				return fmt.Errorf("dangerous volume mount detected: %s", hostPath)
			}
		}
	}

	// Validate network is a known type or custom name
	if c.Network != "" {
		validNetworks := map[string]bool{
			"bridge": true,
			"host":   true,
			"none":   true,
		}
		// Allow custom network names (they might be user-created networks)
		if !validNetworks[c.Network] && !isValidDockerName(c.Network) {
			return fmt.Errorf("invalid network name: %s", c.Network)
		}
	}

	// Validate memory format (e.g., 512m, 1g)
	if c.Memory != "" && !isValidMemoryString(c.Memory) {
		return fmt.Errorf("invalid memory format: %s (expected format like 512m or 1g)", c.Memory)
	}
	if c.MemorySwap != "" && !isValidMemoryString(c.MemorySwap) {
		return fmt.Errorf("invalid memory-swap format: %s (expected format like 512m or 1g)", c.MemorySwap)
	}

	// Validate CPUs format (e.g., 0.5, 1, 2)
	if c.CPUs != "" && !isValidCPUString(c.CPUs) {
		return fmt.Errorf("invalid cpus format: %s (expected decimal number like 0.5 or 2)", c.CPUs)
	}

	return nil
}


// isValidDockerName validates Docker resource names (networks, volumes, etc.)
func isValidDockerName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	// Docker names can contain lowercase letters, digits, underscores, periods and hyphens
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-') {
			return false
		}
	}
	return true
}

// isValidMemoryString validates Docker memory format (e.g., 512m, 1g)
func isValidMemoryString(mem string) bool {
	if mem == "" {
		return false
	}
	// Simple validation - should end with b, k, m, or g
	lastChar := strings.ToLower(mem[len(mem)-1:])
	if lastChar != "b" && lastChar != "k" && lastChar != "m" && lastChar != "g" {
		return false
	}
	// Check the numeric part
	numPart := mem[:len(mem)-1]
	if numPart == "" {
		return false
	}
	// Simple numeric validation (allowing digits only)
	for _, r := range numPart {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isValidCPUString validates Docker CPU format (e.g., 0.5, 1, 2)
func isValidCPUString(cpu string) bool {
	if cpu == "" {
		return false
	}
	dotCount := 0
	for i, r := range cpu {
		if r == '.' {
			dotCount++
			if dotCount > 1 || i == 0 || i == len(cpu)-1 {
				return false
			}
		} else if r < '0' || r > '9' {
			return false
		}
	}
	return true
}