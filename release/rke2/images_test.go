package rke2

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/ecm-distro-tools/registry"
)

type mockRegistryClient struct {
	images map[string]registry.Image
}

func (m *mockRegistryClient) Inspect(_ context.Context, ref name.Reference) (registry.Image, error) {
	key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
	if img, ok := m.images[key]; ok {
		return img, nil
	}
	return registry.Image{Exists: false, Platforms: make(map[registry.Platform]bool)}, nil
}

func newMockFS() fs.FS {
	return fstest.MapFS{
		ListLinuxAmd64: &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime:v1.23.4-rke2r1\nrancher/rke2-cloud-provider:v1.23.4-rke2r1"),
		},
		ListLinuxArm64: &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime:v1.23.4-rke2r1"),
		},
		ListWindowsAmd64: &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime-windows:v1.23.4-rke2r1"),
		},
	}
}

func TestImageMap(t *testing.T) {
	inspector := NewReleaseInspector(newMockFS(), nil, false)

	images, err := inspector.releaseImages()
	if err != nil {
		t.Fatalf("releaseImage() error = %v", err)
	}

	expectedImages := map[string]struct {
		amd64 bool
		arm64 bool
		win   bool
	}{
		"rancher/rke2-runtime:v1.23.4-rke2r1": {
			amd64: true,
			arm64: true,
			win:   false,
		},
		"rancher/rke2-cloud-provider:v1.23.4-rke2r1": {
			amd64: true,
			arm64: false,
			win:   false,
		},
		"rancher/rke2-runtime-windows:v1.23.4-rke2r1": {
			amd64: false,
			arm64: false,
			win:   true,
		},
	}

	for imageName, expected := range expectedImages {
		image, ok := images[imageName]
		if !ok {
			t.Errorf("releaseImages() missing expected image %s", imageName)
			continue
		}

		if image.ExpectsLinuxAmd64 != expected.amd64 {
			t.Errorf("image %s: got amd64 = %v, want %v", imageName, image.ExpectsLinuxAmd64, expected.amd64)
		}
		if image.ExpectsLinuxArm64 != expected.arm64 {
			t.Errorf("image %s: got arm64 = %v, want %v", imageName, image.ExpectsLinuxArm64, expected.arm64)
		}
		if image.ExpectsWindows != expected.win {
			t.Errorf("image %s: got windows = %v, want %v", imageName, image.ExpectsWindows, expected.win)
		}
	}
}

func TestInspectRelease(t *testing.T) {
	ossImages := map[string]registry.Image{
		"rancher/rke2-runtime:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[registry.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
				{OS: "linux", Architecture: "arm64"}: true,
			},
		},
		"rancher/rke2-cloud-provider:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[registry.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
			},
		},
	}

	primeImages := map[string]registry.Image{
		"rancher/rke2-runtime:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[registry.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
				{OS: "linux", Architecture: "arm64"}: true,
			},
		},
		"rancher/rke2-cloud-provider:v1.23.4-rke2r1": {
			Exists: false,
			Platforms: map[registry.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
			},
		},
	}

	registries := map[string]registry.Inspector{
		"oss":   &mockRegistryClient{images: ossImages},
		"prime": &mockRegistryClient{images: primeImages},
	}

	inspector := NewReleaseInspector(newMockFS(), registries, false)

	results, err := inspector.InspectRelease(context.Background(), "v1.23.4+rke2r1")
	if err != nil {
		t.Fatalf("InspectRelease() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("InspectRelease() returned %d images, want 3", len(results))
	}

	// Check specific results
	for _, result := range results {
		imageName := result.Reference.Context().RepositoryStr() + ":" + result.Reference.Identifier()
		switch imageName {
		case "rancher/rke2-runtime:v1.23.4-rke2r1":
			if !result.OSSImage().Exists {
				t.Errorf("expected OSSImage to exist, but it doesn't")
			}
			if !result.PrimeImage().Exists {
				t.Errorf("expected PrimeImage to exist, but it doesn't")
			}
		case "rancher/rke2-cloud-provider:v1.23.4-rke2r1":
			if !result.OSSImage().Exists {
				t.Errorf("expected OSSImage to exist, but it doesn't")
			}
			if result.PrimeImage().Exists {
				t.Errorf("expected PrimeImage not to exist, but it does")
			}
		}
	}
}

func TestInspectReleaseUnsupportedVersion(t *testing.T) {
	inspector := NewReleaseInspector(newMockFS(), nil, false)

	_, err := inspector.InspectRelease(context.Background(), "v1.25.4+k3s1")
	if err == nil {
		t.Errorf("InspectRelease() expected error for unsupported version")
	}

	if !strings.Contains(err.Error(), "only RKE2 releases are currently supported") {
		t.Errorf("InspectRelease() error = %v, want error about RKE2 support", err)
	}
}
