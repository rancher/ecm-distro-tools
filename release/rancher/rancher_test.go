package rancher

import (
	"testing"
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
	images, err := imageSliceToMap(imagesWithoutVersion)
	if err != nil {
		t.Error(err)
	}
	if _, ok := images[imagesWithoutVersion[0]]; !ok {
		t.Error("expected image not found on map " + imagesWithoutVersion[0])
	}
	images, err = imageSliceToMap(imagesWithVersion)
	if err == nil {
		t.Error("expected to flag image with version as malformed")
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

func TestRancherUICLIVersions(t *testing.T) {
	ui := "2.9.2-alpha3"
	cli := "v2.9.0"
	dockerfile := []string{
		"empty line",
		"ENV CATTLE_UI_VERSION " + ui,
		"ENV CATTLE_DASHBOARD_UI_VERSION v2.9.2-alpha3",
		"ENV CATTLE_CLI_VERSION " + cli,
		"",
		"another empty line",
	}
	uiVersion, cliVersion, err := rancherUICLIVersions(dockerfile)
	if err != nil {
		t.Error(err)
	}
	if uiVersion != ui {
		t.Error("wrong ui version, expected '" + ui + "', instead, got: " + uiVersion)
	}
	if cliVersion != cli {
		t.Error("wrong cli version, expected '" + cli + "', instead, got: " + cliVersion)
	}
}

func TestRancherImagesComponentsWithRC(t *testing.T) {
	cisOperatorImage := "rancher/cis-operator v1.0.15-rc.2"
	fleetImage := "rancher/fleet v0.9.9-rc.1"
	systemAgentComponent := "SYSTEM_AGENT_VERSION v0.3.9-rc.4"
	winsAgentComponent := "WINS_AGENT_VERSION v0.4.18-rc1"

	rancherComponents := []string{
		"# Images with -rc",
		cisOperatorImage,
		fleetImage,
		"# Components with -rc",
		systemAgentComponent,
		winsAgentComponent,
		"",
		"# Min version components with -rc",
		"",
		"# Chart/KDM sources",
		"* SYSTEM_CHART_DEFAULT_BRANCH: dev-v2.8 (`scripts/package-env`)",
	}

	images, components, err := rancherImagesComponentsWithRC(rancherComponents)
	if err != nil {
		t.Error(err)
	}

	if images[0] != cisOperatorImage {
		t.Error("image mismatch, expected '" + cisOperatorImage + "', instead, got: " + images[0])
	}
	if images[1] != fleetImage {
		t.Error("image mismatch, expected '" + fleetImage + "', instead, got: " + images[1])
	}
	if components[0] != systemAgentComponent {
		t.Error("image mismatch, expected '" + systemAgentComponent + "', instead, got: " + components[0])
	}
	if components[1] != winsAgentComponent {
		t.Error("image mismatch, expected '" + winsAgentComponent + "', instead, got: " + components[1])
	}
}
