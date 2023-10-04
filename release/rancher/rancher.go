package rancher

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/rancher/ecm-distro-tools/docker"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	rancherImagesBaseURL     = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName    = "/rancher-images.txt"
	rancherHelmRepositoryURL = "https://releases.rancher.com/server-charts/latest/index.yaml"

	setKDMBranchReferencesScriptFile = "set-kdm-branch-references.sh"

	setKDMBranchReferencesScript = `#!/bin/bash
set -x

BRANCH_NAME=kdm-set-{{ .NewKDMBranch }}

cd {{ .RancherRepoDir }}
git remote add upstream https://github.com/rancher/rancher.git
git fetch upstream
git stash
git branch -D $BRANCH_NAME
git checkout -B $BRANCH_NAME upstream/{{.RancherBaseBranch}}
git clean -xfd

if [ "$(uname)" == "Darwin" ];then
	sed -i '' 's/NewSetting(\"kdm-branch\", \"{{ .CurrentKDMBranch }}\")/NewSetting(\"kdm-branch\", \"{{ .NewKDMBranch }}\")/' pkg/settings/setting.go
	sed -i '' 's/CATTLE_KDM_BRANCH={{ .CurrentKDMBranch }}/CATTLE_KDM_BRANCH={{ .NewKDMBranch }}/' package/Dockerfile
	sed -i '' 's/CATTLE_KDM_BRANCH={{ .CurrentKDMBranch }}/CATTLE_KDM_BRANCH={{ .NewKDMBranch }}/' Dockerfile.dapper
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then 
	sed -i 's/NewSetting("kdm-branch", "{{ .CurrentKDMBranch }}")/NewSetting("kdm-branch", "{{ .NewKDMBranch }}")/' pkg/settings/setting.go
	sed -i 's/CATTLE_KDM_BRANCH={{ .CurrentKDMBranch }}/CATTLE_KDM_BRANCH={{ .NewKDMBranch }}/' package/Dockerfile
	sed -i 's/CATTLE_KDM_BRANCH={{ .CurrentKDMBranch }}/CATTLE_KDM_BRANCH={{ .NewKDMBranch }}/' Dockerfile.dapper
fi

git add pkg/settings/setting.go
git add package/Dockerfile
git add Dockerfile.dapper

git commit --all --signoff -m "update kdm branch to {{ .NewKDMBranch }}"
git push --set-upstream origin $BRANCH_NAME`
)

type SetKDMBranchReferencesArgs struct {
	RancherRepoDir    string
	CurrentKDMBranch  string
	NewKDMBranch      string
	RancherBaseBranch string
}

type HelmIndex struct {
	Entries struct {
		Rancher []struct {
			AppVersion string `yaml:"appVersion"`
		} `yaml:"rancher"`
	} `yaml:"entries"`
}

func ListRancherImagesRC(tag string) (string, error) {
	downloadURL := rancherImagesBaseURL + tag + rancherImagesFileName
	imagesFile, err := rancherImages(downloadURL)
	if err != nil {
		return "", err
	}
	rcImages := nonMirroredRCImages(imagesFile)

	if len(rcImages) == 0 {
		return "There are none non-mirrored images still in rc form for tag " + tag, nil
	}

	output := "The following non-mirrored images for tag *" + tag + "* are still in RC form\n```\n"
	for _, image := range rcImages {
		output += image + "\n"
	}
	output += "```"

	return output, nil
}

func nonMirroredRCImages(images string) []string {
	var rcImages []string

	scanner := bufio.NewScanner(strings.NewReader(images))
	for scanner.Scan() {
		image := scanner.Text()
		if strings.Contains(image, "mirrored") {
			continue
		}
		if strings.Contains(image, "-rc") {
			rcImages = append(rcImages, image)
		}
	}
	return rcImages
}

func rancherImages(imagesURL string) (string, error) {
	httpClient := http.Client{Timeout: time.Second * 15}
	logrus.Debug("downloading: " + imagesURL)
	resp, err := httpClient.Get(imagesURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to download rancher-images.txt file, expected status code 200, got: " + strconv.Itoa(resp.StatusCode))
	}
	images, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(images), nil
}

func CheckRancherDockerImage(ctx context.Context, org, repo, tag string, archs []string) error {
	exists, err := docker.CheckImageArchs(ctx, org, repo, tag, archs)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("image tag doesn't exist or doesn't have all architectures")
	}
	return nil
}

func CheckHelmChartVersion(tag string) error {
	versions, err := rancherHelmChartVersions(rancherHelmRepositoryURL)
	if err != nil {
		return err
	}
	var foundVersion bool
	for _, version := range versions {
		logrus.Debug("checking version " + version)
		if tag == version {
			logrus.Info("found chart for version " + version)
			foundVersion = true
			break
		}
	}
	if !foundVersion {
		return errors.New("failed to find chart for rancher app version " + tag)
	}
	return nil
}

func rancherHelmChartVersions(repoURL string) ([]string, error) {
	httpClient := http.Client{Timeout: time.Second * 15}
	logrus.Debug("downloading: " + repoURL)
	resp, err := httpClient.Get(repoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to download index.yaml file, expected status code 200, got: " + strconv.Itoa(resp.StatusCode))
	}
	var helmIndex HelmIndex
	if err := yaml.NewDecoder(resp.Body).Decode(&helmIndex); err != nil {
		return nil, err
	}
	versions := make([]string, len(helmIndex.Entries.Rancher))
	for i, entry := range helmIndex.Entries.Rancher {
		versions[i] = entry.AppVersion
	}
	return versions, nil
}

func SetKDMBranchReferences(rancherRepoDir, rancherBaseBranch, currentKDMBranch, newKDMBranch string) error {
	logrus.Info("oi")
	if _, err := os.Stat(rancherRepoDir); err != nil {
		return err
	}

	logrus.Info("1")
	scriptPath := filepath.Join(rancherRepoDir, setKDMBranchReferencesScriptFile)
	f, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	logrus.Info("2")
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return err
	}
	logrus.Info("3")
	tmpl, err := template.New(setKDMBranchReferencesScriptFile).Parse(setKDMBranchReferencesScript)
	if err != nil {
		return err
	}
	logrus.Info("4")
	data := SetKDMBranchReferencesArgs{
		RancherRepoDir:    rancherRepoDir,
		CurrentKDMBranch:  currentKDMBranch,
		NewKDMBranch:      newKDMBranch,
		RancherBaseBranch: rancherBaseBranch,
	}
	logrus.Info("5")
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	logrus.Info("6")
	if _, err := ecmExec.RunCommand(rancherRepoDir, "bash", "./"+setKDMBranchReferencesScriptFile); err != nil {
		return err
	}

	return nil
}
