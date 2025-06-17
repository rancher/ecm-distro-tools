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
	images, err = imageSliceToMap(imagesWithVersion, true)
	if err == nil {
		t.Error("expected to flag image with version as malformed")
	}
	images, err = imageSliceToMap(imagesWithVersion, false)
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
