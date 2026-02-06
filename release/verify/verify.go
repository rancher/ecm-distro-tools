// Package verify stores the logic to validate the current release, if it's respecting the release process.
package verify

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v81/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

func GA(ctx context.Context, ghClient *github.Client, version string) error {
	switch {
	// The rpmRegex should be the first case to avoid RKE2 releases being pulled in for containing rke2r*
	case strings.Contains(version, "rke2"):
		if err := verifyRKE2(ctx, ghClient, version); err != nil {
			return err
		}

	case strings.Contains(version, "k3s"):
		if err := verifyK3s(ctx, ghClient, version); err != nil {
			return err
		}
	// Rancher versions will be handled in default
	default:
		if err := verifyRancher(ctx, ghClient, version); err != nil {
			return err
		}
	}
	return nil
}

func verifyRKE2(ctx context.Context, ghClient *github.Client, version string) error {
	versionWithRC := appendRC1(version)
	found, err := checkRelease(
		ctx,
		ghClient,
		config.RancherGithubOrganization,
		config.RKE2RepositoryName,
		versionWithRC,
	)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	return fmt.Errorf("previous k3s RC not found '%s'", versionWithRC)
}

func verifyK3s(ctx context.Context, ghClient *github.Client, version string) error {
	versionWithRC := appendRC1(version)
	found, err := checkRelease(
		ctx,
		ghClient,
		config.K3sGithubOrganization,
		config.K3sRepositoryName,
		versionWithRC,
	)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	return fmt.Errorf("previous k3s RC not found '%s'", versionWithRC)
}

func verifyRancher(ctx context.Context, ghClient *github.Client, version string) error {
	versionWithRC := appendRC1(version)
	found, err := checkRelease(
		ctx,
		ghClient,
		config.RancherGithubOrganization,
		config.RancherRepositoryName,
		versionWithRC,
	)

	if err != nil {
		return err
	}
	if found {
		return nil
	}

	return fmt.Errorf("previous rancher RC not found '%s'", versionWithRC)
}

func appendRC1(version string) string {
	switch {
	// In this switch we can omit the case for RPM because
	// it would be the same process for RPMs, K3s and RKE2 releases.
	case strings.Contains(version, "rke2"), strings.Contains(version, "k3s"):
		return strings.ReplaceAll(version, "+", "-rc1+")
	default:
		return version + "-rc1"
	}
}

// checkRelease will verify if the given release exists
func checkRelease(ctx context.Context, client *github.Client, owner, repo, version string) (bool, error) {
	_, res, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, version)

	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
