package rke2

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

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
	inspector := NewReleaseInspector(newMockFS(), nil, nil, false)

	imageMap, err := inspector.imageMap()
	if err != nil {
		t.Fatalf("imageMap() error = %v", err)
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
		if image.ExpectsWindows != expected.win {
			t.Errorf("image %s: got windows = %v, want %v", imageName, image.ExpectsWindows, expected.win)
		}
	}
}

func TestReadImageList(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []string
		wantErr  bool
	}{
		{
			name:     "read rke2-images-all.linux-amd64.txt",
			filename: "rke2-images-all.linux-amd64.txt",
			want:     []string{"rancher/rke2-runtime:v1.23.4-rke2r1", "rancher/rke2-cloud-provider:v1.23.4-rke2r1"},
		},
		{
			name:     "read nonexistent file",
			filename: "fake.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := NewReleaseInspector(newMockFS(), nil, nil, false)

			got, err := inspector.readImageList(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("readImageList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("readImageList() = %v, want %v", got, tt.want)
			}
		})
	}
}
