package rke2

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/rancher/ecm-distro-tools/registry"
)

// RegistryClient defines the interface for interacting with container registries
type RegistryClient interface {
	Image(ctx context.Context, ref name.Reference) (reg.Image, error)
}

type Architecture string

const (
	LinuxAmd64   Architecture = "linux/amd64"
	LinuxArm64   Architecture = "linux/arm64"
	WindowsAmd64 Architecture = "windows/amd64"

	ListLinuxAmd64   = "rke2-images-all.linux-amd64.txt"
	ListLinuxArm64   = "rke2-images-all.linux-arm64.txt"
	ListWindowsAmd64 = "rke2-images.windows-amd64.txt"
)

// ReleaseImage is an image listed in the images file for one or more platforms of a given RKE2 release
type ReleaseImage struct {
	Reference         name.Reference
	ExpectsLinuxAmd64 bool
	ExpectsLinuxArm64 bool
	ExpectsWindows    bool
}

// Image contains the manifest info of an image in the oss and prime registries
type Image struct {
	ReleaseImage
	OSSImage   reg.Image
	PrimeImage reg.Image
}

type ReleaseInspector struct {
	fs    fs.FS
	oss   RegistryClient
	prime RegistryClient
	debug bool
}

func NewReleaseInspector(fs fs.FS, oss, prime RegistryClient, debug bool) *ReleaseInspector {
	return &ReleaseInspector{
		fs:    fs,
		oss:   oss,
		prime: prime,
		debug: debug,
	}
}

func (r *ReleaseInspector) InspectRelease(ctx context.Context, version string) ([]Image, error) {
	if !strings.Contains(version, "+rke2") {
		return nil, errors.New("only RKE2 releases are currently supported")
	}

	requiredImages, err := r.imageMap()
	if err != nil {
		return nil, err
	}

	return r.checkImages(ctx, requiredImages)
}

// imageMap reads per-platform image list files and coalesces them
// into one map to collect images for all platforms.
func (r *ReleaseInspector) imageMap() (map[string]ReleaseImage, error) {
	amd64Images, err := r.readImageList(ListLinuxAmd64)
	if err != nil {
		return nil, err
	}

	arm64Images, err := r.readImageList(ListLinuxArm64)
	if err != nil {
		return nil, err
	}

	winImages, err := r.readImageList(ListWindowsAmd64)
	if err != nil {
		return nil, err
	}

	// merge all images into a single map
	imageMap := make(map[string]ReleaseImage)
	for _, imagePair := range [][2]interface{}{
		{amd64Images, LinuxAmd64},
		{arm64Images, LinuxArm64},
		{winImages, WindowsAmd64},
	} {
		images, arch := imagePair[0].([]string), imagePair[1].(Architecture)
		for _, image := range images {
			if image == "" {
				continue
			}

			ref, err := name.ParseReference(image)
			if err != nil {
				continue
			}

			key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
			info := imageMap[key]
			info.Reference = ref

			switch arch {
			case LinuxAmd64:
				info.ExpectsLinuxAmd64 = true
			case LinuxArm64:
				info.ExpectsLinuxArm64 = true
			case WindowsAmd64:
				info.ExpectsWindows = true
			}

			imageMap[key] = info
		}
	}

	return imageMap, nil
}

// readImageList reads an image list file and returns its contents
func (r *ReleaseInspector) readImageList(filename string) ([]string, error) {
	file, err := r.fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(content)), "\n"), nil
}

// checkImages checks if the required images exist in the OSS and Prime registries
func (r *ReleaseInspector) checkImages(ctx context.Context, requiredImages map[string]ReleaseImage) ([]Image, error) {
	var results []Image

	for _, required := range requiredImages {
		ossImage, err := r.oss.Image(ctx, required.Reference)
		if err != nil {
			ossImage = reg.Image{
				Exists:    false,
				Platforms: make(map[reg.Platform]bool),
			}
		}

		var primeImage reg.Image
		if r.prime != nil {
			primeImage, _ = r.prime.Image(ctx, required.Reference)
		}

		status := Image{
			ReleaseImage: required,
			OSSImage:     ossImage,
			PrimeImage:   primeImage,
		}

		results = append(results, status)
	}

	return results, nil
}
