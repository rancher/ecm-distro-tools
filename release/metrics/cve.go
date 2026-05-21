package metrics

import (
	"context"
	"fmt"

	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

const (
	reportsFolder = "reports"
)

type Reports struct {
	Harvester          []CVE
	K3s                []CVE
	Longhorn           []CVE
	Observability      []CVE
	ObservabilityAgent []CVE
	Rancher            []CVE
	RKE2               []CVE
}

type CVE struct {
	Image           string
	Release         string
	PackageName     string
	PackageVersion  string
	Type            string
	VulnerabilityID string
	Severity        string
	URL             string
	Target          string
	PatchedVersion  string
	Mirrored        bool
	Status          string
	Justification   string
}

func CVEsMetrics(ctx context.Context, client *github.Client) error {
	opts := &github.RepositoryContentGetOptions{
		Ref: "main",
	}

	// GetContents returns (fileContent, directoryContent, response, error).
	// But since we are querying a directory(reports), fileContent will be nil, and directoryContent will be populated.
	_, directoryContent, _, err := client.Repositories.GetContents(ctx, config.RancherGithubOrganization, config.ImageScanningRepositoryName, reportsFolder, opts)
	if err != nil {
		return fmt.Errorf("failed to get contents for %s/%s at path %s: %w", config.RancherGithubOrganization, config.ImageScanningRepositoryName, reportsFolder, err)
	}

	for _, item := range directoryContent {
		switch item.GetType() {
		case "file":
			fmt.Printf("File: %s\n", item.GetName())
			//fmt.Printf("\tURL: %s\n", item.GetDownloadURL())
		}
	}

	return nil
}
