package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDockerConfig(t *testing.T) {
	tests := []struct {
		name           string
		envMap         map[string]string
		expectedDocker *DockerConfig
		expectedEnv    map[string]string
	}{
		{
			name: "parse volume configuration",
			envMap: map[string]string{
				"docker.volume": "/host/path:/container/path:ro",
				"API_KEY":       "secret123",
			},
			expectedDocker: &DockerConfig{
				Volumes: []string{"/host/path:/container/path:ro"},
			},
			expectedEnv: map[string]string{
				"API_KEY": "secret123",
			},
		},
		{
			name: "parse multiple volumes with suffixes",
			envMap: map[string]string{
				"docker.volume.data":   "/data:/var/lib/data",
				"docker.volume.config": "/config:/etc/config:ro",
			},
			expectedDocker: &DockerConfig{
				Volumes: []string{"/config:/etc/config:ro", "/data:/var/lib/data"},
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse network configuration",
			envMap: map[string]string{
				"docker.network": "bridge",
			},
			expectedDocker: &DockerConfig{
				Network: "bridge",
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse port mappings",
			envMap: map[string]string{
				"docker.port":         "8080:8080",
				"docker.port.metrics": "9090:9090",
			},
			expectedDocker: &DockerConfig{
				Ports: []string{"8080:8080", "9090:9090"},
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse resource limits",
			envMap: map[string]string{
				"docker.memory":      "512m",
				"docker.memory-swap": "1g",
				"docker.cpus":        "0.5",
			},
			expectedDocker: &DockerConfig{
				Memory:     "512m",
				MemorySwap: "1g",
				CPUs:       "0.5",
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse security settings",
			envMap: map[string]string{
				"docker.user":      "1000:1000",
				"docker.workdir":   "/app",
				"docker.read-only": "true",
			},
			expectedDocker: &DockerConfig{
				User:     "1000:1000",
				WorkDir:  "/app",
				ReadOnly: true,
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse tmpfs configuration",
			envMap: map[string]string{
				"docker.tmpfs":      "/tmp:size=100m",
				"docker.tmpfs.data": "/var/tmp:size=200m",
			},
			expectedDocker: &DockerConfig{
				Tmpfs: []string{"/tmp:size=100m", "/var/tmp:size=200m"},
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse env-file configuration",
			envMap: map[string]string{
				"docker.env-file": ".env.production",
			},
			expectedDocker: &DockerConfig{
				EnvFile: ".env.production",
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "parse remove flag",
			envMap: map[string]string{
				"docker.remove": "false",
			},
			expectedDocker: &DockerConfig{
				Remove: boolPtr(false),
			},
			expectedEnv: map[string]string{},
		},
		{
			name: "mixed docker and regular env vars",
			envMap: map[string]string{
				"docker.volume":  "/data:/data",
				"docker.network": "host",
				"API_KEY":        "secret",
				"DATABASE_URL":   "postgres://localhost",
			},
			expectedDocker: &DockerConfig{
				Network: "host",
				Volumes: []string{"/data:/data"},
			},
			expectedEnv: map[string]string{
				"API_KEY":      "secret",
				"DATABASE_URL": "postgres://localhost",
			},
		},
		{
			name: "no docker config",
			envMap: map[string]string{
				"API_KEY": "secret",
				"PORT":    "8080",
			},
			expectedDocker: nil,
			expectedEnv: map[string]string{
				"API_KEY": "secret",
				"PORT":    "8080",
			},
		},
		{
			name:           "nil envMap",
			envMap:         nil,
			expectedDocker: nil,
			expectedEnv:    nil,
		},
		{
			name: "case insensitive docker keys",
			envMap: map[string]string{
				"DOCKER.VOLUME":  "/data:/data",
				"Docker.Network": "bridge",
				"docker.PORT":    "8080:8080",
			},
			expectedDocker: &DockerConfig{
				Network: "bridge",
				Volumes: []string{"/data:/data"},
				Ports:   []string{"8080:8080"},
			},
			expectedEnv: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerConfig, regularEnv := ParseDockerConfig(tt.envMap)

			if tt.expectedDocker == nil {
				assert.Nil(t, dockerConfig)
			} else {
				require.NotNil(t, dockerConfig)
				assert.Equal(t, tt.expectedDocker.Network, dockerConfig.Network)
				assert.ElementsMatch(t, tt.expectedDocker.Volumes, dockerConfig.Volumes)
				assert.ElementsMatch(t, tt.expectedDocker.Ports, dockerConfig.Ports)
				assert.ElementsMatch(t, tt.expectedDocker.Tmpfs, dockerConfig.Tmpfs)
				assert.Equal(t, tt.expectedDocker.Memory, dockerConfig.Memory)
				assert.Equal(t, tt.expectedDocker.MemorySwap, dockerConfig.MemorySwap)
				assert.Equal(t, tt.expectedDocker.CPUs, dockerConfig.CPUs)
				assert.Equal(t, tt.expectedDocker.User, dockerConfig.User)
				assert.Equal(t, tt.expectedDocker.WorkDir, dockerConfig.WorkDir)
				assert.Equal(t, tt.expectedDocker.ReadOnly, dockerConfig.ReadOnly)
				assert.Equal(t, tt.expectedDocker.EnvFile, dockerConfig.EnvFile)
				if tt.expectedDocker.Remove != nil {
					require.NotNil(t, dockerConfig.Remove)
					assert.Equal(t, *tt.expectedDocker.Remove, *dockerConfig.Remove)
				} else {
					assert.Nil(t, dockerConfig.Remove)
				}
			}

			assert.Equal(t, tt.expectedEnv, regularEnv)
		})
	}
}

func TestDockerConfigBuildDockerArgs(t *testing.T) {
	tests := []struct {
		name         string
		config       *DockerConfig
		expectedArgs []string
	}{
		{
			name:         "nil config returns defaults",
			config:       nil,
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host"},
		},
		{
			name:         "empty config returns defaults",
			config:       &DockerConfig{},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host"},
		},
		{
			name: "custom network",
			config: &DockerConfig{
				Network: "bridge",
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "bridge"},
		},
		{
			name: "volumes configuration",
			config: &DockerConfig{
				Volumes: []string{"/host:/container:ro", "/data:/data"},
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host", "-v", "/host:/container:ro", "-v", "/data:/data"},
		},
		{
			name: "ports with bridge network",
			config: &DockerConfig{
				Network: "bridge",
				Ports:   []string{"8080:8080", "9090:9090"},
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "bridge", "-p", "8080:8080", "-p", "9090:9090"},
		},
		{
			name: "ports ignored with host network",
			config: &DockerConfig{
				Network: "host",
				Ports:   []string{"8080:8080"},
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host"},
		},
		{
			name: "resource limits",
			config: &DockerConfig{
				Memory:     "512m",
				MemorySwap: "1g",
				CPUs:       "0.5",
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host", "--memory", "512m", "--memory-swap", "1g", "--cpus", "0.5"},
		},
		{
			name: "security settings",
			config: &DockerConfig{
				User:     "1000:1000",
				WorkDir:  "/app",
				ReadOnly: true,
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host", "--read-only", "--user", "1000:1000", "--workdir", "/app"},
		},
		{
			name: "tmpfs mounts",
			config: &DockerConfig{
				Tmpfs: []string{"/tmp:size=100m", "/var/tmp"},
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host", "--tmpfs", "/tmp:size=100m", "--tmpfs", "/var/tmp"},
		},
		{
			name: "env file",
			config: &DockerConfig{
				EnvFile: ".env.production",
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host", "--env-file", ".env.production"},
		},
		{
			name: "remove flag false",
			config: &DockerConfig{
				Remove: boolPtr(false),
			},
			expectedArgs: []string{"run", "-i", "--network", "host"},
		},
		{
			name: "remove flag true",
			config: &DockerConfig{
				Remove: boolPtr(true),
			},
			expectedArgs: []string{"run", "-i", "--rm", "--network", "host"},
		},
		{
			name: "complete configuration",
			config: &DockerConfig{
				Network:    "bridge",
				Volumes:    []string{"/data:/data"},
				Ports:      []string{"8080:8080"},
				Tmpfs:      []string{"/tmp"},
				ReadOnly:   true,
				Memory:     "1g",
				MemorySwap: "2g",
				CPUs:       "2",
				User:       "1000:1000",
				WorkDir:    "/app",
				EnvFile:    ".env",
				Remove:     boolPtr(true),
			},
			expectedArgs: []string{
				"run", "-i", "--rm", "--network", "bridge",
				"-v", "/data:/data",
				"-p", "8080:8080",
				"--tmpfs", "/tmp",
				"--read-only",
				"--memory", "1g",
				"--memory-swap", "2g",
				"--cpus", "2",
				"--user", "1000:1000",
				"--workdir", "/app",
				"--env-file", ".env",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.config.BuildDockerArgs()
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestDockerConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *DockerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &DockerConfig{},
			expectError: false,
		},
		{
			name: "valid configuration",
			config: &DockerConfig{
				Network: "bridge",
				Volumes: []string{"/data:/data", "/logs:/var/log"},
				Memory:  "512m",
				CPUs:    "0.5",
			},
			expectError: false,
		},
		{
			name: "dangerous volume mount root",
			config: &DockerConfig{
				Volumes: []string{"/:/host"},
			},
			expectError: true,
			errorMsg:    "dangerous volume mount detected: /",
		},
		{
			name: "dangerous volume mount etc",
			config: &DockerConfig{
				Volumes: []string{"/etc:/etc"},
			},
			expectError: true,
			errorMsg:    "dangerous volume mount detected: /etc",
		},
		{
			name: "invalid memory format",
			config: &DockerConfig{
				Memory: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid memory format",
		},
		{
			name: "invalid memory-swap format",
			config: &DockerConfig{
				MemorySwap: "xyz",
			},
			expectError: true,
			errorMsg:    "invalid memory-swap format",
		},
		{
			name: "invalid cpus format",
			config: &DockerConfig{
				CPUs: "abc",
			},
			expectError: true,
			errorMsg:    "invalid cpus format",
		},
		{
			name: "valid memory formats",
			config: &DockerConfig{
				Memory:     "512m",
				MemorySwap: "1g",
			},
			expectError: false,
		},
		{
			name: "valid cpu formats",
			config: &DockerConfig{
				CPUs: "0.5",
			},
			expectError: false,
		},
		{
			name: "valid network names",
			config: &DockerConfig{
				Network: "my-custom-network",
			},
			expectError: false,
		},
		{
			name: "invalid network name with spaces",
			config: &DockerConfig{
				Network: "my network",
			},
			expectError: true,
			errorMsg:    "invalid network name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function for creating bool pointers in tests
func boolPtr(b bool) *bool {
	return &b
}