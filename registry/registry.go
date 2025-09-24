package registry

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
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

// MappingClient rewrites image refs
type MappingClient struct {
	client   *Client
	mappings map[string]string
}

func NewClient(registry string, debug bool) *Client {
	return &Client{registry}
}

func NewMappingClient(registry string, mappings map[string]string, debug bool) *MappingClient {
	return &MappingClient{
		client:   NewClient(registry, debug),
		mappings: mappings,
	}
}

func (m *MappingClient) Inspect(ctx context.Context, ref name.Reference) (Image, error) {
	originalRepo := ref.Context().RepositoryStr()
	if mappedRepo, exists := m.mappings[originalRepo]; exists {
		newRepoName := m.client.registry + "/" + mappedRepo
		newRepo, err := name.NewRepository(newRepoName)
		if err != nil {
			return Image{Platforms: make(map[Platform]bool)}, err
		}

		mappedRef, err := name.NewTag(newRepo.String() + ":" + ref.Identifier())
		if err != nil {
			return Image{Platforms: make(map[Platform]bool)}, err
		}

		return m.client.Inspect(ctx, mappedRef)
	}

	return m.client.Inspect(ctx, ref)
}

func replaceRegistry(registry string, ref name.Reference) (name.Tag, error) {
	newRef, err := name.NewRepository(registry + "/" + ref.Context().RepositoryStr())
	if err != nil {
		return name.Tag{}, err
	}

	return name.NewTag(newRef.String() + ":" + ref.Identifier())
}

func (c *Client) Inspect(ctx context.Context, ref name.Reference) (Image, error) {
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

type ImageFetchResult struct {
	Reference name.Reference
	Results   map[string]Image // registry name -> image info
}

type Inspector interface {
	Inspect(ctx context.Context, ref name.Reference) (Image, error)
}

func NewInspectorFromConfig(regConfig *config.RegistryConfig, debug bool) Inspector {
	registry := regConfig.FullRegistry()
	if len(regConfig.RepositoryMappings) > 0 {
		return NewMappingClient(registry, regConfig.RepositoryMappings, debug)
	}
	return NewClient(registry, debug)
}

type RegistryGroup struct {
	registries map[string]Inspector
}

func NewRegistryGroup(registries map[string]Inspector) *RegistryGroup {
	return &RegistryGroup{
		registries: registries,
	}
}

func (r *RegistryGroup) Images(ctx context.Context, refs []name.Reference) (<-chan ImageFetchResult, <-chan error) {
	resultChan := make(chan ImageFetchResult, len(refs))
	errorChan := make(chan error, len(refs)*len(r.registries))

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		var wg sync.WaitGroup
		emptyImage := Image{
			Exists:    false,
			Platforms: make(map[Platform]bool),
		}

		for _, ref := range refs {
			wg.Add(1)
			go func(imageRef name.Reference) {
				defer wg.Done()

				results := make(map[string]Image)
				for registryName, client := range r.registries {
					image, err := client.Inspect(ctx, imageRef)
					if err != nil {
						errorChan <- err
						image = emptyImage
					}
					results[registryName] = image
				}

				resultChan <- ImageFetchResult{
					Reference: imageRef,
					Results:   results,
				}
			}(ref)
		}

		wg.Wait()
	}()

	return resultChan, errorChan
}
