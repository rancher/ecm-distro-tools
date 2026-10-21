package cmd

import (
	"testing"

	"github.com/rancher/ecm-distro-tools/release/rancher"
)

func TestParseRegistryCredentials(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		want    map[string]rancher.Credentials
		wantErr bool
	}{
		{
			name:    "empty returns nil",
			entries: nil,
			want:    nil,
		},
		{
			name:    "parses multiple registries",
			entries: []string{"docker.io=user:pass", "registry.rancher.com=user2:pass2"},
			want: map[string]rancher.Credentials{
				"docker.io":            {Username: "user", Password: "pass"},
				"registry.rancher.com": {Username: "user2", Password: "pass2"},
			},
		},
		{
			name:    "password may contain colons",
			entries: []string{"docker.io=user:pa:ss:word"},
			want: map[string]rancher.Credentials{
				"docker.io": {Username: "user", Password: "pa:ss:word"},
			},
		},
		{
			name:    "missing separator is an error",
			entries: []string{"docker.io-user-pass"},
			wantErr: true,
		},
		{
			name:    "missing password is an error",
			entries: []string{"docker.io=user"},
			wantErr: true,
		},
		{
			name:    "empty host is an error",
			entries: []string{"=user:pass"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRegistryCredentials(tt.entries)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d entries, got %d", len(tt.want), len(got))
			}
			for host, want := range tt.want {
				if got[host] != want {
					t.Errorf("host %s: expected %+v, got %+v", host, want, got[host])
				}
			}
		})
	}
}
