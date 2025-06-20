package daemon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid localhost numeric port", "localhost:8080", false},
		{"valid empty host numeric port", ":8080", false},
		{"valid IP address", "127.0.0.1:80", false},
		{"valid IPv6 address", "[::1]:443", false},
		{"valid named port", "localhost:http", false},
		{"missing port", "localhost", true},
		{"invalid port string", "localhost:notaport", true},
		{"missing host and port", "", true},
		{"missing host with invalid port", ":!@#", true},
		{"host only colon", "host:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := IsValidAddr(tt.addr)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
