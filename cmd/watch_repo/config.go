package main

import (
	"os"
)

type config struct {
	DroneRancherPrToken      string
	DroneRancherPublishToken string
	DroneK3sPrToken          string
	DroneK3sPublishToken     string
	GitHubToken              string
}

func newConfig(path string) (*config, error) {
	c := &config{
		DroneRancherPrToken:      os.Getenv("DRONE_RANCHER_PR_TOKEN"),
		DroneRancherPublishToken: os.Getenv("DRONE_RANCHER_PUBLISH_TOKEN"),
		DroneK3sPrToken:          os.Getenv("DRONE_K3S_PR_TOKEN"),
		DroneK3sPublishToken:     os.Getenv("DRONE_K3S_PUBLISH_TOKEN"),
		GitHubToken:              os.Getenv("GITHUB_TOKEN"),
	}

	return c, nil
}
