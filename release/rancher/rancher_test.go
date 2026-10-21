package rancher

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/ecm-distro-tools/registry"
)

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

	tests := []struct {
		name        string
		env         map[string]string
		perRegistry map[string]Credentials
		registry    string
		wantUser    string
		wantPass    string
	}{
		{
			name:     "resolves credentials from the registry env vars",
			env:      map[string]string{"PRIME_REGISTRY_USERNAME": "prime-user", "PRIME_REGISTRY_PASSWORD": "prime-pass"},
			registry: registry.Prime,
			wantUser: "prime-user",
			wantPass: "prime-pass",
		},
		{
			name:     "resolves distinct credentials for docker.io",
			env:      map[string]string{"DOCKERIO_REGISTRY_USERNAME": "docker-user", "DOCKERIO_REGISTRY_PASSWORD": "docker-pass"},
			registry: registry.DockerIO,
			wantUser: "docker-user",
			wantPass: "docker-pass",
		},
		{
			name:        "explicit per-registry credentials take precedence over env vars",
			env:         map[string]string{"DOCKERIO_REGISTRY_USERNAME": "docker-user", "DOCKERIO_REGISTRY_PASSWORD": "docker-pass"},
			perRegistry: map[string]Credentials{registry.DockerIO: {Username: "explicit-user", Password: "explicit-pass"}},
			registry:    registry.DockerIO,
			wantUser:    "explicit-user",
			wantPass:    "explicit-pass",
		},
		{
			name:        "ignores explicit credentials for a different registry",
			perRegistry: map[string]Credentials{registry.Prime: {Username: "prime-user", Password: "prime-pass"}},
			registry:    registry.DockerIO,
			wantUser:    fallbackUser,
			wantPass:    fallbackPass,
		},
		{
			name:     "falls back when the registry env vars are not set",
			env:      map[string]string{"STAGING_REGISTRY_USERNAME": "", "STAGING_REGISTRY_PASSWORD": ""},
			registry: registry.SUSEStaging,
			wantUser: fallbackUser,
			wantPass: fallbackPass,
		},
		{
			name:     "falls back for an unknown registry",
			registry: "unknown.registry.io",
			wantUser: fallbackUser,
			wantPass: fallbackPass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			user, pass := registryCredentials(tt.registry, tt.perRegistry, fallbackUser, fallbackPass)
			if user != tt.wantUser || pass != tt.wantPass {
				t.Errorf("expected %q/%q, got %q/%q", tt.wantUser, tt.wantPass, user, pass)
			}
		})
	}
}

func TestCheckIfImageExists(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		authenticated bool
		wantExists    bool
		wantErr       bool
	}{
		{name: "image exists", statusCode: http.StatusOK, authenticated: true, wantExists: true},
		{name: "not found is missing", statusCode: http.StatusNotFound, authenticated: true, wantExists: false},
		{name: "anonymous 401 is treated as missing", statusCode: http.StatusUnauthorized, authenticated: false, wantExists: false},
		{name: "authenticated 401 is an error", statusCode: http.StatusUnauthorized, authenticated: true, wantErr: true},
		{name: "unexpected status is an error", statusCode: http.StatusInternalServerError, authenticated: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			exists, err := checkIfImageExists(server.URL, "rancher/rancher", "v2.9.0", "token", tt.authenticated)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tt.wantExists {
				t.Errorf("expected exists=%v, got %v", tt.wantExists, exists)
			}
		})
	}
}

func TestGenerateRegsyncConfigCredentials(t *testing.T) {
	config, err := generateRegsyncConfig(imagesWithVersion, registry.DockerIO, registry.Prime)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		registry string
		wantUser string
		wantPass string
	}{
		{registry.DockerIO, `{{env "DOCKERIO_REGISTRY_USERNAME"}}`, `{{env "DOCKERIO_REGISTRY_PASSWORD"}}`},
		{registry.Prime, `{{env "PRIME_REGISTRY_USERNAME"}}`, `{{env "PRIME_REGISTRY_PASSWORD"}}`},
	}

	creds := make(map[string]regsyncCreds, len(config.Creds))
	for _, cred := range config.Creds {
		creds[cred.Registry] = cred
	}

	for _, tt := range tests {
		t.Run(tt.registry, func(t *testing.T) {
			cred, ok := creds[tt.registry]
			if !ok {
				t.Fatalf("missing creds for registry %s", tt.registry)
			}
			if cred.User != tt.wantUser || cred.Pass != tt.wantPass {
				t.Errorf("expected %q/%q, got %q/%q", tt.wantUser, tt.wantPass, cred.User, cred.Pass)
			}
		})
	}
}
