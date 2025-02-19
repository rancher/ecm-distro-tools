package rke2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/rancher/ecm-distro-tools/registry"
)

type Architecture string

const (
	ArchLinuxAmd64   Architecture = "linux/amd64"
	ArchLinuxArm64   Architecture = "linux/arm64"
	ArchWindowsAmd64 Architecture = "windows/amd64"
)

type imageExpectations struct {
	Reference         name.Reference
	ExpectsLinuxAmd64 bool
	ExpectsLinuxArm64 bool
	ExpectsWindows    bool
}

type ImageStatus struct {
	imageExpectations
	OSSImage   reg.Image
	PrimeImage reg.Image
}

type ReleaseInspector struct {
	fs    fs.FS
	oss   reg.Client
	prime reg.Client
	debug bool
}

func NewReleaseInspector(fs fs.FS, oss, prime reg.Client, debug bool) *ReleaseInspector {
	return &ReleaseInspector{
		fs:    fs,
		oss:   oss,
		prime: prime,
		debug: debug,
	}
}

func (r *ReleaseInspector) InspectRelease(ctx context.Context, version string) ([]ImageStatus, error) {
	if !strings.Contains(version, "+rke2") {
		return nil, errors.New("only RKE2 releases are currently supported")
	}

	archLists, err := r.getArchitectureLists()
	if err != nil {
		return nil, err
	}
	if r.debug {
		slog.Debug("found architecture lists", "lists", mapKeys(archLists))
	}

	imageExpectations, err := r.processImageLists(archLists)
	if err != nil {
		return nil, err
	}
	if r.debug {
		slog.Debug("processed images", "count", len(imageExpectations))
	}

	return r.checkImages(ctx, imageExpectations)
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (r *ReleaseInspector) getArchitectureLists() (map[Architecture]string, error) {
	lists := make(map[Architecture]string)

	entries, err := fs.ReadDir(r.fs, ".")
	if err != nil {
		return nil, fmt.Errorf("reading release assets: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if r.debug {
			slog.Debug("found asset", "name", name)
		}
		switch name {
		case "rke2-images-all.linux-amd64.txt":
			lists[ArchLinuxAmd64] = name
		case "rke2-images-all.linux-arm64.txt":
			lists[ArchLinuxArm64] = name
		case "rke2-images.windows-amd64.txt":
			lists[ArchWindowsAmd64] = name
		}
	}
	return lists, nil
}

func (r *ReleaseInspector) processImageLists(archLists map[Architecture]string) (map[string]imageExpectations, error) {
	imageMap := make(map[string]imageExpectations)

	for arch, filename := range archLists {
		if filename == "" {
			continue
		}

		if r.debug {
			slog.Debug("reading image list", "arch", arch, "filename", filename)
		}
		images, err := r.readImageList(filename)
		if err != nil {
			return nil, err
		}
		if r.debug {
			slog.Debug("found images", "arch", arch, "count", len(images))
		}

		for _, image := range images {
			if image == "" {
				continue
			}

			ref, err := reg.ParseReference(image)
			if err != nil {
				if r.debug {
					slog.Debug("failed to parse image reference", "image", image, "error", err)
				}
				continue
			}

			key := ref.Context().RepositoryStr() + ":" + ref.Identifier()
			info := imageMap[key]
			info.Reference = ref

			switch arch {
			case ArchLinuxAmd64:
				info.ExpectsLinuxAmd64 = true
			case ArchLinuxArm64:
				info.ExpectsLinuxArm64 = true
			case ArchWindowsAmd64:
				info.ExpectsWindows = true
			}

			imageMap[key] = info
		}
	}

	return imageMap, nil
}

func (r *ReleaseInspector) readImageList(filename string) ([]string, error) {
	file, err := r.fs.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening image list: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading image list: %w", err)
	}

	return strings.Split(strings.TrimSpace(string(content)), "\n"), nil
}

func (r *ReleaseInspector) checkImages(ctx context.Context, expectations map[string]imageExpectations) ([]ImageStatus, error) {
	var results []ImageStatus
	if r.debug {
		slog.Debug("checking images in registries", "count", len(expectations))
	}

	for key, expect := range expectations {
		if r.debug {
			slog.Debug("checking image", "image", key)
		}

		ossImage, err := r.oss.GetImageInfo(ctx, expect.Reference)
		if err != nil {
			if r.debug {
				slog.Debug("failed to get OSS info", "image", key, "error", err)
			}
			ossImage = reg.Image{
				Exists:    false,
				Platforms: make(map[reg.Platform]bool),
			}
		}

		var primeImage reg.Image
		if r.prime != nil {
			primeImage, err = r.prime.GetImageInfo(ctx, expect.Reference)
			if err != nil && r.debug {
				slog.Debug("failed to get Prime info", "image", key, "error", err)
			}
		}

		status := ImageStatus{
			imageExpectations: expect,
			OSSImage:          ossImage,
			PrimeImage:        primeImage,
		}

		results = append(results, status)
	}

	if r.debug {
		slog.Debug("found images", "count", len(results))
	}
	return results, nil
}
