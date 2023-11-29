package rancher

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNonMirroredRCImages(t *testing.T) {
	const images = "rancher/pushprox-proxy:v0.1.0-rancher2-proxy\nrancher/rancher-agent:v2.7.6-rc5\nrancher/rancher-csp-adapter:v2.0.2\nrancher/rancher-webhook:v0.3.5\nrancher/rancher:v2.7.6-rc5\nrancher/mirrored-rke-tools:v0.1.88-rc3"
	exptectedRCImages := []string{"rancher/rancher-agent:v2.7.6-rc5", "rancher/rancher:v2.7.6-rc5"}
	result := nonMirroredRCImages(images)
	if reflect.DeepEqual(exptectedRCImages, result) != true {
		t.Errorf("failed: result images does not equal expected images, expected %+v, got %+v", exptectedRCImages, result)
	}
}

func TestRancherImages(t *testing.T) {
	const path = "/rancher/rancher/releases/download/v2.7.7-rc4/rancher-images.txt"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf(
				"Expected to request '/rancher/rancher/releases/download/v2.7.7-rc4/rancher-images.txt', got: %s",
				r.URL.Path,
			)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("images"))
	}))
	defer server.Close()

	result, err := rancherImages(server.URL + path)
	if err != nil {
		t.Error(err)
	}
	if result != "images" {
		t.Errorf("Expected 'images', got %s", result)
	}
}

func TestRancherHelmChartVersions(t *testing.T) {
	const path = "/server-charts/latest/index.yaml"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("Expected to request '%s', got: %s", path, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{apiVersion: v1, entries: {rancher: [{appVersion: v2.7.7}, {appVersion: v2.7.6}]}}`))
	}))
	defer server.Close()

	versions, err := rancherHelmChartVersions(server.URL + path)
	if err != nil {
		t.Error(err)
	}
	expectedVersions := []string{"v2.7.7", "v2.7.6"}
	if !reflect.DeepEqual(expectedVersions, versions) {
		t.Errorf("expected %v, got %v", expectedVersions, versions)
	}
}

func TestRancherReleaseName(t *testing.T) {
	const tag = "v2.8.0"
	const expectedGAReleaseName = tag
	const expectedPreReleaseName = "Pre-release " + tag

	gaReleaseName := rancherReleaseName(true, tag)
	if expectedGAReleaseName != gaReleaseName {
		t.Error(errors.New("expected GA release name to be '" + expectedGAReleaseName + "' got '" + gaReleaseName + "' instead"))
	}
	preReleaseName := rancherReleaseName(false, tag)
	if expectedPreReleaseName != preReleaseName {
		t.Error(errors.New("expected GA release name to be '" + expectedPreReleaseName + "' got '" + preReleaseName + "' instead"))
	}
}
