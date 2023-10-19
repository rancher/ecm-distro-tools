package rke2

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGoVersions(t *testing.T) {
	path := "/dl/?mode=json"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"version": "go1.21.3", "stable": true}, {"version": "go1.20.10", "stable": true}]`))
	}))
	defer server.Close()

	versions, err := goVersions(server.URL + path)
	if err != nil {
		t.Error(err)
	}
	expectedVersions := []goVersionRecord{{Version: "go1.21.3", Stable: true}, {Version: "go1.20.10", Stable: true}}
	if !reflect.DeepEqual(expectedVersions, versions) {
		t.Errorf("expected %v, got %v", expectedVersions, versions)
	}
}
