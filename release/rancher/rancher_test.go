package rancher

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFindRCNonMirroredImages(t *testing.T) {
	images := "rancher/pushprox-proxy:v0.1.0-rancher2-proxy\nrancher/rancher-agent:v2.7.6-rc5\nrancher/rancher-csp-adapter:v2.0.2\nrancher/rancher-webhook:v0.3.5\nrancher/rancher:v2.7.6-rc5\nrancher/mirrored-rke-tools:v0.1.88-rc3"
	exptectedRCImages := []string{"rancher/rancher-agent:v2.7.6-rc5", "rancher/rancher:v2.7.6-rc5"}

	result := nonMirroredRCImages(images)
	if reflect.DeepEqual(exptectedRCImages, result) != true {
		t.Errorf("failed: result images does not equal expected images, expected %+v, got %+v", exptectedRCImages, result)
	}
}

func TestRancherImages(t *testing.T) {
	path := "/rancher/rancher/releases/download/v2.7.7-rc4/rancher-images.txt"
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
