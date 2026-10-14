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
	t.Run("defaults to anonymous", func(t *testing.T) {
		c := NewClient("docker.io", false)
		if c.auth != authn.Anonymous {
			t.Errorf("expected anonymous auth, got %#v", c.auth)
		}
	})

	t.Run("WithBasicAuth sets basic credentials", func(t *testing.T) {
		c := NewClient("registry.rancher.com", false, WithBasicAuth("user", "pass"))
		basic, ok := c.auth.(*authn.Basic)
		if !ok {
			t.Fatalf("expected *authn.Basic, got %#v", c.auth)
		}
		if basic.Username != "user" || basic.Password != "pass" {
			t.Errorf("expected user/pass credentials, got %s/%s", basic.Username, basic.Password)
		}
	})

	t.Run("WithBasicAuth stays anonymous when credentials are empty", func(t *testing.T) {
		c := NewClient("docker.io", false, WithBasicAuth("", ""))
		if c.auth != authn.Anonymous {
			t.Errorf("expected anonymous auth when credentials are empty, got %#v", c.auth)
		}
	})
}

func TestCredentials(t *testing.T) {
	t.Run("resolves per-registry env vars", func(t *testing.T) {
		t.Setenv("PRIME_REGISTRY_USERNAME", "prime-user")
		t.Setenv("PRIME_REGISTRY_PASSWORD", "prime-pass")
		t.Setenv("DOCKERIO_REGISTRY_USERNAME", "docker-user")
		t.Setenv("DOCKERIO_REGISTRY_PASSWORD", "docker-pass")

		if u, p := Credentials("registry.rancher.com"); u != "prime-user" || p != "prime-pass" {
			t.Errorf("expected prime creds, got %s/%s", u, p)
		}
		if u, p := Credentials("docker.io"); u != "docker-user" || p != "docker-pass" {
			t.Errorf("expected docker creds, got %s/%s", u, p)
		}
	})

	t.Run("returns empty for an unknown registry", func(t *testing.T) {
		if u, p := Credentials("unknown.registry.io"); u != "" || p != "" {
			t.Errorf("expected empty creds for unknown registry, got %s/%s", u, p)
		}
	})
}

func TestLookup(t *testing.T) {
	info, ok := Lookup(SUSEStaging)
	if !ok {
		t.Fatal("expected staging registry to be known")
	}
	if info.Host != SUSEStaging {
		t.Errorf("unexpected host: %s", info.Host)
	}
	if info.UsernameEnv != "STAGING_REGISTRY_USERNAME" || info.PasswordEnv != "STAGING_REGISTRY_PASSWORD" {
		t.Errorf("unexpected env vars for staging: %s/%s", info.UsernameEnv, info.PasswordEnv)
	}
	if info.BaseURL == "" || info.AuthURL == "" || info.Service == "" {
		t.Errorf("expected endpoints to be populated, got %+v", info)
	}
	if _, ok := Lookup("unknown.registry.io"); ok {
		t.Error("expected ok=false for unknown registry")
	}
}
