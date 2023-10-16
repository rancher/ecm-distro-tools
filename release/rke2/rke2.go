package rke2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
)

const (
	goDevURL           = "https://go.dev/dl/?mode=json"
	githubApiURL       = "https://api.github.com/"
	imageBuildBaseRepo = "image-build-base"
)

type goVersionRecord struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type GithubRelease struct {
	Name string `json:"name"`
}

func ImageBuildBaseRelease(ctx context.Context, ghClient *github.Client) error {
	versions, err := goVersions(goDevURL)
	if err != nil {
		return err
	}
	imageBuildBaseOrg, err := repository.OrgFromRepo(imageBuildBaseRepo)
	if err != nil {
		return err
	}
	for _, version := range versions {
		logrus.Info("version: " + version.Version)
		if !version.Stable {
			logrus.Info("version " + version.Version + " is not stable")
			continue
		}
		v := "v" + strings.Split(version.Version, "go")[1] + "b1"
		logrus.Info("stripped version: " + v)
		_, _, err := ghClient.Repositories.GetReleaseByTag(ctx, imageBuildBaseOrg, imageBuildBaseRepo, v)
		if err == nil {
			logrus.Info("release " + v + " already exists")
			continue
		}
		logrus.Info("release " + v + " doesn't exists, creating release")
		_, _, err = ghClient.Repositories.CreateRelease(ctx, imageBuildBaseOrg, imageBuildBaseRepo, &github.RepositoryRelease{
			TagName:    github.String(v),
			Name:       github.String(v),
			Prerelease: github.Bool(false),
		})
		if err != nil {
			return err
		}
		logrus.Info("created release for version: " + v)
	}
	return nil
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
