package rke2

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/ecm-distro-tools/registry"
	"golang.org/x/sync/errgroup"
)

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

// Image contains the manifest info of an image in multiple registries
type Image struct {
	ReleaseImage
	RegistryResults map[string]registry.Image
}

func (i Image) OSSImage() registry.Image {
	if img, ok := i.RegistryResults["oss"]; ok {
		return img
	}
	return registry.Image{Exists: false, Platforms: make(map[registry.Platform]bool)}
}

// PrimeImage returns the Prime registry image for backward compatibility
func (i Image) PrimeImage() registry.Image {
	if img, ok := i.RegistryResults["prime"]; ok {
		return img
	}
	return registry.Image{Exists: false, Platforms: make(map[registry.Platform]bool)}
}

type ReleaseInspector struct {
	assets     fs.FS
	registries map[string]registry.Inspector
	debug      bool
}

func NewReleaseInspector(fs fs.FS, registries map[string]registry.Inspector, debug bool) *ReleaseInspector {
	return &ReleaseInspector{
		assets:     fs,
		registries: registries,
		debug:      debug,
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
	// download image lists for release
	var (
		amd64Images []string
		arm64Images []string
		winImages   []string
	)

	g := new(errgroup.Group)

	g.Go(func() (err error) {
		amd64Images, err = r.imageList(ListLinuxAmd64)
		return err
	})
	g.Go(func() (err error) {
		arm64Images, err = r.imageList(ListLinuxArm64)
		return err
	})
	g.Go(func() (err error) {
		winImages, err = r.imageList(ListWindowsAmd64)
		return
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// merge all images into a map
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

func (r *ReleaseInspector) imageList(filename string) ([]string, error) {
	file, err := r.assets.Open(filename)
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

// checkImages fetches the manifest of all rke2 images in the configured registries
func (r *ReleaseInspector) checkImages(ctx context.Context, requiredImages map[string]ReleaseImage) ([]Image, error) {
	var refs []name.Reference
	refToReleaseImage := make(map[string]ReleaseImage)

	for _, img := range requiredImages {
		refs = append(refs, img.Reference)
		key := img.Reference.Context().RepositoryStr() + ":" + img.Reference.Identifier()
		refToReleaseImage[key] = img
	}

	group := registry.NewRegistryGroup(r.registries)
	resultChan, _ := group.FetchImages(ctx, refs)

	var results []Image
	for fetchResult := range resultChan {
		key := fetchResult.Reference.Context().RepositoryStr() + ":" + fetchResult.Reference.Identifier()
		releaseImage := refToReleaseImage[key]

		result := Image{
			ReleaseImage:    releaseImage,
			RegistryResults: fetchResult.Results,
		}

		results = append(results, result)
	}

	return results, nil
}
