package prime

import (
	"slices"
	"testing"
)

func TestGenerateArtifactsIndexContentGA(t *testing.T) {
	rancherKeys := []string{"rancher/v2.10.0/images.txt"}
	rke2Keys := []string{"rke2/v1.30.1+rke2r1/images.txt"}
	got := generateArtifactsIndexContent(rancherKeys, rke2Keys, nil)
	if !slices.Equal(got.GA.Rancher.Versions, []string{"v2.10.0"}) {
		t.Fatalf("unexpected GA rancher versions: %v", got.GA.Rancher.Versions)
	}
	if len(got.PreRelease.Rancher.Versions) != 0 {
		t.Fatalf("expected no prerelease rancher versions, got %v", got.PreRelease.Rancher.Versions)
	}
	if !slices.Equal(got.GA.RKE2.Versions, []string{"v1.30.1+rke2r1"}) {
		t.Fatalf("unexpected GA rke2 versions: %v", got.GA.RKE2.Versions)
	}
	if len(got.PreRelease.RKE2.Versions) != 0 {
		t.Fatalf("expected no prerelease rke2 versions, got %v", got.PreRelease.RKE2.Versions)
	}
}

func TestGenerateArtifactsIndexContentPreRelease(t *testing.T) {
	rancherKeys := []string{
		"rancher/v2.9.0-rc1/images.txt",
		"rancher/v2.9.0-rc1/foo.txt",
		"rancher/v2.9.0-hotfix-1/images.txt",
	}
	got := generateArtifactsIndexContent(rancherKeys, nil, nil)
	wantPre := []string{"v2.9.0-rc1", "v2.9.0-hotfix-1"}
	if !slices.Equal(got.PreRelease.Rancher.Versions, wantPre) {
		t.Fatalf("unexpected prerelease versions: %v", got.PreRelease.Rancher.Versions)
	}
	files := got.PreRelease.Rancher.VersionsFiles["v2.9.0-rc1"]
	if !slices.Equal(files, []string{
		"images.txt",
		"foo.txt",
	}) {
		t.Fatalf("unexpected files for rc1: %v", files)
	}
}

func TestGenerateArtifactsIndexContentIgnoredVersions(t *testing.T) {
	rancherKeys := []string{
		"rancher/v2.8.1/images.txt",
		"rancher/v2.8.2/images.txt",
	}
	ignore := map[string]bool{"v2.8.1": true}
	got := generateArtifactsIndexContent(rancherKeys, nil, ignore)
	if len(got.GA.Rancher.Versions) != 1 || got.GA.Rancher.Versions[0] != "v2.8.2" {
		t.Fatalf("expected only v2.8.2, got %v", got.GA.Rancher.Versions)
	}
	if _, ok := got.GA.Rancher.VersionsFiles["v2.8.1"]; ok {
		t.Fatalf("ignored version present in files map")
	}
}

func TestGenerateArtifactsIndexContentSkipUnexpectedKeys(t *testing.T) {
	rancherKeys := []string{
		"unexpected/v2.10.0/file.txt",
		"rancher/v2.11.0/",
		"rancher/v2.11.0/file.txt",
	}
	got := generateArtifactsIndexContent(rancherKeys, nil, nil)
	if !slices.Equal(got.GA.Rancher.Versions, []string{"v2.11.0"}) {
		t.Fatalf("expected only valid version captured, got %v", got.GA.Rancher.Versions)
	}
	if files := got.GA.Rancher.VersionsFiles["v2.11.0"]; !slices.Equal(files, []string{"file.txt"}) {
		t.Fatalf("unexpected files for v2.11.0: %v", files)
	}
}
