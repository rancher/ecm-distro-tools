package registry

import (
	"testing"

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
