package registry

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

func TestReplaceRegistry(t *testing.T) {
	tests := []struct {
		name      string
		registry  string
		inputRef  string
		expected  string
		expectErr bool
	}{
		{
			name:      "replace docker.io registry",
			registry:  "custom.registry.io",
			inputRef:  "docker.io/library/nginx:latest",
			expected:  "custom.registry.io/library/nginx:latest",
			expectErr: false,
		},
		{
			name:      "handle docker library images",
			registry:  "custom.registry.io",
			inputRef:  "hello-world",
			expected:  "custom.registry.io/library/hello-world:latest",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := name.ParseReference(tt.inputRef)
			if err != nil {
				t.Fatalf("failed to parse input reference: %v", err)
			}

			result, err := replaceRegistry(tt.registry, ref)
			if (err != nil) != tt.expectErr {
				t.Errorf("replaceRegistry() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr && result.Name() != tt.expected {
				t.Errorf("replaceRegistry() = %v, want %v", result.Name(), tt.expected)
			}
		})
	}
}

func TestNewClientAuth(t *testing.T) {
	tests := []struct {
		name         string
		opts         []Option
		wantBasic    bool
		wantUser     string
		wantPassword string
	}{
		{
			name:      "defaults to anonymous",
			opts:      nil,
			wantBasic: false,
		},
		{
			name:         "WithBasicAuth sets basic credentials",
			opts:         []Option{WithBasicAuth("user", "pass")},
			wantBasic:    true,
			wantUser:     "user",
			wantPassword: "pass",
		},
		{
			name:      "WithBasicAuth stays anonymous when credentials are empty",
			opts:      []Option{WithBasicAuth("", "")},
			wantBasic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("docker.io", false, tt.opts...)
			if !tt.wantBasic {
				if c.auth != authn.Anonymous {
					t.Fatalf("expected anonymous auth, got %#v", c.auth)
				}
				return
			}
			basic, ok := c.auth.(*authn.Basic)
			if !ok {
				t.Fatalf("expected *authn.Basic, got %#v", c.auth)
			}
			if basic.Username != tt.wantUser || basic.Password != tt.wantPassword {
				t.Errorf("expected %s/%s, got %s/%s", tt.wantUser, tt.wantPassword, basic.Username, basic.Password)
			}
		})
	}
}

func TestCredentials(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		registry string
		wantUser string
		wantPass string
	}{
		{
			name:     "resolves prime env vars",
			env:      map[string]string{"PRIME_REGISTRY_USERNAME": "prime-user", "PRIME_REGISTRY_PASSWORD": "prime-pass"},
			registry: Prime,
			wantUser: "prime-user",
			wantPass: "prime-pass",
		},
		{
			name:     "resolves docker env vars",
			env:      map[string]string{"DOCKERIO_REGISTRY_USERNAME": "docker-user", "DOCKERIO_REGISTRY_PASSWORD": "docker-pass"},
			registry: DockerIO,
			wantUser: "docker-user",
			wantPass: "docker-pass",
		},
		{
			name:     "returns empty for an unknown registry",
			registry: "unknown.registry.io",
			wantUser: "",
			wantPass: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if u, p := Credentials(tt.registry); u != tt.wantUser || p != tt.wantPass {
				t.Errorf("expected %q/%q, got %q/%q", tt.wantUser, tt.wantPass, u, p)
			}
		})
	}
}

func TestLookup(t *testing.T) {
	tests := []struct {
		name        string
		registry    string
		wantOK      bool
		wantUserEnv string
		wantPassEnv string
	}{
		{
			name:        "known staging registry",
			registry:    SUSEStaging,
			wantOK:      true,
			wantUserEnv: "STAGING_REGISTRY_USERNAME",
			wantPassEnv: "STAGING_REGISTRY_PASSWORD",
		},
		{
			name:        "known prime registry",
			registry:    Prime,
			wantOK:      true,
			wantUserEnv: "PRIME_REGISTRY_USERNAME",
			wantPassEnv: "PRIME_REGISTRY_PASSWORD",
		},
		{
			name:     "unknown registry",
			registry: "unknown.registry.io",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := Lookup(tt.registry)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if !tt.wantOK {
				return
			}
			if info.Host != tt.registry {
				t.Errorf("unexpected host: %s", info.Host)
			}
			if info.UsernameEnv != tt.wantUserEnv || info.PasswordEnv != tt.wantPassEnv {
				t.Errorf("unexpected env vars: %s/%s", info.UsernameEnv, info.PasswordEnv)
			}
			if info.BaseURL == "" || info.AuthURL == "" || info.Service == "" {
				t.Errorf("expected endpoints to be populated, got %+v", info)
			}
		})
	}
}
