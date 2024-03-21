package rancher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v39/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	"github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

const (
	rancherOrg               = "rancher"
	rancherRepo              = rancherOrg
	rancherImagesBaseURL     = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName    = "/rancher-images.txt"
	rancherHelmRepositoryURL = "https://releases.rancher.com/server-charts/latest/index.yaml"
	rancherArtifactsListURL  = "https://prime-artifacts.s3.amazonaws.com"
	rancherArtifactsBaseURL  = "https://prime.ribs.rancher.io"

	setKDMBranchReferencesScriptFileName = "set_kdm_branch_references.sh"
	setChartReferencesScriptFileName     = `set_chart_references.sh`
	navigateCheckoutRancherScript        = `#!/bin/sh
set -e

BRANCH_NAME={{ .BranchName }}
DRY_RUN={{ .DryRun }}
echo "branch name: ${BRANCH_NAME}"
echo "dry run: ${DRY_RUN}"

echo "navigating into the rancher repo"
cd {{ .RancherRepoPath }}
echo "adding upstream remote if not exists"
git remote -v | grep -w upstream || git remote add upstream https://github.com/rancher/rancher.git
echo "fetching upstream"
git fetch upstream
echo "stashing local changes"
git stash
echo "if local branch already exists, delete it"
git branch -D ${BRANCH_NAME} &>/dev/null || true
echo "creating local branch"
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
	sed -i '' 's/NewSetting("chart-default-branch", ".*")/NewSetting("chart-default-branch", "{{ .NewBranch }}")/' pkg/settings/setting.go
	sed -i '' 's/ARG\ SYSTEM_CHART_DEFAULT_BRANCH=.*$/ARG\ SYSTEM_CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i '' 's/ARG\ CHART_DEFAULT_BRANCH=.*$/ARG\ CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i '' 's/{SYSTEM_CHART_DEFAULT_BRANCH:-".*"}/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .NewBranch }}"}/' scripts/package-env
	;;
Linux)
	sed -i 's/NewSetting("chart-default-branch", ".*")/NewSetting("chart-default-branch", "{{ .NewBranch }}")/' pkg/settings/setting.go
	sed -i 's/ARG\ SYSTEM_CHART_DEFAULT_BRANCH=.*$/ARG\ SYSTEM_CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i 's/ARG\ CHART_DEFAULT_BRANCH=.*$/ARG\ CHART_DEFAULT_BRANCH={{ .NewBranch }}/' package/Dockerfile
	sed -i 's/{SYSTEM_CHART_DEFAULT_BRANCH:-".*"}/{SYSTEM_CHART_DEFAULT_BRANCH:-"{{ .NewBranch }}"}/' scripts/package-env
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

const templateCheckRCDevDeps = `{{- define "componentsFile" -}}
# Images with -rc
{{range .RancherImages}}
* {{ .Content }} ({{ .File }}, line {{ .Line }})
{{- end}}

# Components with -rc
{{range .FilesWithRC}}
* {{ .Content }} ({{ .File }}, line {{ .Line }})
{{- end}}

# Min version components with -rc
{{range .MinFilesWithRC}}
* {{ .Content }} ({{ .File }}, line {{ .Line }})
{{- end}} 

# KDM References with dev branch
{{range .KDMWithDev}}
* {{ .Content }} ({{ .File }}, line {{ .Line }})
{{- end}}

# Chart References with dev branch
{{range .ChartsWithDev}}
* {{ .Content }} ({{ .File }}, line {{ .Line }})
{{- end}}
{{ end }}`

type SetBranchReferencesArgs struct {
	RancherRepoPath   string
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

type ContentLine struct {
	Line    int
	File    string
	Content string
}

type Content struct {
	RancherImages  []ContentLine
	FilesWithRC    []ContentLine
	MinFilesWithRC []ContentLine
	ChartsWithDev  []ContentLine
	KDMWithDev     []ContentLine
}

type ListBucketResult struct {
	Contents []struct {
		Key string `xml:"Key"`
	} `xml:"Contents"`
}

type ArtifactsIndexContent struct {
	GA         ArtifactsIndexContentGroup `json:"ga"`
	PreRelease ArtifactsIndexContentGroup `json:"preRelease"`
}

type ArtifactsIndexContentGroup struct {
	Versions map[string][]string `json:"versions"`
	BaseURL  string              `json:"baseUrl"`
}

func GeneratePrimeArtifactsIndex(path string) error {
	client := ecmHTTP.NewClient(time.Second * 15)
	resp, err := client.Get(rancherArtifactsListURL)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code on " + rancherArtifactsListURL + " expected 200, got " + strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()
	contentDecoder := xml.NewDecoder(resp.Body)
	var listBucket ListBucketResult
	if err := contentDecoder.Decode(&listBucket); err != nil {
		return err
	}
	content := generateArtifactsIndexContent(listBucket)
	gaIndex, err := generatePrimeArtifactsHTML(content.GA)
	if err != nil {
		return err
	}
	preReleaseIndex, err := generatePrimeArtifactsHTML(content.PreRelease)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "index.html"), gaIndex, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "index-prerelease.html"), preReleaseIndex, 0644)
}

func generateArtifactsIndexContent(listBucket ListBucketResult) ArtifactsIndexContent {
	indexContent := ArtifactsIndexContent{
		GA: ArtifactsIndexContentGroup{
			Versions: map[string][]string{},
			BaseURL:  rancherArtifactsBaseURL,
		},
		PreRelease: ArtifactsIndexContentGroup{
			Versions: map[string][]string{},
			BaseURL:  rancherArtifactsBaseURL,
		},
	}
	for _, content := range listBucket.Contents {
		if !strings.Contains(content.Key, "rancher/") {
			continue
		}
		keyFile := strings.Split(strings.TrimPrefix(content.Key, "rancher/"), "/")
		if len(keyFile) < 2 || keyFile[1] == "" {
			continue
		}
		key := keyFile[0]
		file := keyFile[1]

		// only non ga releases contains '-' e.g: -rc, -debug
		if strings.Contains(key, "-") {
			indexContent.PreRelease.Versions[key] = append(indexContent.PreRelease.Versions[key], file)
		} else {
			indexContent.GA.Versions[key] = append(indexContent.GA.Versions[key], file)
		}
	}
	return indexContent
}

func generatePrimeArtifactsHTML(content ArtifactsIndexContentGroup) ([]byte, error) {
	tmpl, err := htmlTemplate.New("release-artifacts-index").Parse(artifactsIndexTempalte)
	if err != nil {
		return nil, err
	}
	buff := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buff, "release-artifacts-index", content); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
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

func CheckHelmChartVersion(tag string) error {
	versions, err := rancherHelmChartVersions(rancherHelmRepositoryURL)
	if err != nil {
		return err
	}
	var foundVersion bool
	for _, version := range versions {
		logrus.Info("checking version " + version)
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
	logrus.Info("downloading: " + repoURL)
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

	script := navigateCheckoutRancherScript + setKDMBranchReferencesScript + pushChangesScript
	logrus.Info("running update files and apply updates script...")
	output, err := exec.RunTemplatedScript(forkPath, setKDMBranchReferencesScriptFileName, script, template.FuncMap{}, data)
	if err != nil {
		return err
	}
	logrus.Info(output)

	if createPR {
		prName := "Update KDM to " + newKDMBranch
		logrus.Info("creating PR")
		if dryRun {
			logrus.Info("dry run, PR will not be created")
			logrus.Info("PR:\n  Name: " + prName + "\n  From: " + githubUser + ":" + branchName + "\n  To rancher:" + rancherBaseBranch)
			return nil
		}
		ghClient := repository.NewGithub(ctx, githubToken)

		if err := createPRFromRancher(ctx, rancherBaseBranch, prName, branchName, githubUser, ghClient); err != nil {
			return err
		}
	}
	return nil
}

func SetChartBranchReferences(ctx context.Context, forkPath, rancherBaseBranch, newBranch, githubUser, githubToken string, createPR, dryRun bool) error {
	branchName := "charts-set-" + newBranch
	data := SetBranchReferencesArgs{
		RancherRepoPath:   forkPath,
		NewBranch:         newBranch,
		RancherBaseBranch: rancherBaseBranch,
		DryRun:            dryRun,
		BranchName:        branchName,
	}
	script := navigateCheckoutRancherScript + setChartBranchReferencesScript + pushChangesScript
	logrus.Info("running update files script")
	output, err := exec.RunTemplatedScript(forkPath, setChartReferencesScriptFileName, script, template.FuncMap{}, data)
	if err != nil {
		return err
	}
	logrus.Info(output)

	if createPR {
		prName := "Update charts branch references to " + newBranch
		logrus.Info("creating PR")
		if dryRun {
			logrus.Info("dry run, PR will not be created")
			logrus.Info("PR:\n  Name: " + prName + "\n  From: " + githubUser + ":" + branchName + "\n  To rancher:" + rancherBaseBranch)
			return nil
		}
		ghClient := repository.NewGithub(ctx, githubToken)

		if err := createPRFromRancher(ctx, rancherBaseBranch, prName, branchName, githubUser, ghClient); err != nil {
			return err
		}
	}

	return nil
}

func CreateRelease(ctx context.Context, ghClient *github.Client, r *ecmConfig.RancherRelease, opts *repository.CreateReleaseOpts, preRelease bool, releaseType string) error {
	fmt.Println("validating tag semver format")
	if !semver.IsValid(opts.Tag) {
		return errors.New("the tag isn't a valid semver: " + opts.Tag)
	}
	fmt.Println("getting remote branch information from " + r.RancherRepoOwner + "/" + rancherRepo)
	branch, _, err := ghClient.Repositories.GetBranch(ctx, r.RancherRepoOwner, rancherRepo, r.ReleaseBranch, true)
	if err != nil {
		return err
	}
	if branch.Commit.SHA == nil {
		return errors.New("branch commit sha is nil")
	}
	fmt.Println("the latest commit on branch " + r.ReleaseBranch + " is: " + *branch.Commit.SHA)
	if !r.SkipStatusCheck {
		fmt.Println("checking if CI is passing")
		if err := commitStateSuccess(ctx, ghClient, r.RancherRepoOwner, rancherRepo, *branch.Commit.SHA); err != nil {
			return err
		}
	}

	releaseName := opts.Tag
	if preRelease {
		if releaseType == "debug" {
			if r.IssueNumber == "" {
				return errors.New("debug releases require an issue number")
			}
			releaseType = "debug-" + r.IssueNumber + "-"
		}
		latestVersionNumber := 1
		latestVersion, err := release.LatestPreRelease(ctx, ghClient, opts.Owner, opts.Repo, opts.Tag, releaseType)
		if err != nil {
			return err
		}
		if latestVersion != nil {
			trimmedVersionNumber := strings.TrimPrefix(*latestVersion, opts.Tag+"-"+releaseType)
			currentVersionNumber, err := strconv.Atoi(trimmedVersionNumber)
			if err != nil {
				return errors.New("failed to parse trimmed latest version number: " + err.Error())
			}
			latestVersionNumber = currentVersionNumber + 1
		}
		opts.Tag = opts.Tag + "-" + releaseType + strconv.Itoa(latestVersionNumber)
		releaseName = "Pre-release " + opts.Tag
	}

	opts.Name = releaseName
	opts.Prerelease = true
	opts.Draft = !preRelease
	opts.ReleaseNotes = ""

	fmt.Printf("creating release with options: %+v\n", opts)
	if r.DryRun {
		fmt.Println("dry run, skipping tag creation")
		return nil
	}
	createdRelease, err := repository.CreateRelease(ctx, ghClient, opts)
	if err != nil {
		return err
	}
	fmt.Println("release created: " + *createdRelease.URL)
	return nil
}

func commitStateSuccess(ctx context.Context, ghClient *github.Client, owner, repo, commit string) error {
	status, _, err := ghClient.Repositories.GetCombinedStatus(ctx, owner, repo, commit, &github.ListOptions{})
	if err != nil {
		return err
	}
	if *status.State != "success" {
		return errors.New("expected commit " + commit + " to have state 'success', instead, got " + *status.State)
	}
	return nil
}

func rancherReleaseName(generalAvailability bool, tag string) string {
	releaseName := ""
	if !generalAvailability {
		releaseName += "Pre-release "
	}
	releaseName += tag
	return releaseName
}

func createPRFromRancher(ctx context.Context, rancherBaseBranch, title, branchName, forkOwner string, ghClient *github.Client) error {
	pull := &github.NewPullRequest{
		Title:               github.String(title),
		Base:                github.String(rancherBaseBranch),
		Head:                github.String(forkOwner + ":" + branchName),
		MaintainerCanModify: github.Bool(true),
	}
	_, _, err := ghClient.PullRequests.Create(ctx, "rancher", "rancher", pull)

	return err
}

func CheckRancherRCDeps(ctx context.Context, local, forCi bool, org, repo, commitHash, files string) error {
	var (
		content  Content
		badFiles bool
	)

	devDependencyPattern := regexp.MustCompile(`dev-v[0-9]+\.[0-9]+`)
	rcTagPattern := regexp.MustCompile(`-rc[0-9]+`)

	ghClient := repository.NewGithub(ctx, "")

	for _, filePath := range strings.Split(files, ",") {
		var scanner *bufio.Scanner
		if local {
			content, err := contentLocal("./" + filePath)
			if err != nil {
				if os.IsNotExist(err) {
					logrus.Debugf("file '%s' not found, skipping...", filePath)
					continue
				}
				return err
			}
			defer content.Close()
			scanner = bufio.NewScanner(content)
		} else {
			if strings.Contains(filePath, "bin") {
				continue
			}
			content, err := contentRemote(ctx, ghClient, org, repo, commitHash, filePath)
			if err != nil {
				return err
			}
			scanner = bufio.NewScanner(strings.NewReader(content))
		}

		lineNum := 1

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(filePath, "bin/rancher") {
				badFiles = true
				lineContent := ContentLine{File: filePath, Line: lineNum, Content: formatContentLine(line)}
				content.RancherImages = append(content.RancherImages, lineContent)
				continue
			}
			if devDependencyPattern.MatchString(line) {
				lineContent := ContentLine{File: filePath, Line: lineNum, Content: formatContentLine(line)}
				lineContentLower := strings.ToLower(lineContent.Content)
				if strings.Contains(lineContentLower, "chart") {
					badFiles = true
					content.ChartsWithDev = append(content.ChartsWithDev, lineContent)
				}
				if strings.Contains(lineContentLower, "kdm") {
					badFiles = true
					content.KDMWithDev = append(content.KDMWithDev, lineContent)
				}
			}
			if strings.Contains(filePath, "/package/Dockerfile") {
				if regexp.MustCompile(`CATTLE_(\S+)_MIN_VERSION`).MatchString(line) && strings.Contains(line, "-rc") {
					badFiles = true
					lineContent := ContentLine{Line: lineNum, File: filePath, Content: formatContentLine(line)}
					content.MinFilesWithRC = append(content.MinFilesWithRC, lineContent)
				}
			}
			if rcTagPattern.MatchString(line) {
				badFiles = true
				lineContent := ContentLine{File: filePath, Line: lineNum, Content: formatContentLine(line)}
				content.FilesWithRC = append(content.FilesWithRC, lineContent)
			}
			lineNum++
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	tmpl := template.New("rancher-release-rc-dev-deps")
	tmpl = template.Must(tmpl.Parse(templateCheckRCDevDeps))
	buff := bytes.NewBuffer(nil)
	err := tmpl.ExecuteTemplate(buff, "componentsFile", content)
	if err != nil {
		return err
	}

	fmt.Println(buff.String())

	if forCi && badFiles {
		return errors.New("check failed, some files don't match the expected dependencies for a final release candidate")
	}

	return nil
}

func contentLocal(filePath string) (*os.File, error) {
	repoContent, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return repoContent, nil
}

func contentRemote(ctx context.Context, ghClient *github.Client, org, repo, commitHash, filePath string) (string, error) {
	content, _, _, err := ghClient.Repositories.GetContents(ctx, org, repo, filePath, &github.RepositoryContentGetOptions{Ref: commitHash})
	if err != nil {
		return "", err
	}
	decodedContent, err := content.GetContent()
	if err != nil {
		return "", err
	}
	return decodedContent, nil
}

func formatContentLine(line string) string {
	re := regexp.MustCompile(`\s+`)
	line = re.ReplaceAllString(line, " ")
	return strings.TrimSpace(line)
}

const artifactsIndexTempalte = `{{ define "release-artifacts-index" }}
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Rancher Prime Artifacts</title>
    <style>
    body { font-family: Verdana, Geneneva; }
    header { display: flex; flex-direction: row; justify-items: center; }
    #rancher-logo { width: 200px; }
    .project { margin-left: 20px; }
    .release { margin-left: 40px; margin-bottom: 20px; }
    .release h3 { margin-bottom: 0px; }
    .files { margin-left: 60px; display: flex; flex-direction: column; }
    .release-title { display: flex; flex-direction: row; }
    .release-title-tag { margin-right: 20px; }
    .release-title-expand { background-color: #2453ff; color: white; border-radius: 5px; border: none; }
    .release-title-expand:hover, .expand-active{ background-color: white; color: #2453ff; border: 1px solid #2453ff; }
    .hidden { display: none; overflow: hidden; }
    </style>
  </head>
  <body>
    <header>
      <img src="https://www.rancher.com/assets/img/logos/rancher-suse-logo-horizontal-color.svg" alt="rancher logo" id="rancher-logo" />
      <h1>Prime Artifacts</h1>
    </header>
    <main>
      <div class="project-rancher project">
        <h2>rancher</h2>
        {{ range $version, $files := .Versions }}
        <div class="release-{{ $version }} release">
          <div class="release-title">
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files hidden" id="release-{{ $version }}-files">
            <ul>
              {{ range $files }}
              <li><a href="{{ $.BaseURL }}/rancher/{{ $version }}/{{ . }}">{{ $.BaseURL }}/rancher/{{ $version }}/{{ . }}</a></li>
              {{ end }}
            </ul>
          </div>
        </div>
				{{ end }}
      </div>
    </main>
  <script>
    function expand(tag) {
      const filesId = "release-" + tag + "-files"
      const expandButtonId = "release-" + tag + "-expand"
      document.getElementById(filesId).classList.toggle("hidden")
      document.getElementById(expandButtonId).classList.toggle("expand-active")
    }
  </script>
  </body>
</html>
{{end}}`
