package runtime

import (
	"errors"
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
		{UVX, "--from", true},
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
			name:    "python with -m, expect package",
			runtime: Python,
			args:    []string{"-m", "mcp_server_time"},
			want:    "mcp_server_time",
			wantErr: false,
		},
		{
			name:    "python with -m, no package, expect error",
			runtime: Python,
			args:    []string{"-m"},
			wantErr: true,
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
		{
			name:    "npx from with git",
			runtime: UVX,
			args: []string{
				"--from",
				"git+https://github.com/oceanbase/mcp-oceanbase",
				"oceanbase_mcp_server",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "npx from with http",
			runtime: UVX,
			args: []string{
				"--from",
				"https://github.com/oceanbase/mcp-oceanbase",
				"oceanbase_mcp_server",
			},
			want:    "",
			wantErr: true,
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

func TestSpecs_ShouldIgnoreFlag_Additional(t *testing.T) {
	t.Parallel()
	specs := Specs()

	tests := []struct {
		runtime Runtime
		flag    string
		want    bool
	}{
		{Docker, "-d", true},
		{Docker, "--detach", true},
		{Docker, "-i", true},
		{Docker, "--volume", true},
		{Docker, "--unknown-flag", false},
		{Python, "-x", false},
		{UVX, "--unknown", false},
	}

	for _, tc := range tests {
		t.Run(string(tc.runtime)+"/"+tc.flag, func(t *testing.T) {
			t.Parallel()
			spec := specs[tc.runtime]
			got := spec.ShouldIgnoreFlag(tc.flag)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSpecs_ExtractPackageName_Extra(t *testing.T) {
	t.Parallel()
	specs := Specs()

	tests := []struct {
		name    string
		runtime Runtime
		args    []string
		want    string
		wantErr bool
	}{
		// Docker missing image name after run
		{
			name:    "docker run no image",
			runtime: Docker,
			args:    []string{"run", "-d", "--rm"},
			wantErr: true,
		},
		// Docker flags with missing values (should skip next flag but next is missing)
		{
			name:    "docker run missing flag value",
			runtime: Docker,
			args:    []string{"run", "-p"},
			wantErr: true,
		},
		// NPX remote with uppercase scheme, expect error
		{
			name:    "npx remote uppercase scheme",
			runtime: NPX,
			args:    []string{"git+HTTPS://github.com/some/package"},
			wantErr: true,
		},
		// UVX remote with mixed-case scheme
		{
			name:    "uvx remote mixed case https",
			runtime: UVX,
			args:    []string{"--from", "HTTPS://example.com/repo", "pkg"},
			wantErr: true,
		},
		// Python missing module after -m with extra args
		{
			name:    "python -m missing module with extra args",
			runtime: Python,
			args:    []string{"-m", "-x"},
			wantErr: true,
		},
		// Python no -m flag present, expect error
		{
			name:    "python no -m flag",
			runtime: Python,
			args:    []string{"somefile.py"},
			wantErr: true,
		},
		// NPX no args
		{
			name:    "npx no args",
			runtime: NPX,
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := specs[tc.runtime]
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

func Test_dockerPackageExtractor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "basic run with image",
			args: []string{"run", "--rm", "nginx"},
			want: "nginx",
		},
		{
			name: "run with flags before image",
			args: []string{"run", "-d", "--name", "test", "redis"},
			want: "redis",
		},
		{
			name: "volume and network flags before image",
			args: []string{"run", "-v", "/tmp:/tmp", "--network", "host", "ubuntu"},
			want: "ubuntu",
		},
		{
			name:    "no image provided",
			args:    []string{"run", "--rm"},
			wantErr: true,
		},
		{
			name:    "missing run command",
			args:    []string{"exec", "something"},
			wantErr: true,
		},
	}

	extractor := dockerPackageExtractor()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractor(tc.args)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func Test_pythonPackageExtractor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "module flag with package",
			args: []string{"-m", "requests"},
			want: "requests",
		},
		{
			name: "module flag with package and other args",
			args: []string{"-m", "http.server", "--bind", "0.0.0.0"},
			want: "http.server",
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "just -m with no value",
			args:    []string{"-m"},
			wantErr: true,
		},
	}

	extractor := pythonPackageExtractor()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractor(tc.args)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func Test_flagNameSet_ShouldIgnoreFlag(t *testing.T) {
	t.Parallel()

	flags := flagNameSet{
		"--foo": {},
		"-b":    {},
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "known long flag",
			input:    "--foo",
			expected: true,
		},
		{
			name:     "known short flag",
			input:    "-b",
			expected: true,
		},
		{
			name:     "unknown flag",
			input:    "--bar",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := NewSpec(flags, nil)
			ok := spec.ShouldIgnoreFlag(tc.input)
			require.Equal(t, tc.expected, ok)
		})
	}
}

func Test_NewSpec(t *testing.T) {
	t.Parallel()

	flags := flagNameSet{
		"--skip": {},
		"-x":     {},
	}

	extractor := func(args []string) (string, error) {
		if len(args) == 0 {
			return "", errors.New("no args")
		}
		return args[0], nil
	}

	spec := NewSpec(flags, extractor)

	require.True(t, spec.ShouldIgnoreFlag("--skip"))
	require.False(t, spec.ShouldIgnoreFlag("--other"))

	got, err := spec.ExtractPackageName([]string{"somepkg"})
	require.NoError(t, err)
	require.Equal(t, "somepkg", got)
}
