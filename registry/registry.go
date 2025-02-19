package registry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// Platform represents an OS/architecture combination
type Platform struct {
	OS           string
	Architecture string
}

func (p Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Architecture)
}

// Image contains information about an image in a specific registry
type Image struct {
	Exists    bool
	Platforms map[Platform]bool
}

// ImageInfo contains information about an image across registries
type ImageInfo struct {
	Reference  name.Reference
	OSSImage   Image
	PrimeImage Image
}

// Client provides methods for interacting with a container registry
type Client interface {
	GetImageInfo(ctx context.Context, ref name.Reference) (Image, error)
}

type defaultClient struct {
	registry string
	debug    bool
}

// NewClient returns a new registry client for the specified registry
func NewClient(registry string, debug bool) Client {
	return &defaultClient{
		registry: registry,
		debug:    debug,
	}
}

func (c *defaultClient) GetImageInfo(ctx context.Context, ref name.Reference) (Image, error) {
	info := Image{
		Platforms: make(map[Platform]bool),
	}

	newRef, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", c.registry, ref.Context().RepositoryStr(), ref.Identifier()))
	if err != nil {
		return info, fmt.Errorf("creating registry reference: %w", err)
	}

	if c.debug {
		slog.Debug("getting image descriptor", "registry", c.registry, "reference", newRef.String())
	}

	desc, err := remote.Get(newRef)
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusNotFound {
			if c.debug {
				slog.Debug("image not found", "registry", c.registry, "reference", newRef.String())
			}
			return info, nil
		}
		return info, fmt.Errorf("getting descriptor: %w", err)
	}

	info.Exists = true

	if desc.MediaType.IsIndex() {
		if c.debug {
			slog.Debug("handling multi-arch image", "registry", c.registry, "reference", newRef.String())
		}
		if err := c.handleMultiArchImage(desc, &info); err != nil {
			return info, err
		}
	} else {
		if c.debug {
			slog.Debug("handling single-arch image", "registry", c.registry, "reference", newRef.String())
		}
		if err := c.handleSingleArchImage(desc, &info); err != nil {
			return info, err
		}
	}

	if c.debug {
		slog.Debug("image info retrieved",
			"registry", c.registry,
			"reference", newRef.String(),
			"exists", info.Exists,
			"platform_count", len(info.Platforms))
	}

	return info, nil
}

func (c *defaultClient) handleMultiArchImage(desc *remote.Descriptor, info *Image) error {
	idx, err := desc.ImageIndex()
	if err != nil {
		return fmt.Errorf("getting index: %w", err)
	}

	manifest, err := idx.IndexManifest()
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}

	for _, m := range manifest.Manifests {
		platform := Platform{
			OS:           m.Platform.OS,
			Architecture: m.Platform.Architecture,
		}
		info.Platforms[platform] = true
		if c.debug {
			slog.Debug("found platform", "os", platform.OS, "arch", platform.Architecture)
		}
	}

	return nil
}

func (c *defaultClient) handleSingleArchImage(desc *remote.Descriptor, info *Image) error {
	img, err := desc.Image()
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}

	platform := Platform{
		OS:           cfg.OS,
		Architecture: cfg.Architecture,
	}
	info.Platforms[platform] = true
	if c.debug {
		slog.Debug("found platform", "os", platform.OS, "arch", platform.Architecture)
	}

	return nil
}

// ParseReference parses an image reference and returns a normalized reference
func ParseReference(imageRef string) (name.Reference, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}
	return ref, nil
}
