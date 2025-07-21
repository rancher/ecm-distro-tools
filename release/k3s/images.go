package k3s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/ecm-distro-tools/registry"
)

const (
	k3sImagesFile = "k3s-images.txt"
)

type ReleaseImage struct {
	Reference         name.Reference
	ExpectsLinuxAmd64 bool
	ExpectsLinuxArm64 bool
}

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

type ReleaseInspector struct {
	assets     fs.FS
	registries map[string]registry.Inspector
	debug      bool
}

func NewReleaseInspector(assets fs.FS, registries map[string]registry.Inspector, debug bool) *ReleaseInspector {
	return &ReleaseInspector{
		assets:     assets,
		registries: registries,
		debug:      debug,
	}
}

func (r *ReleaseInspector) InspectRelease(ctx context.Context, version string) ([]Image, error) {
	if !strings.Contains(version, "+k3s") {
		return nil, errors.New("only k3s releases supported")
	}

	requiredImages, err := r.imageMap(version)
	if err != nil {
		return nil, err
	}

	return r.checkImages(ctx, requiredImages)
}

// imageMap reads the k3s-images.txt file and creates image map
func (r *ReleaseInspector) imageMap(version string) (map[string]ReleaseImage, error) {
	imageMap := make(map[string]ReleaseImage)

	// convert version format for docker tag
	imageTag := strings.ReplaceAll(version, "+", "-")
	mainImageRef := "rancher/k3s:" + imageTag
	ref, err := name.ParseReference(mainImageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse k3s image reference: %w", err)
	}

	key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
	imageMap[key] = ReleaseImage{
		Reference:         ref,
		ExpectsLinuxAmd64: true,
		ExpectsLinuxArm64: true,
	}

	// read additional images from k3s-images.txt
	images, err := r.imageList(K3sImagesFile)
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if image == "" {
			continue
		}

		ref, err := name.ParseReference(image)
		if err != nil {
			if r.debug {
				fmt.Printf("skipping invalid image reference: %s\n", image)
			}
			continue
		}

		key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
		imageMap[key] = ReleaseImage{
			Reference:         ref,
			ExpectsLinuxAmd64: true,
			ExpectsLinuxArm64: true,
		}
	}

	return imageMap, nil
}

// imageList reads an image list file and returns its contents
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

// checkImages fetches the manifest of all k3s images in the configured registries
func (r *ReleaseInspector) checkImages(ctx context.Context, requiredImages map[string]ReleaseImage) ([]Image, error) {
	var refs []name.Reference
	refToReleaseImage := make(map[string]ReleaseImage)

	for _, img := range requiredImages {
		refs = append(refs, img.Reference)
		key := img.Reference.Context().RepositoryStr() + ":" + img.Reference.Identifier()
		refToReleaseImage[key] = img
	}

	// Fetch images concurrently
	group := registry.NewRegistryGroup(r.registries)
	resultChan, _ := group.FetchImages(ctx, refs)

	// Collect results
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
