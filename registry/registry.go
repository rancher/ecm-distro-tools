package registry

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// Registry hosts used across the release tooling.
const (
	DockerIO    = "docker.io"
	Prime       = "registry.rancher.com"
	SUSEStaging = "stgregistry.suse.com"

	// OSS is the registry that hosts the open-source (public) images.
	OSS = DockerIO
)

// Info is the single descriptor for a container registry. It holds the host,
// the endpoints used for the Docker v2 token auth flow, and the names of the
// environment variables that hold its credentials (the same names used by the
// regsync config).
type Info struct {
	Host        string
	BaseURL     string
	AuthURL     string
	Service     string
	UsernameEnv string
	PasswordEnv string
}

const suseRegistryService = "SUSE+Linux+Docker+Registry"

// registries is the single source of truth for registry metadata.
var registries = map[string]Info{
	DockerIO: {
		Host:        DockerIO,
		BaseURL:     "https://registry-1.docker.io",
		AuthURL:     "https://auth.docker.io/token",
		Service:     "registry.docker.io",
		UsernameEnv: "DOCKERIO_REGISTRY_USERNAME",
		PasswordEnv: "DOCKERIO_REGISTRY_PASSWORD",
	},
	Prime: {
		Host:        Prime,
		BaseURL:     "https://registry.rancher.com",
		AuthURL:     "https://scc.suse.com/api/registry/authorize",
		Service:     suseRegistryService,
		UsernameEnv: "PRIME_REGISTRY_USERNAME",
		PasswordEnv: "PRIME_REGISTRY_PASSWORD",
	},
	SUSEStaging: {
		Host:        SUSEStaging,
		BaseURL:     "https://stgregistry.suse.com",
		AuthURL:     "https://stgscc.suse.com/api/registry/authorize",
		Service:     suseRegistryService,
		UsernameEnv: "STAGING_REGISTRY_USERNAME",
		PasswordEnv: "STAGING_REGISTRY_PASSWORD",
	},
}

// Lookup returns the descriptor for a registry host and whether it is known.
func Lookup(host string) (Info, bool) {
	info, ok := registries[host]
	return info, ok
}

// Credentials returns the username and password for a registry read from its
// environment variables, or empty strings when the registry is unknown or the
// variables aren't set.
func Credentials(host string) (string, string) {
	info, ok := registries[host]
	if !ok {
		return "", ""
	}
	return os.Getenv(info.UsernameEnv), os.Getenv(info.PasswordEnv)
}

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
	auth     authn.Authenticator
}

// ClientOpts holds optional configuration for a Client. When Username and
// Password are both set, the client authenticates with those basic credentials;
// otherwise it stays anonymous.
type ClientOpts struct {
	Username string
	Password string
}

func NewClient(registry string, debug bool, opts *ClientOpts) *Client {
	c := &Client{registry: registry, auth: authn.Anonymous}
	if opts != nil && opts.Username != "" && opts.Password != "" {
		c.auth = &authn.Basic{Username: opts.Username, Password: opts.Password}
	}
	return c
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

	desc, err := remote.Get(tagRef, remote.WithContext(ctx), remote.WithAuth(c.auth))
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
