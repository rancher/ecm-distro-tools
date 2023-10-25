package rancher

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/docker"
	"github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	rancherImagesBaseURL     = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName    = "/rancher-images.txt"
	rancherHelmRepositoryURL = "https://releases.rancher.com/server-charts/latest/index.yaml"

	setKDMBranchReferencesScriptFileName = "set_kdm_branch_references.sh"
	setChartReferencesScriptFileName     = `set_chart_references.sh`
	cloneCheckoutRancherScript           = `#!/bin/sh
set -e

BRANCH_NAME={{ .BranchName }}
DRY_RUN={{ .DryRun }}

cd {{ .RancherRepoPath }}
git remote -v | grep -w upstream || git remote add upstream https://github.com/rancher/rancher.git
git fetch upstream
git stash
if [ 'git branch --list ${BRANCH_NAME}' ]
then
   echo "Branch name ${BRANCH_NAME} already exists, deleting"
   git branch -D ${BRANCH_NAME}
fi
git checkout -B ${BRANCH_NAME} upstream/{{.RancherBaseBranch}}
git clean -xfd`
	setKDMBranchReferencesScript = `
echo "\nCurrent set KDM Branch: $(cat Dockerfile.dapper | grep CATTLE_KDM_BRANCH)"
echo "\nUpdating\n    - pkg/settings/setting.go\n    - package/Dockerfile\n    - Dockerfile.dapper"

OS=$(uname -s)
case ${OS} in
Darwin)
	sed -i '' 's/NewSetting("kdm-branch", ".*")/NewSetting("kdm-branch", "{{ .NewBranch }}")/' pkg/settings/setting.go
	sed -i '' 's/ARG\ CATTLE_KDM_BRANCH=.*$/ARG\ CATTLE_KDM_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i '' 's/CATTLE_KDM_BRANCH=.*$/CATTLE_KDM_BRANCH={{ .NewBranch }}/' Dockerfile.dapper
	;;
Linux)
	sed -i 's/NewSetting("kdm-branch", ".*")/NewSetting("kdm-branch", "{{ .NewBranch }}")/' pkg/settings/setting.go
	sed -i 's/ARG\ CATTLE_KDM_BRANCH=.*$/ARG\ CATTLE_KDM_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i 's/CATTLE_KDM_BRANCH=.*$/CATTLE_KDM_BRANCH={{ .NewBranch }}/' Dockerfile.dapper
	;;
*)
	>&2 echo "$(OS) not supported yet"
	exit 1
	;;
esac
git add pkg/settings/setting.go
git add package/Dockerfile
git add Dockerfile.dapper
git commit --all --signoff -m "update kdm branch to {{ .NewBranch }}"`
	setChartBranchReferencesScript = `
OS=$(uname -s)
case ${OS} in
Darwin)
	sed -i '' 's/NewSetting(\"chart-default-branch\", \"{{ .CurrentBranch }}\")/NewSetting(\"chart-default-branch\", \"{{ .NewBranch }}\")/' pkg/settings/setting.go
	sed -i '' 's/SYSTEM_CHART_DEFAULT_BRANCH={{ .CurrentBranch }}/SYSTEM_CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i '' 's/CHART_DEFAULT_BRANCH={{ .CurrentBranch }}/CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i '' 's/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .CurrentBranch }}"}/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .NewBranch }}"}/' scripts/package-env
	;;
Linux)
	sed -i 's/NewSetting("chart-default-branch", "{{ .CurrentBranch }}")/NewSetting("chart-default-branch", "{{ .NewBranch }}")/' pkg/settings/setting.go
	sed -i 's/SYSTEM_CHART_DEFAULT_BRANCH={{ .CurrentBranch }}/SYSTEM_CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i 's/CHART_DEFAULT_BRANCH={{ .CurrentBranch }}/CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i 's/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .CurrentBranch }}"}/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .NewBranch }}"}/' scripts/package-env
	;;
*)
	>&2 echo "$(OS) not supported yet"
	exit 1
	;;
esac

git add pkg/settings/setting.go
git add package/Dockerfile
git add scripts/package-env
git commit --all --signoff -m "update chart branch references to {{ .NewBranch }}"`
	pushChangesScript = `
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin ${BRANCH_NAME}
fi`
)

type SetBranchReferencesArgs struct {
	RancherRepoPath   string
	CurrentBranch     string
	NewBranch         string
	RancherBaseBranch string
	BranchName        string
	DryRun            bool
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
	httpClient := ecmHTTP.NewClient(time.Second * 15)
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
	return docker.CheckImageArchs(ctx, org, repo, tag, archs)
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
	httpClient := ecmHTTP.NewClient(time.Second * 15)
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

func SetKDMBranchReferences(ctx context.Context, forkPath, rancherBaseBranch, newKDMBranch, githubUser, githubToken string, createPR, dryRun bool) error {
	branchName := "kdm-set-" + newKDMBranch
	data := SetBranchReferencesArgs{
		RancherRepoPath:   forkPath,
		NewBranch:         newKDMBranch,
		RancherBaseBranch: rancherBaseBranch,
		DryRun:            dryRun,
		BranchName:        branchName,
	}

	script := cloneCheckoutRancherScript + setKDMBranchReferencesScript + pushChangesScript
	logrus.Info("running update files and apply updates script...")
	output, err := exec.RunTemplatedScript(forkPath, setKDMBranchReferencesScriptFileName, script, data)
	if err != nil {
		return err
	}
	logrus.Info(output)

	if createPR {
		prName := "Update KDM to " + newKDMBranch
		logrus.Info("creating PR: " + prName)
		if dryRun {
			logrus.Info("dry run, PR will not be created")
			return nil
		}
		ghClient := repository.NewGithub(ctx, githubToken)

		if err := createPRFromRancher(ctx, rancherBaseBranch, prName, branchName, githubUser, ghClient); err != nil {
			return err
		}
	}
	return nil
}

func SetChartBranchReferences(ctx context.Context, forkPath, rancherBaseBranch, currentBranch, newBranch, forkOwner, githubToken string, createPR, dryRun bool) error {
	branchName := "charts-set-" + newBranch
	data := SetBranchReferencesArgs{
		RancherRepoPath:   forkPath,
		CurrentBranch:     currentBranch,
		NewBranch:         newBranch,
		RancherBaseBranch: rancherBaseBranch,
		DryRun:            dryRun,
		BranchName:        branchName,
	}
	script := cloneCheckoutRancherScript + setChartBranchReferencesScript + pushChangesScript
	if _, err := exec.RunTemplatedScript(forkPath, setChartReferencesScriptFileName, script, data); err != nil {
		return err
	}

	if createPR && !dryRun {
		ghClient := repository.NewGithub(ctx, githubToken)

		if err := createPRFromRancher(ctx, rancherBaseBranch, "Update charts branch references to "+newBranch, branchName, forkOwner, ghClient); err != nil {
			return err
		}
	}

	return nil
}

func createPRFromRancher(ctx context.Context, rancherBaseBranch, title, branchName, forkOwner string, ghClient *github.Client) error {
	const repo = "rancher"
	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return err
	}
	pull := &github.NewPullRequest{
		Title:               github.String(title),
		Base:                github.String(rancherBaseBranch),
		Head:                github.String(forkOwner + ":" + branchName),
		MaintainerCanModify: github.Bool(true),
	}
	_, _, err = ghClient.PullRequests.Create(ctx, org, repo, pull)

	return err
}
