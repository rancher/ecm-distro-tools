package cmd

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/rancher/ecm-distro-tools/registry"
	"github.com/rancher/ecm-distro-tools/release/rke2"
)

type mockRegistryClient struct {
	images map[string]reg.Image
}

func (m *mockRegistryClient) Image(_ context.Context, ref name.Reference) (reg.Image, error) {
	key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
	if img, ok := m.images[key]; ok {
		return img, nil
	}
	return reg.Image{Exists: false, Platforms: make(map[reg.Platform]bool)}, nil
}

func newMockFS() fs.FS {
	return fstest.MapFS{
		"rke2-images-all.linux-amd64.txt": &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime:v1.23.4-rke2r1\nrancher/rke2-cloud-provider:v1.23.4-rke2r1"),
		},
		"rke2-images-all.linux-arm64.txt": &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime:v1.23.4-rke2r1"),
		},
		"rke2-images.windows-amd64.txt": &fstest.MapFile{
			Data: []byte("rancher/rke2-runtime-windows:v1.23.4-rke2r1"),
		},
	}
}

func TestInspectAndCSVOutput(t *testing.T) {
	ossImages := map[string]reg.Image{
		"rancher/rke2-runtime:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[reg.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
				{OS: "linux", Architecture: "arm64"}: true,
			},
		},
		"rancher/rke2-cloud-provider:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[reg.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
			},
		},
	}

	primeImages := map[string]reg.Image{
		"rancher/rke2-runtime:v1.23.4-rke2r1": {
			Exists: true,
			Platforms: map[reg.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
				{OS: "linux", Architecture: "arm64"}: true,
			},
		},
		"rancher/rke2-cloud-provider:v1.23.4-rke2r1": {
			Exists: false,
			Platforms: map[reg.Platform]bool{
				{OS: "linux", Architecture: "amd64"}: true,
			},
		},
	}

	inspector := rke2.NewReleaseInspector(
		newMockFS(),
		&mockRegistryClient{images: ossImages},
		&mockRegistryClient{images: primeImages},
		false,
	)

	results, err := inspector.InspectRelease(context.Background(), "v1.23.4+rke2r1")
	if err != nil {
		t.Fatalf("InspectRelease() error = %v", err)
	}

	var buf bytes.Buffer
	csv(&buf, results)

	expectedBytes, err := os.ReadFile("testdata/inspect_test_output.csv")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	expected := string(expectedBytes)
	if got := buf.String(); got != expected {
		t.Errorf("csv() output = %q, want %q", got, expected)
	}
}

func mustParseRef(s string) name.Reference {
	ref, err := name.ParseReference(s)
	if err != nil {
		panic(err)
	}
	return ref
}
