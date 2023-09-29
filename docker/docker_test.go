package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestDockerTag(t *testing.T) {
	org := "rancher"
	repo := "k3s"
	tag := "v1.25.14-k3s1"

	path := "/v2/repositories/" + org + "/" + repo + "/tags/" + tag
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("Expected to request '%s', got: %s", path, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"creator":11292096,"id":516991909,"images":[{"architecture":"amd64","features":"","variant":null,"digest":"sha256:5d972920146d1fdacb806ffff492cf3d6c6b11ef061e7c9b02345e8cdcc1f817","os":"linux","os_features":"","os_version":null,"size":77338322,"status":"active","last_pulled":"2023-09-27T18:25:01.56539Z","last_pushed":"2023-09-21T02:33:00.168796Z"},{"architecture":"arm64","features":"","variant":null,"digest":"sha256:6a42a300bfd291baa45a8fb87768070a70151e11c5a564148319814d66b84179","os":"linux","os_features":"","os_version":null,"size":70783833,"status":"active","last_pulled":"2023-09-27T17:32:19.147729Z","last_pushed":"2023-09-21T00:54:22.424019Z"},{"architecture":"arm","features":"","variant":null,"digest":"sha256:4a897498d92e55eb2d7f610675c5e03100d79d77be80a476684ead3c34c1c3c7","os":"linux","os_features":"","os_version":null,"size":72527158,"status":"active","last_pulled":"2023-09-27T18:03:46.773767Z","last_pushed":"2023-09-21T00:55:31.424019Z"},{"architecture":"s390x","features":"","variant":null,"digest":"sha256:f19501733d2e07b3ad7957a6b1cfe633c4d5c3bc5ea1a6bd8aabc6cb283b30a0","os":"linux","os_features":"","os_version":null,"size":74795527,"status":"active","last_pulled":"2023-09-27T18:03:48.603478Z","last_pushed":"2023-09-21T00:55:31.381413Z"}],"last_updated":"2023-09-21T02:55:46.86982Z","last_updater":11292096,"last_updater_username":"ks3serviceaccount","name":"v1.25.14-k3s1","repository":6586245,"full_size":77338322,"v2":true,"tag_status":"active","tag_last_pulled":"2023-09-27T18:25:01.56539Z","tag_last_pushed":"2023-09-21T02:55:46.86982Z","media_type":"application/vnd.docker.distribution.manifest.list.v2+json","content_type":"image","digest":"sha256:5f7b660e6f2a6dd712350a8f1fd5ad6beafd62a06df5e2f46e432e6f151652fc"}`))
	}))
	defer server.Close()

	images, err := dockerTag(context.TODO(), org, repo, tag, server.URL)
	if err != nil {
		t.Error(err)
	}
	expectedImages := map[string]DockerImage{
		"amd64": {Architecture: "amd64", Status: "active", Size: 77338322, LastPushed: time.Date(2023, time.September, 21, 2, 33, 0, 168796000, time.UTC)},
		"arm":   {Architecture: "arm", Status: "active", Size: 72527158, LastPushed: time.Date(2023, time.September, 21, 0, 55, 31, 424019000, time.UTC)},
		"arm64": {Architecture: "arm64", Status: "active", Size: 70783833, LastPushed: time.Date(2023, time.September, 21, 0, 54, 22, 424019000, time.UTC)},
		"s390x": {Architecture: "s390x", Status: "active", Size: 74795527, LastPushed: time.Date(2023, time.September, 21, 0, 55, 31, 381413000, time.UTC)},
	}
	if !reflect.DeepEqual(expectedImages, images) {
		t.Errorf("expected:\n %+v\ngot:\n %+v", expectedImages, images)
	}
}
