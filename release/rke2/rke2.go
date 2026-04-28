package rke2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v85/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/docker"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
)

const (
	goDevURL           = "https://go.dev/dl/?mode=json"
	dockerHubTagsURL   = "https://hub.docker.com/v2/repositories/library/golang/tags"
	imageBuildBaseRepo = "image-build-base"
)

type goVersionRecord struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func ImageBuildBaseRelease(ctx context.Context, ghClient *github.Client, dryRun bool) error {
	versions, err := goVersions(goDevURL)
	if err != nil {
		return err
	}

	for _, version := range versions {
		logrus.Info("version: " + version.Version)
		if !version.Stable {
			logrus.Info("version " + version.Version + " is not stable")
			continue
		}
		goVersion := strings.Split(version.Version, "go")[1]

		// Dynamically find the Alpine version for this Go version.
		alpineVersion, err := alpineGoVersion(goVersion)
		if err != nil {
			return fmt.Errorf("failed to find a corresponding alpine version for go %s: %v", goVersion, err)
		}
		logrus.Infof("found alpine v%s for go v%s", alpineVersion, goVersion)

		alpineTag := goVersion + "-alpine" + alpineVersion

		if err := docker.CheckImageArchs(ctx, "library", "golang", alpineTag, []string{"amd64", "arm64", "s390x"}); err != nil {
			return fmt.Errorf("failed to check image archs for %s: %v", alpineTag, err)
		}

		imageBuildBaseTag := "v" + goVersion + "b1"
		logrus.Info("stripped version: " + imageBuildBaseTag)
		if _, _, err := ghClient.Repositories.GetReleaseByTag(ctx, "rancher", imageBuildBaseRepo, imageBuildBaseTag); err == nil {
			logrus.Info("release " + imageBuildBaseTag + " already exists")
			continue
		}
		logrus.Info("release " + imageBuildBaseTag + " doesn't exists, creating release")
		if dryRun {
			logrus.Info("dry run, release won't be created")
			logrus.Infof("Release:\n  Owner: rancher\n  Repo: %s\n  TagName: %s\n  Name: %s\n", imageBuildBaseRepo, imageBuildBaseTag, imageBuildBaseTag)
			return nil
		}
		release := &github.RepositoryRelease{
			TagName:    github.String(imageBuildBaseTag),
			Name:       github.String(imageBuildBaseTag),
			Prerelease: github.Bool(false),
		}
		if _, _, err := ghClient.Repositories.CreateRelease(ctx, "rancher", imageBuildBaseRepo, release); err != nil {
			return err
		}
		logrus.Info("created release for version: " + imageBuildBaseTag)
	}
	return nil
}

func CreateRef(ctx context.Context, client *github.Client, r *ecmConfig.RKE2Release, opts *repository.CreateRefOpts, rc bool) error {
	fmt.Println("validating tag")
	_, err := semver.NewVersion(opts.Tag)
	if err != nil {
		return errors.New("tag isn't a valid semver: " + opts.Tag)
	}

	name := r.NewK8sVersion + "+" + r.NewSuffix

	latestRC, err := release.LatestRC(ctx, opts.Owner, opts.Repo, r.NewK8sVersion, r.NewSuffix, client)
	if err != nil {
		return err
	}
	if latestRC == nil && !rc {
		return errors.New("couldn't find the latest RC")
	}
	if rc {
		latestRCNumber := 1
		if latestRC != nil {
			trimmedRCNumber, _, found := strings.Cut(strings.TrimPrefix(*latestRC, r.NewK8sVersion+"-rc"), "+rke2r")
			if !found {
				return errors.New("failed to parse rc number from " + *latestRC)
			}
			currentRCNumber, err := strconv.Atoi(trimmedRCNumber)
			if err != nil {
				return err
			}
			latestRCNumber = currentRCNumber + 1
		}
		name = r.NewK8sVersion + "-rc" + strconv.Itoa(latestRCNumber) + "+" + r.NewSuffix
	}

	opts.Tag = name

	fmt.Printf("create ref options: %+v\n", *opts)

	if r.DryRun {
		fmt.Println("dry run, skipping creating tag")
		return nil
	}
	createdRef, err := repository.CreateRef(ctx, client, opts)
	if err != nil {
		return err
	}

	fmt.Println("ref created: " + *createdRef.URL)
	return nil
}

// dockerHubResponse defines the structure for the Docker Hub API response.
type dockerHubResponse struct {
	Next    string `json:"next"`
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

// alpineGoVersion queries the Docker Hub API to find the Alpine version
// associated with a specific Go version.
func alpineGoVersion(goVersion string) (string, error) {
	// Compile regex to find a tag like "1.22.5-alpine3.20" and extract "3.20"
	re := regexp.MustCompile(fmt.Sprintf(`^%s-alpine(\d+\.\d+)$`, regexp.QuoteMeta(goVersion)))

	client := ecmHTTP.NewClient(time.Second * 15)
	url := dockerHubTagsURL

	for url != "" {
		res, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return "", errors.New("failed to query docker hub, status: " + res.Status)
		}

		var resp dockerHubResponse
		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			return "", err
		}

		for _, result := range resp.Results {
			matches := re.FindStringSubmatch(result.Name)
			if len(matches) > 1 {
				return matches[1], nil // Return the first captured group (the version number)
			}
		}
		url = resp.Next // Move to the next page
	}

	return "", errors.New("no matching alpine tag found for go version " + goVersion)
}

func goVersions(goDevURL string) ([]goVersionRecord, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	res, err := httpClient.Get(goDevURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get stable go versions")
	}

	var versions []goVersionRecord
	if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}
