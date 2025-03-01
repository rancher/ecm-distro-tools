package registry

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type Platform struct {
	OS           string
	Architecture string
}

func (p Platform) String() string {
	return p.OS + "/" + p.Architecture
}

type Image struct {
	Platforms map[Platform]bool
	Exists    bool
}

type Client struct {
	registry string
}

func NewClient(registry string, debug bool) *Client {
	return &Client{registry}
}

func replaceRegistry(registry string, ref name.Reference) (name.Tag, error) {
	newRef, err := name.NewRepository(registry + "/" + ref.Context().RepositoryStr())
	if err != nil {
		return name.Tag{}, err
	}

	return name.NewTag(newRef.String() + ":" + ref.Identifier())
}

func (c *Client) Image(ctx context.Context, ref name.Reference) (Image, error) {
	info := Image{
		Platforms: make(map[Platform]bool),
	}

	tagRef, err := replaceRegistry(c.registry, ref)
	if err != nil {
		return info, err
	}

	desc, err := remote.Get(tagRef)
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusNotFound {
			return info, nil
		}
		return info, err
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
