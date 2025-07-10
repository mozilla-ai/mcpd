package mcpm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// Define a dummy JSON payload to be served by the mock HTTP server.
// This simulates the content of https://getmcp.io/api/servers.json
const dummyMCPMJSON = `{
	"time": {
		"name": "time",
		"display_name": "Time Server",
		"description": "A server for time and timezone conversions.",
		"license": "MIT",
		"arguments": {
			"TZ": { "description": "Override timezone", "required": false, "example": "America/New_York" }
		},
		"installations": {
			"uvx": {
				"type": "uvx",
				"command": "uvx",
				"args": ["mcp-server-time", "--local-timezone=${TZ}"],
				"description": "Install with uvx",
				"recommended": true,
				"transport": "stdio"
			},
			"python": {
				"type": "python",
				"command": "python",
				"args": ["-m", "mcp_server_time", "--local-timezone=${TZ}", "${ANOTHER_VAR}"],
				"description": "Run with Python module",
				"transport": "stdio"
			},
			"docker": {
				"type": "docker",
				"command": "docker",
				"args": ["run", "-i", "--rm", "mcp/time", "--local-timezone=${TZ}"],
				"description": "Run with Docker",
				"transport": "sse"
			}
		},
		"tools": [
			{ "name": "get_current_time", "description": "Get current time", "inputSchema": {"type": "object", "properties": {"timezone": {"type": "string"}}, "required": ["timezone"]} },
			{ "name": "convert_time", "description": "Convert time", "inputSchema": {"type": "object", "properties": {"source_timezone": {"type": "string"}}, "required": ["source_timezone"]} }
		],
		"is_official": true
	},
	"github": {
		"name": "github",
		"display_name": "GitHub Server",
		"description": "GitHub API interaction.",
		"license": "MIT",
		"arguments": {},
		"installations": {
			"docker": {
				"type": "docker",
				"command": "docker",
				"args": ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"],
				"env": {
					"GITHUB_PERSONAL_ACCESS_TOKEN": "${input:github_token}"
				},
				"description": "Run with Docker",
				"transport": "stdio"
			}
		},
		"tools": [
      {
        "name": "get_me",
        "description": "Create or update a single file in a GitHub repository",
        "inputSchema": {
          "type": "object",
          "properties": {
            "owner": {
              "type": "string",
              "description": "Repository owner (username or organization)"
            },
            "repo": {
              "type": "string",
              "description": "Repository name"
            },
            "path": {
              "type": "string",
              "description": "Path where to create/update the file"
            },
            "content": {
              "type": "string",
              "description": "Content of the file"
            },
            "message": {
              "type": "string",
              "description": "Commit message"
            },
            "branch": {
              "type": "string",
              "description": "Branch to create/update the file in"
            },
            "sha": {
              "type": "string",
              "description": "SHA of the file being replaced (required when updating existing files)"
            }
          },
          "required": [
            "owner",
            "repo",
            "path",
            "content",
            "message",
            "branch"
          ]
        }
      },
      {
        "name": "search_repositories",
        "description": "Search for GitHub repositories",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {
              "type": "string",
              "description": "Search query (see GitHub search syntax)"
            },
            "page": {
              "type": "number",
              "description": "Page number for pagination (default: 1)"
            },
            "perPage": {
              "type": "number",
              "description": "Number of results per page (default: 30, max: 100)"
            }
          },
          "required": [
            "query"
          ]
        }
      }],
		"is_official": true
	},
	"math": {
		"name": "math",
		"display_name": "Math Server",
		"description": "Provides basic math operations.",
		"license": "Apache-2.0",
		"arguments": {
			"API_KEY": { "description": "API key", "required": false, "example": "your_api_key" }
		},
		"installations": {
			"uvx": {
				"type": "uvx",
				"command": "uvx",
				"args": ["mcp-server-math", "--api-key=${API_KEY}"],
				"env":  {"API_KEY": "${API_KEY}"},
				"description": "Install with uvx",
				"recommended": true,
				"transport": "stdio"
			}
		},
		"tools": [{ "name": "add", "description": "Add numbers", "inputSchema": {"type": "object", "properties": {"a": {"type": "number"}, "b": {"type": "number"}}, "required": ["a", "b"]} }],
		"is_official": true
	},
	"no_env_or_args": {
		"name": "no_env_or_args",
		"display_name": "No Env or Args Server",
		"description": "A server with no specific env vars or args.",
		"license": "MIT",
		"arguments": {},
		"installations": {
			"uvx": {
				"type": "uvx",
				"command": "uvx",
				"args": ["mcp-server-no-env"],
				"description": "Simple uvx run",
				"recommended": true,
				"transport": "stdio"
			}
		},
		"tools": [{ "name": "tool1", "description": "Does things with a tool", "inputSchema": {"type": "object", "properties": {"a": {"type": "number"}, "b": {"type": "number"}}, "required": ["a", "b"]} }],
		"is_official": true
	}
}`

func newTestLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Debug,
		Output: os.Stderr,
		Name:   "test.mcpd",
	})
}

func TestNewMCPMRegistry(t *testing.T) {
	// Set up a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/servers.json", r.URL.Path, "Unexpected request path")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(dummyMCPMJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger()

	// Test successful creation
	t.Run("successful creation", func(t *testing.T) {
		registry, err := NewRegistry(logger, ts.URL+"/api/servers.json")
		require.NoError(t, err)
		require.NotNil(t, registry)
		require.Len(t, registry.mcpServers, 4, "Expected 4 servers in the map")
	})

	// Test error on HTTP request failure
	t.Run("http request failure", func(t *testing.T) {
		_, err := NewRegistry(logger, "http://nonexistent-domain.test/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch 'mcpm' registry data from URL")
	})

	// Test error on bad status code
	t.Run("bad status code", func(t *testing.T) {
		badStatusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer badStatusServer.Close()
		_, err := NewRegistry(logger, badStatusServer.URL+"/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "received non-OK HTTP status from 'mcpm' registry for URL")
	})

	// Test error on invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"servers": "not an object"`))
			require.NoError(t, err) // This write should succeed, but the JSON is invalid
		}))
		defer invalidJSONServer.Close()
		_, err := NewRegistry(logger, invalidJSONServer.URL+"/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal 'mcpm' registry JSON")
	})
}

func TestMCPMRegistrySearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(dummyMCPMJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger()
	registry, err := NewRegistry(logger, ts.URL+"/api/mcpServers.json")
	require.NoError(t, err)
	require.NotNil(t, registry)

	tests := []struct {
		name          string
		queryName     string
		filters       map[string]string
		expectedCount int
		expectedIDs   []string
		expectedEnv   map[string][]string // Map of ID to expected configurable env vars
		expectedArgs  map[string][]string // Map of ID to expected configurable cmd line args
	}{
		{
			name:          "Basic search for 'time'",
			queryName:     "time",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "Search for 'TIME' (case-insensitive, explicit env)",
			queryName:     "TIME",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "Search for 'github' (case-insensitive, sample json only supports docker)",
			queryName:     "GitHub",
			filters:       nil,
			expectedCount: 0,
			expectedIDs:   nil,
			expectedEnv:   nil,
		},
		{
			name:          "Search for 'math' by display name",
			queryName:     "math server",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"math"},
			expectedEnv:   map[string][]string{"math": {"API_KEY"}},
			expectedArgs:  map[string][]string{"math": {"--api-key"}},
		},
		{
			name:          "Search with runtime filter 'uvx'",
			queryName:     "*", // Empty query name to match all, then filter by runtime
			filters:       map[string]string{"runtime": "uvx"},
			expectedCount: 3, // time, math, no_env_or_args
			expectedIDs:   []string{"time", "math", "no_env_or_args"},
			expectedEnv:   map[string][]string{"time": {}, "math": {"API_KEY"}, "no_env_or_args": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}, "math": {"--api-key"}, "no_env_or_args": {}},
		},
		{
			name:          "Search with tool filter 'add'",
			queryName:     "*",
			filters:       map[string]string{"tools": "add"},
			expectedCount: 1, // math
			expectedIDs:   []string{"math"},
			expectedEnv:   map[string][]string{"math": {"API_KEY"}},
			expectedArgs:  map[string][]string{"math": {"--api-key"}},
		},
		{
			name:          "Search with non-existent query",
			queryName:     "nonexistent",
			filters:       nil,
			expectedCount: 0,
			expectedIDs:   []string{},
			expectedEnv:   nil,
		},
		{
			name:          "Search with combined filters (runtime: uvx, tool: convert_time)",
			queryName:     "time",
			filters:       map[string]string{"runtime": "uvx", "tools": "convert_time"},
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "Search with combined filters (runtime: docker, tool: convert_time) - docker not supported",
			queryName:     "time",
			filters:       map[string]string{"runtime": "docker", "tools": "convert_time"},
			expectedCount: 0,
			expectedIDs:   nil,
			expectedEnv:   nil,
		},
		{
			name:          "Search with combined filters (math, uvx, add)",
			queryName:     "math",
			filters:       map[string]string{"runtime": "uvx", "tools": "add"},
			expectedCount: 1,
			expectedIDs:   []string{"math"},
			expectedEnv:   map[string][]string{"math": {"API_KEY"}},
			expectedArgs:  map[string][]string{"math": {"--api-key"}},
		},
		{
			name:          "Search with unsupported version filter (expect no match and warning)",
			queryName:     "time",
			filters:       map[string]string{"version": "1.2.3"},
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   nil,
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "Search for 'no_env_or_args'",
			queryName:     "no_env_or_args",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"no_env_or_args"},
			expectedEnv:   map[string][]string{"no_env_or_args": {}}, // Expect empty slice
		},
		{
			name:          "Search for '*'",
			queryName:     "*",
			filters:       nil,
			expectedCount: 3,
			expectedIDs:   []string{"no_env_or_args", "time", "math"},
			expectedEnv:   map[string][]string{"time": {}, "math": {"API_KEY"}, "no_env_or_args": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}, "math": {"--api-key"}, "no_env_or_args": {}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := registry.Search(tc.queryName, tc.filters)
			require.NoError(t, err)
			require.Len(t, results, tc.expectedCount, "Mismatch in result count")

			foundIDs := make([]string, 0, len(results))
			for _, res := range results {
				foundIDs = append(foundIDs, res.ID)
			}
			require.ElementsMatch(t, tc.expectedIDs, foundIDs, "Mismatch in returned IDs")

			if tc.expectedEnv != nil {
				for _, res := range results {
					expectedEnv := tc.expectedEnv[res.ID]
					expectedArgs := tc.expectedArgs[res.ID]
					require.ElementsMatch(t, expectedEnv, res.Arguments.FilterBy(packages.EnvVar).Names())
					require.ElementsMatch(t, expectedArgs, res.Arguments.FilterBy(packages.Argument).Names())
				}
			}
		})
	}
}

func TestMCPMRegistryResolve(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(dummyMCPMJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger()
	registry, err := NewRegistry(logger, ts.URL+"/api/servers.json")
	require.NoError(t, err)
	require.NotNil(t, registry)

	tests := []struct {
		name         string
		id           string
		version      string
		expectError  bool
		expectedID   string
		expectedEnv  []string // Expected configurable env vars for the single result
		expectedArgs []string
	}{
		{
			name:         "Get existing package 'time' with empty version",
			id:           "time",
			version:      "", // Should default to "latest" internally
			expectError:  false,
			expectedID:   "time",
			expectedEnv:  []string{},
			expectedArgs: []string{"--local-timezone"},
		},
		{
			name:         "Get existing package 'time' with 'latest' version",
			id:           "time",
			version:      "latest",
			expectError:  false,
			expectedID:   "time",
			expectedEnv:  []string{},
			expectedArgs: []string{"--local-timezone"},
		},
		{
			name:        "Get non-existent package",
			id:          "nonexistent-package",
			version:     "",
			expectError: true,
			expectedID:  "",
			expectedEnv: nil,
		},
		{
			name:         "Get existing package 'math' with specific version (expect warning, but still return)",
			id:           "math",
			version:      "1.0.0", // MCPM does not filter by version, should return "math"
			expectError:  false,
			expectedID:   "math",
			expectedEnv:  []string{"API_KEY"},
			expectedArgs: []string{"--api-key"},
		},
		{
			name:        "Get 'no_env_or_args' package",
			id:          "no_env_or_args",
			version:     "",
			expectError: false,
			expectedID:  "no_env_or_args",
			expectedEnv: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := registry.Resolve(tc.id, options.WithResolveVersion(tc.version))
			if tc.expectError {
				require.Error(t, err)
				require.Equal(t, packages.Package{}, result, "Expected empty result on error")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedID, result.ID, "Mismatch in returned ID")
				require.ElementsMatch(t, tc.expectedEnv, result.Arguments.FilterBy(packages.EnvVar).Names())
				require.ElementsMatch(t, tc.expectedArgs, result.Arguments.FilterBy(packages.Argument).Names())
			}
		})
	}
}

func TestShouldIgnoreFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rt   runtime.Runtime
		flag string
		want bool
	}{
		{runtime.Docker, "--rm", true},
		{runtime.Docker, "--name", true},
		{runtime.Docker, "--local-timezone", false},
		{runtime.NPX, "-y", true},
		{runtime.Python, "-m", true},
		{runtime.Python, "--debug", false},
		{runtime.UVX, "--experimental", false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s/%s", tc.rt, tc.flag), func(t *testing.T) {
			t.Parallel()

			got := shouldIgnoreFlag(tc.rt, tc.flag)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBuildPackageResult_MultipleSupportedRuntimeInstallations(t *testing.T) {
	t.Parallel()

	server := MCPServer{
		Name:        "time",
		DisplayName: "Time",
		Description: "A Model Context Protocol server that provides time and timezone conversion capabilities. It automatically detects the system's timezone and offers tools for getting current time and converting between timezones.",
		License:     "MIT",
		IsOfficial:  true,
		Arguments: map[string]Argument{
			"TZ": {
				Description: "Environment variable to override the system's default timezone",
				Required:    false,
				Example:     "America/New_York",
			},
		},
		Installations: map[string]Installation{
			"uvx": {
				Type:    "uvx",
				Command: "uvx",
				Args: []string{
					"mcp-server-time",
					"--local-timezone=${TZ}",
				},
				Recommended: true,
			},
			"docker": {
				Type:    "docker",
				Command: "docker",
				Args: []string{
					"run",
					"-i",
					"--rm",
					"mcp/time",
					"--local-timezone=${TZ}",
				},
			},
		},
	}

	reg := &Registry{
		mcpServers: map[string]MCPServer{"time": server},
		logger:     hclog.NewNullLogger(),
		supportedRuntimes: map[runtime.Runtime]struct{}{
			runtime.UVX:    {},
			runtime.Docker: {},
		},
	}

	result, ok := reg.buildPackageResult("time")

	require.True(t, ok, "expected package transformation to succeed")
	require.Contains(t, result.Runtimes, runtime.UVX)
	require.Contains(t, result.Runtimes, runtime.Docker)
	require.Equal(t, "mcp/time", result.InstallationDetails[runtime.Docker].Package)
	require.Equal(t, "mcp-server-time", result.InstallationDetails[runtime.UVX].Package)
	require.Equal(t, "time", result.Name)
}

func TestConvertInstallations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		src               map[string]Installation
		supportedRuntimes map[runtime.Runtime]struct{}
		expected          map[runtime.Runtime]packages.Installation
	}{
		{
			name: "Docker with valid image",
			src: map[string]Installation{
				"docker": {
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run",
						"--rm",
						"greptime/greptimedb:latest",
					},
					Description: "Run GreptimeDB in a container",
					Recommended: true,
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.Docker: {},
			},
			expected: map[runtime.Runtime]packages.Installation{
				runtime.Docker: {
					Command:     "docker",
					Args:        []string{"run", "--rm", "greptime/greptimedb:latest"},
					Package:     "greptime/greptimedb:latest",
					Description: "Run GreptimeDB in a container",
					Recommended: true,
				},
			},
		},
		{
			name: "Unsupported runtime is ignored",
			src: map[string]Installation{
				"python": {
					Type:    "python",
					Command: "python",
					Args:    []string{"-m", "mcp_server_time"},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.Docker: {},
			},
			expected: map[runtime.Runtime]packages.Installation{},
		},
		{
			name: "NPX with simple args",
			src: map[string]Installation{
				"npx": {
					Type:    "npx",
					Command: "npx",
					Args:    []string{"-y", "my-npx-tool"},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.NPX: {},
			},
			expected: map[runtime.Runtime]packages.Installation{
				runtime.NPX: {
					Command: "npx",
					Args:    []string{"-y", "my-npx-tool"},
					Package: "my-npx-tool",
				},
			},
		},
		{
			name: "npm and cli",
			src: map[string]Installation{
				"npm": {
					Type:    "npm",
					Command: "npx",
					Args:    []string{"-y", "@21st-dev/magic@latest", "API_KEY=${API_KEY}"},
					Package: "@21st-dev/magic",
					Env: map[string]string{
						"API_KEY": "${API_KEY}",
					},
					Description: "Install via npm package",
					Recommended: true,
				},
				"cli": {
					Type:    "cli",
					Command: "npx",
					Args: []string{
						"@21st-dev/cli@latest",
						"install",
						"<client>",
						"--api-key",
						"<key>",
					},
					Package:     "",
					Description: "Install using the CLI tool",
					Recommended: true,
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.UVX: {},
				runtime.NPX: {},
			},
			expected: map[runtime.Runtime]packages.Installation{
				runtime.NPX: {
					Command: "npx",
					Args:    []string{"-y", "@21st-dev/magic@latest", "API_KEY=${API_KEY}"},
					Package: "@21st-dev/magic@latest",
					Env: map[string]string{
						"API_KEY": "${API_KEY}",
					},
					Description: "Install via npm package",
					Recommended: true,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := convertInstallations(tc.src, tc.supportedRuntimes)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestExtractArgumentMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		server            MCPServer
		supportedRuntimes map[runtime.Runtime]struct{}
		want              map[string]packages.ArgumentMetadata
	}{
		{
			name: "UVX + Docker install discover placeholders & skip ignored flags",
			server: MCPServer{
				Name: "git",
				Arguments: map[string]Argument{
					"GIT_REPO_PATH": {
						Description: "The path to the Git repository that the mcp-server-git will interact with.",
						Required:    true,
						Example:     "/path/to/git/repo",
					},
				},
				Installations: map[string]Installation{
					// UVX: placeholder is the *next* token after --repository
					"uvx": {
						Type:    "uvx",
						Command: "uvx",
						Args:    []string{"mcp-server-git", "--repository", "${GIT_REPO_PATH}"},
					},
					// Docker: placeholder embedded inside --mount token;
					//        --rm should be ignored.
					"docker": {
						Type:    "docker",
						Command: "docker",
						Args: []string{
							"run", "--rm", "--mount",
							"type=bind,src=${GIT_REPO_PATH},dst=${GIT_REPO_PATH}",
							"mcp/git",
						},
					},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.UVX: {},
				runtime.NPX: {},
			},
			want: map[string]packages.ArgumentMetadata{
				"--repository": {
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "The path to the Git repository that the mcp-server-git will interact with.",
				},
			},
		},
		{
			name: "Env-var placeholder is discovered",
			server: MCPServer{
				Name: "time",
				Arguments: map[string]Argument{
					"TZ": {
						Description: "Environment variable to override the system's default timezone",
						Required:    false,
						Example:     "America/New_York",
					},
				},
				Installations: map[string]Installation{
					"uvx": {
						Type:    "uvx",
						Command: "uvx",
						Args:    []string{"mcp-server-time", "--local-timezone=${TZ}"},
					},
					"docker": {
						Type:    "docker",
						Command: "docker",
						Args:    []string{"run", "--rm", "mcp/time"},
						Env:     map[string]string{"TZ": "${TZ}"},
					},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.NPX: {},
				runtime.UVX: {},
			},
			want: map[string]packages.ArgumentMetadata{
				"--local-timezone": {
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Environment variable to override the system's default timezone",
				},
			},
		},
		{
			name: "Env-var placeholder is discovered looking ahead",
			server: MCPServer{
				Name: "time",
				Arguments: map[string]Argument{
					"TZ": {
						Description: "Environment variable to override the system's default timezone",
						Required:    false,
						Example:     "America/New_York",
					},
				},
				Installations: map[string]Installation{
					"uvx": {
						Type:    "uvx",
						Command: "uvx",
						Args:    []string{"mcp-server-time", "--local-timezone", "${TZ}"},
					},
					"docker": {
						Type:    "docker",
						Command: "docker",
						Args:    []string{"run", "--rm", "mcp/time"},
						Env:     map[string]string{"TZ": "${TZ}"},
					},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.NPX: {},
				runtime.UVX: {},
			},
			want: map[string]packages.ArgumentMetadata{
				"--local-timezone": {
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Environment variable to override the system's default timezone",
				},
			},
		},
		{
			name: "Unsupported runtime is skipped",
			server: MCPServer{
				Name: "foo",
				Arguments: map[string]Argument{
					"BAR": {Description: "", Required: false},
				},
				Installations: map[string]Installation{
					"python": {Type: "python", Command: "python", Args: []string{"-m", "foo"}},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{ // intentionally empty
			},
			want: map[string]packages.ArgumentMetadata{},
		},
		{
			name: "cli types",
			server: MCPServer{
				Name: "magic-mcp",
				Arguments: map[string]Argument{
					"API_KEY": {
						Description: "API key for authentication with Magic AI Agent",
						Required:    true,
						Example:     "your-api-key",
					},
				},
				Installations: map[string]Installation{
					"npm": {
						Type:    "npm",
						Command: "npx",
						Args: []string{
							"-y",
							"@21st-dev/magic@latest",
							"API_KEY=${API_KEY}",
						},
						Package: "@21st-dev/magic",
						Env: map[string]string{
							"API_KEY": "${API_KEY}",
						},
						Recommended: true,
					},
					"cli": {
						Type:    "cli",
						Command: "npx",
						Args: []string{
							"@21st-dev/cli@latest",
							"install",
							"<client>",
							"--api-key",
							"<key>",
						},
						Recommended: true,
					},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.NPX: {},
				runtime.UVX: {},
			},
			want: map[string]packages.ArgumentMetadata{
				"API_KEY": {
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "API key for authentication with Magic AI Agent",
				},
			},
		},
		{
			name: "Env vars declared in installation.Env are returned",
			server: MCPServer{
				Name: "circleci",
				Arguments: map[string]Argument{
					"OKTA_ORG_URL":   {Description: "Okta tenant", Required: true},
					"OKTA_API_TOKEN": {Description: "Okta token", Required: true},
					"OKTA_FOO":       {Description: "Okta foo", Required: true},
				},
				Installations: map[string]Installation{
					"npm": {
						Type:    "npm",
						Command: "npx",
						Args:    []string{"-y", "https://github.com/kapilduraphe/okta-mcp-server"},
						Env: map[string]string{
							"OKTA_ORG_URL":   "${OKTA_ORG_URL}",
							"OKTA_API_TOKEN": "${OKTA_API_TOKEN}",
							"OKTA_BAR":       "${OKTA_FOO}",
						},
					},
				},
			},
			supportedRuntimes: map[runtime.Runtime]struct{}{
				runtime.NPX: {},
				runtime.UVX: {},
			},
			want: map[string]packages.ArgumentMetadata{
				"OKTA_ORG_URL": {
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Okta tenant",
				},
				"OKTA_API_TOKEN": {
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Okta token",
				},
				"OKTA_BAR": {
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Okta foo",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractArgumentMetadata(tc.server, tc.supportedRuntimes)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTool_ToDomainType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   Tool
		want    packages.Tool
		wantErr string
	}{
		{
			name: "valid input",
			input: Tool{
				Name:        "t1",
				Title:       "Tool One",
				Description: "Test tool",
				InputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]any{
						"foo": map[string]any{"type": "string"},
					},
					Required: []string{"foo"},
				},
			},
			want: packages.Tool{
				Name:        "t1",
				Title:       "Tool One",
				Description: "Test tool",
				InputSchema: packages.JSONSchema{
					Type: "object",
					Properties: map[string]any{
						"foo": map[string]any{"type": "string"},
					},
					Required: []string{"foo"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := tc.input.ToDomainType()

			switch {
			case tc.wantErr != "":
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.want, actual)
			}
		})
	}
}

func TestTools_ToDomainType(t *testing.T) {
	t.Parallel()

	validTool := Tool{
		Name:        "valid",
		Title:       "Valid",
		Description: "ok",
		InputSchema: JSONSchema{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
	}

	t.Run("all valid", func(t *testing.T) {
		t.Parallel()

		in := Tools{validTool, validTool}
		out, err := in.ToDomainType()
		require.NoError(t, err)
		require.Len(t, out, 2)
		require.Equal(t, "valid", out[0].Name)
	})
}
