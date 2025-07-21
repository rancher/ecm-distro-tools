package k3s

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/ecm-distro-tools/registry"
)

func newMockFS() fs.FS {
	return os.DirFS("testdata/v1.33.1_k3s1")
}

type mockClient struct {
	images map[string]registry.Image
}

func (m *mockClient) Inspect(ctx context.Context, ref name.Reference) (registry.Image, error) {
	key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
	if image, ok := m.images[key]; ok {
		return image, nil
	}
	return registry.Image{
		Exists:    false,
		Platforms: make(map[registry.Platform]bool),
	}, nil
}

func TestImageMap(t *testing.T) {
	inspector := NewReleaseInspector(newMockFS(), nil, false)

	imageMap, err := inspector.imageMap("v1.33.1+k3s1")
	if err != nil {
		t.Fatalf("imageMap() error = %v", err)
	}

	expectedImages := map[string]struct {
		amd64 bool
		arm64 bool
	}{
		"rancher/k3s:v1.33.1-k3s1": {
			amd64: true,
			arm64: true,
		},
		"rancher/klipper-helm:v0.9.5-build20250306": {
			amd64: true,
			arm64: true,
		},
		"rancher/klipper-lb:v0.4.13": {
			amd64: true,
			arm64: true,
		},
		"rancher/local-path-provisioner:v0.0.31": {
			amd64: true,
			arm64: true,
		},
		"rancher/mirrored-coredns-coredns:1.12.1": {
			amd64: true,
			arm64: true,
		},
		"rancher/mirrored-library-busybox:1.36.1": {
			amd64: true,
			arm64: true,
		},
		"rancher/mirrored-library-traefik:3.3.6": {
			amd64: true,
			arm64: true,
		},
		"rancher/mirrored-metrics-server:v0.7.2": {
			amd64: true,
			arm64: true,
		},
		"rancher/mirrored-pause:3.6": {
			amd64: true,
			arm64: true,
		},
	}

	if len(imageMap) != len(expectedImages) {
		t.Errorf("imageMap() length = %v, want %v", len(imageMap), len(expectedImages))
	}

	for imageName, expected := range expectedImages {
		image, ok := imageMap[imageName]
		if !ok {
			t.Errorf("imageMap() missing expected image %s", imageName)
			continue
		}

		if image.ExpectsLinuxAmd64 != expected.amd64 {
			t.Errorf("image %s: got amd64 = %v, want %v", imageName, image.ExpectsLinuxAmd64, expected.amd64)
		}
		if image.ExpectsLinuxArm64 != expected.arm64 {
			t.Errorf("image %s: got arm64 = %v, want %v", imageName, image.ExpectsLinuxArm64, expected.arm64)
		}
	}
}

func TestInspectRelease(t *testing.T) {
	mockClient := &mockClient{
		images: map[string]registry.Image{
			"rancher/k3s:v1.33.1-k3s1": {
				Exists: true,
				Platforms: map[registry.Platform]bool{
					{OS: "linux", Architecture: "amd64"}: true,
					{OS: "linux", Architecture: "arm64"}: true,
				},
			},
			"rancher/klipper-helm:v0.9.5-build20250306": {
				Exists: true,
				Platforms: map[registry.Platform]bool{
					{OS: "linux", Architecture: "amd64"}: true,
					{OS: "linux", Architecture: "arm64"}: true,
				},
			},
			"rancher/mirrored-pause:3.6": {
				Exists: true,
				Platforms: map[registry.Platform]bool{
					{OS: "linux", Architecture: "amd64"}: true,
					{OS: "linux", Architecture: "arm64"}: true,
				},
			},
		},
	}

	registries := map[string]registry.Inspector{
		"oss": mockClient,
	}

	inspector := NewReleaseInspector(newMockFS(), registries, false)

	results, err := inspector.InspectRelease(context.Background(), "v1.33.1+k3s1")
	if err != nil {
		t.Fatalf("InspectRelease() error = %v", err)
	}

	if len(results) != 9 {
		t.Errorf("InspectRelease() returned %d images, want 9", len(results))
	}

	// check that the main k3s image exists
	found := false
	for _, result := range results {
		if strings.Contains(result.Reference.String(), "rancher/k3s:v1.33.1-k3s1") {
			found = true
			if !result.OSSImage().Exists {
				t.Errorf("expected main k3s image to exist")
			}
			break
		}
	}
	if !found {
		t.Errorf("main k3s image not found in results")
	}
}

func TestInspectReleaseUnsupportedVersion(t *testing.T) {
	inspector := NewReleaseInspector(newMockFS(), nil, false)

	_, err := inspector.InspectRelease(context.Background(), "v1.33.1+rke2")
	if err == nil {
		t.Errorf("InspectRelease() expected error for unsupported version")
	}

	if !strings.Contains(err.Error(), "only k3s releases supported") {
		t.Errorf("InspectRelease() error = %v, want error about k3s support", err)
	}
}
