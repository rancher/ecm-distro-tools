package registry

import (
	"context"
	"errors"
	"fmt"
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

type Image struct {
	Exists    bool
	Platforms map[Platform]bool
}

type Client struct {
	registry string
}

// NewClient returns a new registry client for the specified registry
func NewClient(registry string, debug bool) *Client {
	return &Client{registry}
}

func (c *Client) Image(ctx context.Context, ref name.Reference) (Image, error) {
	info := Image{
		Platforms: make(map[Platform]bool),
	}

	newRef, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", c.registry, ref.Context().RepositoryStr(), ref.Identifier()))
	if err != nil {
		return info, err
	}

	desc, err := remote.Get(newRef)
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusNotFound {
			return info, nil
		}
		return info, fmt.Errorf("getting descriptor: %w", err)
	}

	info.Exists = true

	if desc.MediaType.IsIndex() {
		if err := c.handleMultiArchImage(desc, &info); err != nil {
			return info, err
		}
	} else {
		if err := c.handleSingleArchImage(desc, &info); err != nil {
			return info, err
		}
	}

	return info, nil
}

func (c *Client) handleMultiArchImage(desc *remote.Descriptor, info *Image) error {
	idx, err := desc.ImageIndex()
	if err != nil {
		return err
	}

	manifest, err := idx.IndexManifest()
	if err != nil {
		return err
	}

	for _, m := range manifest.Manifests {
		platform := Platform{
			OS:           m.Platform.OS,
			Architecture: m.Platform.Architecture,
		}
		info.Platforms[platform] = true
	}

	return nil
}

func (c *Client) handleSingleArchImage(desc *remote.Descriptor, info *Image) error {
	img, err := desc.Image()
	if err != nil {
		return err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return err
	}

	platform := Platform{
		OS:           cfg.OS,
		Architecture: cfg.Architecture,
	}
	info.Platforms[platform] = true

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
