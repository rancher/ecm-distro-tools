package rke2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v85/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/docker"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
)

const (
	goDevURL             = "https://go.dev/dl/?mode=json"
	dockerHubTagsURL     = "https://hub.docker.com/v2/repositories/library/golang/tags"
	imageBuildBaseRepo   = "image-build-base"
	updateRKE2ScriptName = "update_rke2_references.sh"

	updateRke2ReferencesScript = `#!/bin/bash
set -ex
OS=$(uname -s)
DRY_RUN={{ .RKE2.DryRun }}
BRANCH_NAME={{ .RKE2.NewK8sVersion }}-{{ .RKE2.NewSuffix }}
cd {{ .RKE2.Workspace }}
ls | grep -w rke2 || git clone "git@github.com:{{ .User.GithubUsername }}/rke2.git"
cd {{ .RKE2.Workspace }}/rke2
git remote -v | grep -w upstream || git remote add upstream https://github.com/{{ .RKE2.RKE2RepoOwner }}/{{ .RKE2.RKE2RepoName }}.git
git fetch upstream
git stash
git branch -D "${BRANCH_NAME}" &>/dev/null || true
git checkout -B "${BRANCH_NAME}" upstream/{{ .RKE2.ReleaseBranch }}
git clean -xfd

case ${OS} in
Darwin)
	# go.mod: Go version
	sed -Ei '' "s/^go [^ ]*/go {{ .RKE2.NewGoVersion }}/" go.mod
	# go.mod: k3s-io/kubernetes fork replace directives
	sed -Ei '' "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .RKE2.OldK8sVersion "." "\\." }}-{{ .RKE2.K3sSuffix }}|{{ replaceAll .RKE2.NewK8sVersion "." "\\." }}-{{ .RKE2.K3sSuffix }}|g" go.mod
	# scripts/version.sh
	sed -Ei '' "s|KUBERNETES_VERSION:-[^}]*|KUBERNETES_VERSION:-{{ .RKE2.NewK8sVersion }}|" scripts/version.sh
	sed -Ei '' "s|KUBERNETES_IMAGE_TAG:-[^}]*|KUBERNETES_IMAGE_TAG:-{{ .NewKubernetesImageTag }}|" scripts/version.sh
	# Dockerfile + Dockerfile.windows: hardened-build-base Go version
	sed -Ei '' "s|rancher/hardened-build-base:v[^b]*b[0-9]*|rancher/hardened-build-base:v{{ .RKE2.NewGoVersion }}b1|g" Dockerfile Dockerfile.windows
	# Dockerfile: hardened-kubernetes image tag
	sed -Ei '' "s|rancher/hardened-kubernetes:[^ ]*|rancher/hardened-kubernetes:{{ .NewKubernetesImageTag }}|" Dockerfile
	# Dockerfile.windows: build-env linux kubectl version
	sed -Ei '' "s/^RUN KUBECTL_VERSION=v[0-9][0-9.]*/RUN KUBECTL_VERSION={{ .RKE2.NewK8sVersion }}/" Dockerfile.windows
	# Dockerfile.windows: build-env linux kubectl SHA (trailing ";;" is the discriminator)
	sed -Ei '' "s/KUBECTL_SHA256=\"[a-f0-9]*\" ;;/KUBECTL_SHA256=\"{{ .LinuxKubectlSHA }}\" ;;/" Dockerfile.windows
	# Dockerfile.windows: Windows binary case statement version key
	sed -Ei '' "s/{{ replaceAll .RKE2.OldK8sVersion "." "\\." }})/{{ .RKE2.NewK8sVersion }})/" Dockerfile.windows
	# Dockerfile.windows: Windows kubectl SHA (trailing "&&" is the discriminator)
	sed -Ei '' "s/KUBECTL_SHA256=\"[a-f0-9]*\" &&/KUBECTL_SHA256=\"{{ .WindowsKubectlSHA }}\" \&\&/" Dockerfile.windows
	# Dockerfile.windows: Windows kubelet + kube-proxy SHAs (each appears only once)
	sed -Ei '' "s/KUBELET_SHA256=\"[a-f0-9]*/KUBELET_SHA256=\"{{ .WindowsKubeletSHA }}/" Dockerfile.windows
	sed -Ei '' "s/KUBE_PROXY_SHA256=\"[a-f0-9]*/KUBE_PROXY_SHA256=\"{{ .WindowsKubeProxySHA }}/" Dockerfile.windows
	;;
Linux)
	sed -Ei "s/^go [^ ]*/go {{ .RKE2.NewGoVersion }}/" go.mod
	sed -Ei "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .RKE2.OldK8sVersion "." "\\." }}-{{ .RKE2.K3sSuffix }}|{{ replaceAll .RKE2.NewK8sVersion "." "\\." }}-{{ .RKE2.K3sSuffix }}|g" go.mod
	sed -Ei "s|KUBERNETES_VERSION:-[^}]*|KUBERNETES_VERSION:-{{ .RKE2.NewK8sVersion }}|" scripts/version.sh
	sed -Ei "s|KUBERNETES_IMAGE_TAG:-[^}]*|KUBERNETES_IMAGE_TAG:-{{ .NewKubernetesImageTag }}|" scripts/version.sh
	sed -Ei "s|rancher/hardened-build-base:v[^b]*b[0-9]*|rancher/hardened-build-base:v{{ .RKE2.NewGoVersion }}b1|g" Dockerfile Dockerfile.windows
	sed -Ei "s|rancher/hardened-kubernetes:[^ ]*|rancher/hardened-kubernetes:{{ .NewKubernetesImageTag }}|" Dockerfile
	sed -Ei "s/^RUN KUBECTL_VERSION=v[0-9][0-9.]*/RUN KUBECTL_VERSION={{ .RKE2.NewK8sVersion }}/" Dockerfile.windows
	sed -Ei "s/KUBECTL_SHA256=\"[a-f0-9]*\" ;;/KUBECTL_SHA256=\"{{ .LinuxKubectlSHA }}\" ;;/" Dockerfile.windows
	sed -Ei "s/{{ replaceAll .RKE2.OldK8sVersion "." "\\." }})/{{ .RKE2.NewK8sVersion }})/" Dockerfile.windows
	sed -Ei "s/KUBECTL_SHA256=\"[a-f0-9]*\" &&/KUBECTL_SHA256=\"{{ .WindowsKubectlSHA }}\" \&\&/" Dockerfile.windows
	sed -Ei "s/KUBELET_SHA256=\"[a-f0-9]*/KUBELET_SHA256=\"{{ .WindowsKubeletSHA }}/" Dockerfile.windows
	sed -Ei "s/KUBE_PROXY_SHA256=\"[a-f0-9]*/KUBE_PROXY_SHA256=\"{{ .WindowsKubeProxySHA }}/" Dockerfile.windows
	;;
*)
	>&2 echo "${OS} not supported yet"
	exit 1
	;;
esac

go mod tidy

git add go.mod go.sum scripts/version.sh Dockerfile Dockerfile.windows
git commit --signoff -m "Update to {{ .RKE2.NewK8sVersion }}-{{ .RKE2.NewSuffix }}"
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}"
fi`
)

type goVersionRecord struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type UpdateScriptVars struct {
	RKE2                  *ecmConfig.RKE2Release
	User                  *ecmConfig.User
	NewKubernetesImageTag string // fetched from rancher/image-build-kubernetes releases
	LinuxKubectlSHA       string // dl.k8s.io linux/amd64 kubectl SHA256
	WindowsKubectlSHA     string // dl.k8s.io windows/amd64 kubectl.exe SHA256
	WindowsKubeletSHA     string // dl.k8s.io windows/amd64 kubelet.exe SHA256
	WindowsKubeProxySHA   string // dl.k8s.io windows/amd64 kube-proxy.exe SHA256
}

// UpdateRKE2References updates k8s, k3s and Go references in a local RKE2
// checkout and optionally opens a pull request.
func UpdateRKE2References(ctx context.Context, ghClient *github.Client, r *ecmConfig.RKE2Release, u *ecmConfig.User) error {
	if err := updateRKE2ReferencesAndPush(ctx, ghClient, r, u); err != nil {
		return err
	}

	if r.DryRun {
		fmt.Println("dry run, skipping creating rke2 update references PR")
		return nil
	}

	return createRKE2ReferencesPR(ctx, ghClient, r, u)
}

func updateRKE2ReferencesAndPush(ctx context.Context, ghClient *github.Client, r *ecmConfig.RKE2Release, u *ecmConfig.User) error {
	if err := release.SetWorkspace(r.Workspace); err != nil {
		return err
	}

	// Default go.mod k3s fork suffix when not explicitly configured.
	if r.K3sSuffix == "" {
		r.K3sSuffix = "k3s1"
	}

	fmt.Println("getting k8s go version")
	goVersion, err := release.KubernetesGoVersion(ctx, ghClient, r.NewK8sVersion)
	if err != nil {
		return err
	}
	r.NewGoVersion = goVersion

	fmt.Println("getting kubernetes image tag from rancher/image-build-kubernetes")
	kubernetesImageTag, err := getKubernetesImageTag(ctx, ghClient, r.NewK8sVersion, r.NewSuffix)
	if err != nil {
		return err
	}

	fmt.Println("fetching linux/amd64 kubectl SHA256")
	linuxKubectlSHA, err := fetchSHA(fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl.sha256", r.NewK8sVersion,
	))
	if err != nil {
		return err
	}

	fmt.Println("fetching windows/amd64 binary SHA256s")
	windowsKubectlSHA, err := fetchSHA(fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/windows/amd64/kubectl.exe.sha256", r.NewK8sVersion,
	))
	if err != nil {
		return err
	}

	windowsKubeletSHA, err := fetchSHA(fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/windows/amd64/kubelet.exe.sha256", r.NewK8sVersion,
	))
	if err != nil {
		return err
	}

	windowsKubeProxySHA, err := fetchSHA(fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/windows/amd64/kube-proxy.exe.sha256", r.NewK8sVersion,
	))
	if err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"replaceAll": strings.ReplaceAll,
	}

	fmt.Println("creating update rke2 references script template")

	scriptVars := UpdateScriptVars{
		RKE2:                  r,
		User:                  u,
		NewKubernetesImageTag: kubernetesImageTag,
		LinuxKubectlSHA:       linuxKubectlSHA,
		WindowsKubectlSHA:     windowsKubectlSHA,
		WindowsKubeletSHA:     windowsKubeletSHA,
		WindowsKubeProxySHA:   windowsKubeProxySHA,
	}

	updateScriptOut, err := ecmExec.RunTemplatedScript(r.Workspace, updateRKE2ScriptName, updateRke2ReferencesScript, funcMap, scriptVars)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)

	return nil
}

func createRKE2ReferencesPR(ctx context.Context, ghClient *github.Client, r *ecmConfig.RKE2Release, u *ecmConfig.User) error {
	pull := &github.NewPullRequest{
		Title:               new(fmt.Sprintf("[%s] Update to %s-%s and Go %s", r.ReleaseBranch, r.NewK8sVersion, r.NewSuffix, r.NewGoVersion)),
		Base:                new(r.ReleaseBranch),
		Head:                new(u.GithubUsername + ":" + r.NewK8sVersion + "-" + r.NewSuffix),
		MaintainerCanModify: new(true),
	}

	_, _, err := ghClient.PullRequests.Create(ctx, r.RKE2RepoOwner, r.RKE2RepoName, pull)
	return err
}

// getKubernetesImageTag queries the rancher/image-build-kubernetes GitHub
// releases and returns the first (most recent) tag that contains both the
// Kubernetes version and the RKE2 suffix, e.g. "v1.36.2-rke2r1-build20260612".
func getKubernetesImageTag(ctx context.Context, ghClient *github.Client, k8sVersion, suffix string) (string, error) {
	searchStr := k8sVersion + "-" + suffix

	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		releases, resp, err := ghClient.Repositories.ListReleases(ctx, config.RancherGithubOrganization, config.ImageBuildKubernetesRepositoryName, opts)
		if err != nil {
			return "", fmt.Errorf("listing %s/%s releases: %w", config.RancherGithubOrganization, config.ImageBuildKubernetesRepositoryName, err)
		}

		for _, r := range releases {
			if r.TagName != nil && strings.Contains(*r.TagName, searchStr) {
				return *r.TagName, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return "", fmt.Errorf("no %s/%s release found containing %q", config.RancherGithubOrganization, config.ImageBuildKubernetesRepositoryName, searchStr)
}

// fetchSHA downloads a .sha256 file and returns the hex digest.
// Handles both bare-hash files ("abc123...") and "hash  filename" formatted files.
func fetchSHA(url string) (string, error) {
	client := ecmHTTP.NewClient(30 * time.Second)

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching SHA from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP %d fetching SHA from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading SHA response from %s: %w", url, err)
	}

	parts := strings.Fields(strings.TrimSpace(string(body)))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty SHA response from %s", url)
	}

	return parts[0], nil
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
			TagName:    new(imageBuildBaseTag),
			Name:       new(imageBuildBaseTag),
			Prerelease: new(false),
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
