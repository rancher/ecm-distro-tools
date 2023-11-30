package rke2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/docker"
	"github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/sirupsen/logrus"
)

const (
	imageBuildDefaultBranch           = "master"
	goDevURL                          = "https://go.dev/dl/?mode=json"
	imageBuildBaseRepo                = "image-build-base"
	getHardenedBuildTagScriptFileName = "get_hardened_build_tag.sh"
	updateImageBuildScriptFileName    = "update_image_build_base.sh"
	getHardenedBuildTagScript         = `#!/bin/sh
set -e
REPO_PATH="{{ .RepoPath }}"
DEFAULT_BRANCH="{{ .DefaultBranch }}"
cd "${REPO_PATH}"
git stash >/dev/null
git switch "${DEFAULT_BRANCH}" >/dev/null
git pull origin "${DEFAULT_BRANCH}" >/dev/null
grep -o -e "hardened-build-base:.*$" Dockerfile
`
	updateImageBuildScript = `#!/bin/sh
set -e
DRY_RUN={{ .DryRun }}
NEW_TAG="{{ .NewTag }}"
REPO_PATH="{{ .RepoPath }}"
BRANCH_NAME="{{ .BranchName }}"
CURRENT_TAG="{{ .CurrentTag }}"
DEFAULT_BRANCH="{{ .DefaultBranch }}"
echo "dry run: ${DRY_RUN}"
echo "current tag: ${CURRENT_TAG}"
echo "branch name: ${BRANCH_NAME}"
echo "default branch: ${DEFAULT_BRANCH}"

echo "navigating to the repo dir"
cd "${REPO_PATH}"
echo "new tag: ${NEW_TAG}"
echo "creating local branch"
git checkout -B "${BRANCH_NAME}" "${DEFAULT_BRANCH}"
git clean -xfd
OS=$(uname -s)
case ${OS} in
Darwin)
	sed -i '' "s/hardened-build-base:${CURRENT_TAG}/hardened-build-base:${NEW_TAG}/" Dockerfile
	sed -i '' "s/hardened-build-base:${CURRENT_TAG}/hardened-build-base:${NEW_TAG}/" .drone.yml
	;;
Linux)
	sed -i "s/hardened-build-base:${CURRENT_TAG}/hardened-build-base:${NEW_TAG}/" Dockerfile
	sed -i "s/hardened-build-base:${CURRENT_TAG}/hardened-build-base:${NEW_TAG}/" .drone.yml
	;;
*)
	>&2 echo "$(OS) not supported yet"
	exit 1
	;;
esac
git add Dockerfile
git add .drone.yml
git commit -m "update hardened-build-base to ${NEW_TAG}"
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin ${BRANCH_NAME}
fi`
)

type UpdateImageBuildArgs struct {
	RepoName      string
	RepoOwner     string
	BranchName    string
	DryRun        bool
	RepoPath      string
	NewTag        string
	CurrentTag    string
	DefaultBranch string
}

var ImageBuildRepos map[string]bool = map[string]bool{
	"image-build-dns-nodecache":                    true,
	"image-build-k8s-metrics-server":               true,
	"image-build-sriov-cni":                        true,
	"image-build-ib-sriov-cni":                     true,
	"image-build-sriov-network-device-plugin":      true,
	"image-build-sriov-network-resources-injector": true,
	"image-build-calico":                           true,
	"image-build-cni-plugins":                      true,
	"image-build-whereabouts":                      true,
	"image-build-flannel":                          true,
	"image-build-etcd":                             true,
	"image-build-containerd":                       true,
	"image-build-runc":                             true,
	"image-build-multus":                           true,
	"image-build-rke2-cloud-provider":              true,
}

type goVersionRecord struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func ImageBuildBaseRelease(ctx context.Context, ghClient *github.Client, alpineVersion string, dryRun bool) error {
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
		version := strings.Split(version.Version, "go")[1]
		alpineTag := version + "-alpine" + alpineVersion

		if err := docker.CheckImageArchs(ctx, "library", "golang", alpineTag, []string{"amd64", "arm64", "s390x"}); err != nil {
			return err
		}

		imageBuildBaseTag := "v" + version + "b1"
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

func UpdateImageBuild(ctx context.Context, ghClient *github.Client, repo, owner, repoPath, workingDir, newTag string, dryRun, createPR bool) error {
	if _, ok := ImageBuildRepos[repo]; !ok {
		return errors.New("invalid repo, please review the supported repos list")
	}
	branchName := "update-to-" + newTag
	data := UpdateImageBuildArgs{
		RepoName:      repo,
		RepoOwner:     owner,
		BranchName:    branchName,
		DryRun:        dryRun,
		RepoPath:      repoPath,
		NewTag:        newTag,
		DefaultBranch: imageBuildDefaultBranch,
	}
	currentTagOutput, err := exec.RunTemplatedScript(workingDir, getHardenedBuildTagScriptFileName, getHardenedBuildTagScript, data)
	if err != nil {
		return err
	}
	logrus.Info(currentTagOutput)
	currentTag := strings.TrimPrefix(strings.TrimSpace(currentTagOutput), "hardened-build-base:")
	data.CurrentTag = currentTag
	updateFilesOutput, err := exec.RunTemplatedScript(workingDir, updateImageBuildScriptFileName, updateImageBuildScript, data)
	if err != nil {
		return err
	}
	logrus.Info(updateFilesOutput)
	if createPR {
		prName := "Update hardened build base to " + newTag
		logrus.Info("preparing PR")
		logrus.Info("PR:\n  Name: " + prName + "\n  From: " + owner + ":" + branchName + "\n  To " + owner + ":" + imageBuildDefaultBranch)
		if dryRun {
			logrus.Info("dry run, PR will not be created")
			return nil
		}
		logrus.Info("creating pr")
		if err := createPRFromRancher(ctx, ghClient, prName, branchName, owner, repo, imageBuildDefaultBranch); err != nil {
			return err
		}
	}
	return nil
}

func createPRFromRancher(ctx context.Context, ghClient *github.Client, title, branchName, owner, repo, baseBranch string) error {
	pull := &github.NewPullRequest{
		Title:               &title,
		Base:                github.String(baseBranch),
		Head:                github.String(owner + ":" + branchName),
		MaintainerCanModify: github.Bool(true),
	}
	_, _, err := ghClient.PullRequests.Create(ctx, owner, repo, pull)
	return err
}
