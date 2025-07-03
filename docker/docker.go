package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/sirupsen/logrus"
)

const registryURL = "https://hub.docker.com"

type DockerImage struct {
	Architecture string    `json:"architecture"`
	Status       string    `json:"status"`
	Size         int       `json:"size"`
	LastPushed   time.Time `json:"last_pushed"`
}

type DockerTag struct {
	Name   string        `json:"name"`
	Images []DockerImage `json:"images"`
}

// CheckImageArchs checks if an image exists and has all the provided architectures
func CheckImageArchs(ctx context.Context, org, repo, tag string, archs []string) error {
	images, err := dockerTag(ctx, org, repo, tag, registryURL)
	if err != nil {
		return err
	}

	for _, arch := range archs {
		logrus.Info("checking " + arch)

		if _, ok := images[arch]; !ok {
			return errors.New("arch " + arch + "not found")
		}

		logrus.Info("passed, " + arch + " exists")
	}

	return nil
}

// dockerTag returns a map whose keys are the architecture of each image
// or an empty map if the tag is not found.
func dockerTag(ctx context.Context, org, repo, tag, registryURL string) (map[string]DockerImage, error) {
	url := registryURL + "/v2/repositories/" + org + "/" + repo + "/tags/" + tag

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	httpClient := ecmHTTP.NewClient(time.Second * 15)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to find docker tag \"%s\", unexpected status code: %d", tag, res.StatusCode)
	}

	var dt DockerTag
	if err := json.NewDecoder(res.Body).Decode(&dt); err != nil {
		return nil, err
	}

	images := make(map[string]DockerImage, len(dt.Images))
	for _, image := range dt.Images {
		images[image.Architecture] = image
	}

	return images, nil
}
