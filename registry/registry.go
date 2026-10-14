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

// credentialsEnv maps a registry host to the names of the environment variables
// that hold its credentials, matching the names used by the regsync config.
var credentialsEnv = map[string]struct {
	Username string
	Password string
}{
	"docker.io":            {"DOCKERIO_REGISTRY_USERNAME", "DOCKERIO_REGISTRY_PASSWORD"},
	"registry.rancher.com": {"PRIME_REGISTRY_USERNAME", "PRIME_REGISTRY_PASSWORD"},
	"stgregistry.suse.com": {"STAGING_REGISTRY_USERNAME", "STAGING_REGISTRY_PASSWORD"},
}

// CredentialsEnvVars returns the username and password environment variable
// names for a registry, and whether the registry is known.
func CredentialsEnvVars(registry string) (usernameEnv, passwordEnv string, ok bool) {
	env, ok := credentialsEnv[registry]
	return env.Username, env.Password, ok
}

// Credentials returns the username and password for a registry read from its
// environment variables, or empty strings when the registry is unknown or the
// variables aren't set.
func Credentials(registry string) (username, password string) {
	env, ok := credentialsEnv[registry]
	if !ok {
		return "", ""
	}
	return os.Getenv(env.Username), os.Getenv(env.Password)
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

// Option configures a Client.
type Option func(*Client)

// WithBasicAuth authenticates registry requests with the given credentials.
// When either value is empty the client stays anonymous, so callers can pass
// optional credentials without extra branching.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		if username != "" && password != "" {
			c.auth = &authn.Basic{Username: username, Password: password}
		}
	}
}

func NewClient(registry string, debug bool, opts ...Option) *Client {
	c := &Client{registry: registry, auth: authn.Anonymous}
	for _, opt := range opts {
		opt(c)
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
