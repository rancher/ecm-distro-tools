package rancher

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
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

	setKDMBranchReferencesScriptFileName = "set_kdm_branch_references.sh"
	setChartReferencesScriptFileName     = `set_chart_references.sh`
	runComponentsFileScriptFileName      = `run_components_file.sh`
	scriptsWorkingDir                    = `/tmp`
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
	cloneRancherRunComponentsFileScript = `#!/bin/sh
set -e
REPO_NAME={{ .RepoName }}
REPO_OWNER={{ .RepoOwner }}
REPO_PATH={{ .RepoPath }}
BRANCH={{ .Branch }}
echo "repo name: ${REPO_NAME}"
echo "org name: ${REPO_OWNER}"
echo "cloning ${REPO_OWNER}/${REPO_NAME} into ${REPO_PATH}"
git clone "git@github.com:${REPO_OWNER}/${REPO_NAME}.git" "${REPO_PATH}"
cd "${REPO_PATH}"
git switch "${BRANCH}"
./scripts/create-components-file.sh`
	navigateRunComponentsFileScript = `#!/bin/sh
set -e
REPO_PATH={{ .RepoPath }}
BRANCH={{ .Branch }}
cd "${REPO_PATH}"
echo "stashing local changes"
git stash
echo "fetching changes"
git fetch 
echo "switch to branch ${BRANCH}"
git switch "${BRANCH}"
echo "pulling latests changes"
git pull origin "${BRANCH}"
./scripts/create-components-file.sh`
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

type RunComponentsFileArgs struct {
	RepoName  string
	RepoOwner string
	RepoPath  string
	Branch    string
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
	output, err := exec.RunTemplatedScript(forkPath, setKDMBranchReferencesScriptFileName, script, data)
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
	output, err := exec.RunTemplatedScript(forkPath, setChartReferencesScriptFileName, script, data)
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

func TagRancherRelease(ctx context.Context, ghClient *github.Client, tag, remoteBranch, repoOwner, repoPath string, generalAvailability, ignoreDraft, dryRun bool) error {
	logrus.Info("validating tag semver format")
	if !semver.IsValid(tag) {
		return errors.New("the tag `" + tag + "` isn't a valid semantic versioning string")
	}
	logrus.Info("getting remote branch information from " + repoOwner + "/" + rancherRepo)
	branch, _, err := ghClient.Repositories.GetBranch(ctx, repoOwner, rancherRepo, remoteBranch, true)
	if err != nil {
		return err
	}
	logrus.Info("the latest commit on branch " + remoteBranch + " is: " + *branch.Commit.SHA)
	logrus.Info("running components file")
	releaseBody, err := rancherComponents(remoteBranch, repoOwner, repoPath)
	if err != nil {
		return err
	}
	createAsDraft := !ignoreDraft
	createAsPrerelease := !generalAvailability
	logrus.Info("creating release ")
	ghRelease := github.RepositoryRelease{
		TagName:              github.String(tag),
		Name:                 github.String(rancherReleaseName(generalAvailability, tag)),
		Body:                 github.String(releaseBody),
		Draft:                &createAsDraft,
		Prerelease:           &createAsPrerelease,
		GenerateReleaseNotes: github.Bool(false),
	}
	logrus.Infof("github release: %+v", ghRelease)
	if dryRun {
		logrus.Info("dry run, skipping release creation")
		return nil
	}
	_, _, err = ghClient.Repositories.CreateRelease(ctx, repoOwner, rancherRepo, &ghRelease)
	return err
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
func rancherComponents(branch, repoOwner, repoPath string) (string, error) {
	script := navigateRunComponentsFileScript
	if repoPath == "" {
		repoPath = scriptsWorkingDir + "/" + rancherRepo
		script = cloneRancherRunComponentsFileScript
	}
	output, err := exec.RunTemplatedScript(scriptsWorkingDir, runComponentsFileScriptFileName, script,
		RunComponentsFileArgs{
			RepoName:  rancherRepo,
			RepoOwner: repoOwner,
			RepoPath:  repoPath,
			Branch:    branch,
		},
	)
	if err != nil {
		return "", err
	}
	logrus.Info(output)
	components, err := os.ReadFile(repoPath + "/bin/rancher-components.txt")
	if err != nil {
		return "", err
	}
	return string(components), nil
}
