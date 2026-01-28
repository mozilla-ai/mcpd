package daemon

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/runtime"
)

func TestDaemon_Dependencies_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		deps    Dependencies
		wantErr string
	}{
		{
			name: "valid dependencies",
			deps: Dependencies{
				APIAddr:        "localhost:8090",
				Logger:         hclog.NewNullLogger(),
				RuntimeServers: []runtime.Server{{}}, // Non-empty to pass validation
			},
		},
		{
			name: "empty runtime servers",
			deps: Dependencies{
				APIAddr:        "localhost:8090",
				Logger:         hclog.NewNullLogger(),
				RuntimeServers: []runtime.Server{},
			},
			wantErr: "runtime server configurations not found",
		},
		{
			name: "invalid API address",
			deps: Dependencies{
				APIAddr:        "invalid-address",
				Logger:         hclog.NewNullLogger(),
				RuntimeServers: []runtime.Server{},
			},
			wantErr: "invalid API address 'invalid-address': invalid address format: address invalid-address: missing port in address",
		},
		{
			name: "nil logger",
			deps: Dependencies{
				APIAddr:        "localhost:8090",
				Logger:         nil,
				RuntimeServers: []runtime.Server{},
			},
			wantErr: "logger cannot be nil",
		},
		{
			name: "logger interface pointing to nil",
			deps: Dependencies{
				APIAddr:        "localhost:8090",
				Logger:         (hclog.Logger)(nil),
				RuntimeServers: []runtime.Server{},
			},
			wantErr: "logger cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.deps.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
			}
		})
	}
}

func TestDaemon_Dependencies_NewDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		logger         hclog.Logger
		apiAddr        string
		runtimeServers []runtime.Server
		wantErr        string
	}{
		{
			name:           "valid dependencies",
			logger:         hclog.NewNullLogger(),
			apiAddr:        "localhost:8090",
			runtimeServers: []runtime.Server{{}}, // Non-empty to pass validation
		},
		{
			name:           "valid dependencies with servers",
			logger:         hclog.NewNullLogger(),
			apiAddr:        "localhost:8090",
			runtimeServers: []runtime.Server{{ /* mock server */ }},
		},
		{
			name:           "nil runtime servers gets empty slice but fails validation",
			logger:         hclog.NewNullLogger(),
			apiAddr:        "localhost:8090",
			runtimeServers: nil,
			wantErr:        "runtime server configurations not found",
		},
		{
			name:           "invalid API address",
			logger:         hclog.NewNullLogger(),
			apiAddr:        "invalid-address",
			runtimeServers: []runtime.Server{},
			wantErr:        "invalid API address 'invalid-address': invalid address format: address invalid-address: missing port in address",
		},
		{
			name:           "nil logger",
			logger:         nil,
			apiAddr:        "localhost:8090",
			runtimeServers: []runtime.Server{},
			wantErr:        "logger cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, err := NewDependencies(tc.logger, tc.apiAddr, tc.runtimeServers)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
				require.Equal(t, Dependencies{}, deps)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.apiAddr, deps.APIAddr)
				require.Equal(t, tc.logger, deps.Logger)
				require.NotNil(t, deps.RuntimeServers)
				if tc.runtimeServers != nil {
					require.Equal(t, tc.runtimeServers, deps.RuntimeServers)
				} else {
					require.Equal(t, []runtime.Server{}, deps.RuntimeServers)
				}
			}
		})
	}
}
