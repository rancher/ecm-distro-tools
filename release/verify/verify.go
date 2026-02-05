// Package verify stores the logic to validate the current release, if it's respecting the release process.
package verify

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v81/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
)

var (
	// rpmRegex matches the RKE2 RPM version format:
	// v<Major>.<Minor>.<Patch>+rke2r<Revision>.<Channel>.<Build>
	// Example: v1.23.6+rke2r1.stable.0
	// No need to support RCs as this will never receive an RC/Alpha versions
	rpmRegex = regexp.MustCompile(`^v\d{1,2}\.\d{1,2}\.\d{1,2}\+rke2r\d\.(testing|latest|stable)\.\d$`)

	// channelRegex matches the RKE2 RPM channel format:
	channelRegex = regexp.MustCompile(`.(testing|latest|stable)\.\d$`)
)

func GA(ctx context.Context, ghClient *github.Client, version string) error {
	switch {
	// The rpmRegex should be the first case to avoid RKE2 releases being pulled in for containing rke2r*
	case rpmRegex.MatchString(version):
		if err := verifyRPMGA(ctx, ghClient, version); err != nil {
			return err
		}
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

func verifyRPMGA(ctx context.Context, ghClient *github.Client, version string) error {
	// For RPM 'testing' GA releases it SHOULD have a 'testing' RC release
	if strings.Contains(version, "testing") {
		versionWithRC := appendRC1(version)
		found, err := checkRelease(
			ctx,
			ghClient,
			config.RancherGithubOrganization,
			config.RPMRepositoryName,
			versionWithRC,
		)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		return fmt.Errorf("previous RC testing RPM not found '%s'", versionWithRC)
	}

	// For RPM 'latest' releases it SHOULD have a 'testing' release
	if strings.Contains(version, "latest") {
		testingVersion := channelRegex.ReplaceAllString(version, "testing.0")
		found, err := checkRelease(
			ctx,
			ghClient,
			config.RancherGithubOrganization,
			config.RPMRepositoryName,
			testingVersion,
		)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		return fmt.Errorf("previous testing RPM not found for '%s'", testingVersion)
	}

	// For RPM 'stable' releases it SHOULD have a 'latest' release
	if strings.Contains(version, "stable") {
		latestVersion := channelRegex.ReplaceAllString(version, "latest.0")
		found, err := checkRelease(
			ctx,
			ghClient,
			config.RancherGithubOrganization,
			config.RPMRepositoryName,
			latestVersion,
		)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		return fmt.Errorf("previous latest RPM not found for '%s'", latestVersion)
	}
	return nil
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

	if res.StatusCode == http.StatusOK {
		return true, nil
	}

	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err
}
