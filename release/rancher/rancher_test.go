package rancher

import "testing"

const (
	rancherRepoImage = "rancher/rancher"
	rancherVersion   = "v2.9.0"
)

var (
	imagesWithVersion = []string{
		rancherRepoImage + ":" + rancherVersion,
		"rancher/rancher-agent:v2.9.0",
		"k3s-io/k3s:v1.25.4",
	}
	imagesWithoutVersion = []string{
		rancherRepoImage,
		"rancher/rancher-agent",
		"k3s-io/k3s",
	}
)

func TestImageSliceToMap(t *testing.T) {
	images, err := imageSliceToMap(imagesWithoutVersion, true)
	if err != nil {
		t.Error(err)
	}
	if _, ok := images[imagesWithoutVersion[0]]; !ok {
		t.Error("expected image not found on map " + imagesWithoutVersion[0])
	}
	_, err = imageSliceToMap(imagesWithVersion, true)
	if err == nil {
		t.Error("expected to flag image with version as malformed")
	}
	_, err = imageSliceToMap(imagesWithVersion, false)
	if err != nil {
		t.Error("didn't expect to flag image with version as malformed")
	}
}

func TestValidateRepoImage(t *testing.T) {
	if err := validateRepoImage(rancherRepoImage); err != nil {
		t.Error(err)
	}
	if err := validateRepoImage(imagesWithVersion[0]); err == nil {
		t.Error("expected to flag image with version as malformed" + imagesWithVersion[0])
	}
}

func TestSplitImageAndVersion(t *testing.T) {
	repoImage, version, err := splitImageAndVersion(imagesWithVersion[0])
	if err != nil {
		t.Error(err)
	}
	if repoImage != rancherRepoImage {
		t.Error("expected repoImage to be " + rancherRepoImage + " instead, got " + repoImage)
	}
	if version != rancherVersion {
		t.Error("expected version to be " + rancherVersion + " instead, got " + version)
	}
	if _, _, err := splitImageAndVersion(imagesWithoutVersion[0]); err == nil {
		t.Error("expected to flag image without version as malformed " + imagesWithoutVersion[0])
	}
}

func TestGenerateRegsyncConfig(t *testing.T) {
	rancherImage := "rancher/rancher"
	rancherAgentImage := "rancher/rancher-agent"
	rancherVersion := "v2.9.0"
	images := []string{rancherImage + ":" + rancherVersion, rancherAgentImage + ":" + rancherVersion}
	sourceRegistry := "docker.io"
	targetRegistry := "registry.rancher.com"
	sourceRancherImage := sourceRegistry + "/" + rancherImage
	sourceRancherAgentImage := sourceRegistry + "/" + rancherAgentImage
	targetRancherImage := targetRegistry + "/" + rancherImage
	config, err := generateRegsyncConfig(images, sourceRegistry, targetRegistry)
	if err != nil {
		t.Error(err)
	}
	if config.Sync[0].Source != sourceRancherImage {
		t.Error("rancher image should be: '" + sourceRancherImage + "' instead, got: '" + config.Sync[0].Source + "'")
	}
	if config.Sync[0].Target != targetRancherImage {
		t.Error("target rancher image should be: '" + targetRancherImage + "' instead, got: '" + config.Sync[0].Target + "'")
	}
	if config.Sync[0].Tags.Allow[0] != rancherVersion {
		t.Error("rancher version should be: '" + rancherVersion + "' instead, got: '" + config.Sync[0].Tags.Allow[0] + "'")
	}
	if config.Sync[1].Source != sourceRancherAgentImage {
		t.Error("rancher agent image should be: '" + sourceRancherAgentImage + "' instead, got: '" + config.Sync[1].Source + "'")
	}
}

func TestRegistryCredentials(t *testing.T) {
	const (
		fallbackUser = "fallback-user"
		fallbackPass = "fallback-pass"
	)

	t.Run("resolves credentials from the registry env vars", func(t *testing.T) {
		t.Setenv("PRIME_REGISTRY_USERNAME", "prime-user")
		t.Setenv("PRIME_REGISTRY_PASSWORD", "prime-pass")
		user, pass := registryCredentials("registry.rancher.com", fallbackUser, fallbackPass)
		if user != "prime-user" || pass != "prime-pass" {
			t.Errorf("expected env credentials 'prime-user'/'prime-pass', got '%s'/'%s'", user, pass)
		}
	})

	t.Run("uses per-registry env vars so multiple registries get distinct credentials", func(t *testing.T) {
		t.Setenv("PRIME_REGISTRY_USERNAME", "prime-user")
		t.Setenv("PRIME_REGISTRY_PASSWORD", "prime-pass")
		t.Setenv("DOCKERIO_REGISTRY_USERNAME", "docker-user")
		t.Setenv("DOCKERIO_REGISTRY_PASSWORD", "docker-pass")

		primeUser, primePass := registryCredentials("registry.rancher.com", fallbackUser, fallbackPass)
		dockerUser, dockerPass := registryCredentials("docker.io", fallbackUser, fallbackPass)

		if primeUser != "prime-user" || primePass != "prime-pass" {
			t.Errorf("expected prime credentials, got '%s'/'%s'", primeUser, primePass)
		}
		if dockerUser != "docker-user" || dockerPass != "docker-pass" {
			t.Errorf("expected docker credentials, got '%s'/'%s'", dockerUser, dockerPass)
		}
	})

	t.Run("falls back when the registry env vars are not set", func(t *testing.T) {
		t.Setenv("STAGING_REGISTRY_USERNAME", "")
		t.Setenv("STAGING_REGISTRY_PASSWORD", "")
		user, pass := registryCredentials("stgregistry.suse.com", fallbackUser, fallbackPass)
		if user != fallbackUser || pass != fallbackPass {
			t.Errorf("expected fallback credentials, got '%s'/'%s'", user, pass)
		}
	})

	t.Run("falls back for an unknown registry", func(t *testing.T) {
		user, pass := registryCredentials("unknown.registry.io", fallbackUser, fallbackPass)
		if user != fallbackUser || pass != fallbackPass {
			t.Errorf("expected fallback credentials, got '%s'/'%s'", user, pass)
		}
	})
}

func TestGenerateRegsyncConfigCredentials(t *testing.T) {
	config, err := generateRegsyncConfig(imagesWithVersion, "docker.io", "registry.rancher.com")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][2]string{
		"docker.io":            {`{{env "DOCKERIO_REGISTRY_USERNAME"}}`, `{{env "DOCKERIO_REGISTRY_PASSWORD"}}`},
		"registry.rancher.com": {`{{env "PRIME_REGISTRY_USERNAME"}}`, `{{env "PRIME_REGISTRY_PASSWORD"}}`},
	}
	for _, cred := range config.Creds {
		want, ok := expected[cred.Registry]
		if !ok {
			t.Errorf("unexpected registry in creds: %s", cred.Registry)
			continue
		}
		if cred.User != want[0] || cred.Pass != want[1] {
			t.Errorf("registry %s: expected creds %q/%q, got %q/%q", cred.Registry, want[0], want[1], cred.User, cred.Pass)
		}
	}
}
