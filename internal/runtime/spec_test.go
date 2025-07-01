package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecs_ShouldIgnoreFlag(t *testing.T) {
	t.Parallel()

	specs := Specs()

	tests := []struct {
		runtime Runtime
		flag    string
		want    bool
	}{
		{Docker, "--rm", true},
		{Docker, "--name", true},
		{Docker, "--local-timezone", false},
		{NPX, "-y", true},
		{NPX, "--help", false},
		{Python, "-m", true},
		{Python, "--debug", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.runtime)+"/"+tc.flag, func(t *testing.T) {
			t.Parallel()

			spec, ok := specs[tc.runtime]
			require.True(t, ok, "runtime spec not found for %s", tc.runtime)
			got := spec.ShouldIgnoreFlag(tc.flag)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSpecs_ExtractPackageName(t *testing.T) {
	t.Parallel()

	specs := Specs()

	tests := []struct {
		name    string
		runtime Runtime
		args    []string
		want    string
		wantErr bool
	}{
		{
			name:    "docker with valid image after run",
			runtime: Docker,
			args: []string{
				"run",
				"-p", "127.0.0.1:4000-4003:4000-4003",
				"-v", "$(pwd)/greptimedb:./greptimedb_data",
				"--name", "greptime",
				"--rm",
				"greptime/greptimedb:latest",
				"standalone", "start",
			},
			want:    "greptime/greptimedb:latest",
			wantErr: false,
		},
		{
			name:    "npx with valid package after -y",
			runtime: NPX,
			args:    []string{"-y", "@some/package"},
			want:    "@some/package",
			wantErr: false,
		},
		{
			name:    "npx with no valid package",
			runtime: NPX,
			args:    []string{"-y", "-v"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "python with -m, expect no package",
			runtime: Python,
			args:    []string{"-m", "mcp_server_time"},
			want:    "",
			wantErr: false,
		},
		{
			name:    "elevenlabs - image name not API key",
			runtime: Docker,
			args: []string{
				"run",
				"-i",
				"--rm",
				"-e",
				"ELEVENLABS_API_KEY",
				"mcp/elevenlabs",
			},
			want:    "mcp/elevenlabs",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec, ok := specs[tc.runtime]
			require.True(t, ok, "runtime spec not found for %s", tc.runtime)
			got, err := spec.ExtractPackageName(tc.args)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
