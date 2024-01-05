package k3s

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetK3sChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"collection","links":{"self":"…/v1-release/channels"},"actions":{},"resourceType":"channels","data":[{"id":"latest","type":"channel","links":{"self":"…/v1-release/channels/latest"},"name":"latest","latest":"v1.29.0+k3s1","latestRegexp":".*","excludeRegexp":"(^[^+]+-|v1\\.25\\.5\\+k3s1|v1\\.26\\.0\\+k3s1)"},{"id":"v1.29","type":"channel","links":{"self":"…/v1-release/channels/v1.29"},"name":"v1.29","latest":"v1.29.0+k3s1","latestRegexp":"v1\\.29\\..*","excludeRegexp":"^[^+]+-"}]}`))
	}))
	defer server.Close()

	channels, err := getK3sChannels(server.URL)
	if err != nil {
		t.Error(err)
	}
	if channels.Data[0].ID != "latest" {
		t.Error("first entry should be latest")
	}
	if channels.Data[1].ID != "v1.29" {
		t.Error("second entry should be v1.29")
	}
	if channels.Data[1].Latest != "v1.29.0+k3s1" {
		t.Error("v1.29 latest version should be v1.29.0+k3s1")
	}
}
