package rancher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"log"
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
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

const (
	rancherOrg               = "rancher"
	rancherRepo              = rancherOrg
	rancherImagesBaseURL     = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName    = "/rancher-images.txt"
	rancherHelmRepositoryURL = "https://releases.rancher.com/server-charts/latest/index.yaml"
	rancherArtifactsListURL  = "https://prime-artifacts.s3.amazonaws.com"
	rancherArtifactsBaseURL  = "https://prime.ribs.rancher.io"
	rancherRegistryBaseURL   = "https://registry.rancher.com"
	sccSUSEBaseURL           = "https://scc.suse.com"
	sccSUSEService           = "SUSE+Linux+Docker+Registry"
)

type RancherRCDepsLine struct {
	Line    int    `json:"line"`
	File    string `json:"file"`
	Content string `json:"content"`
}

type RancherRCDeps struct {
	RancherImages  []RancherRCDepsLine `json:"rancherImages"`
	FilesWithRC    []RancherRCDepsLine `json:"filesWithRc"`
	MinFilesWithRC []RancherRCDepsLine `json:"minFilesWithRc"`
	ChartsWithDev  []RancherRCDepsLine `json:"chartsWithDev"`
	KDMWithDev     []RancherRCDepsLine `json:"kdmWithDev"`
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
	Versions      []string            `json:"versions"`
	VersionsFiles map[string][]string `json:"versionsFiles"`
	BaseURL       string              `json:"baseUrl"`
}

type RegistryAuth struct {
	Token string `json:"token"`
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
			Versions:      []string{},
			VersionsFiles: map[string][]string{},
			BaseURL:       rancherArtifactsBaseURL,
		},
		PreRelease: ArtifactsIndexContentGroup{
			Versions:      []string{},
			VersionsFiles: map[string][]string{},
			BaseURL:       rancherArtifactsBaseURL,
		},
	}
	var versions []string
	versionsFiles := make(map[string][]string)

	for _, content := range listBucket.Contents {
		if !strings.Contains(content.Key, "rancher/") {
			continue
		}
		keyFile := strings.Split(strings.TrimPrefix(content.Key, "rancher/"), "/")
		if len(keyFile) < 2 || keyFile[1] == "" {
			continue
		}
		version := keyFile[0]
		file := keyFile[1]

		if _, ok := versionsFiles[version]; !ok {
			versions = append(versions, version)
		}
		versionsFiles[version] = append(versionsFiles[version], file)
	}

	semver.Sort(versions)

	// starting from the last index will result in a newest to oldest sorting
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		// only non ga releases contains '-' e.g: -rc, -debug
		if strings.Contains(version, "-") {
			indexContent.PreRelease.Versions = append(indexContent.PreRelease.Versions, version)
			indexContent.PreRelease.VersionsFiles[version] = versionsFiles[version]
		} else {
			indexContent.GA.Versions = append(indexContent.GA.Versions, version)
			indexContent.GA.VersionsFiles[version] = versionsFiles[version]
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

func CheckRancherRCDeps(ctx context.Context, org, gitRef string) (*RancherRCDeps, error) {
	var content RancherRCDeps
	files := []string{"Dockerfile.dapper", "go.mod", "/package/Dockerfile", "/pkg/apis/go.mod", "/pkg/settings/setting.go", "/scripts/package-env"}
	devDependencyPattern := regexp.MustCompile(`dev-v[0-9]+\.[0-9]+`)
	rcTagPattern := regexp.MustCompile(`-rc[0-9]+`)
	ghClient := repository.NewGithub(ctx, "")

	for _, filePath := range files {
		var scanner *bufio.Scanner
		fileContent, err := remoteGitContent(ctx, ghClient, org, rancherRepo, gitRef, filePath)
		if err != nil {
			return nil, err
		}
		scanner = bufio.NewScanner(strings.NewReader(fileContent))
		lineNum := 1

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "indirect") {
				continue
			}
			if devDependencyPattern.MatchString(line) {
				lineContent := RancherRCDepsLine{File: filePath, Line: lineNum, Content: formatContentLine(line)}
				lineContentLower := strings.ToLower(lineContent.Content)
				if strings.Contains(lineContentLower, "chart") {
					content.ChartsWithDev = append(content.ChartsWithDev, lineContent)
				} else if strings.Contains(lineContentLower, "kdm") {
					content.KDMWithDev = append(content.KDMWithDev, lineContent)
				}
			}
			if strings.Contains(filePath, "/package/Dockerfile") {
				if regexp.MustCompile(`CATTLE_(\S+)_MIN_VERSION`).MatchString(line) && strings.Contains(line, "-rc") {
					lineContent := RancherRCDepsLine{Line: lineNum, File: filePath, Content: formatContentLine(line)}
					content.MinFilesWithRC = append(content.MinFilesWithRC, lineContent)
				}
			}
			if rcTagPattern.MatchString(line) {
				lineContent := RancherRCDepsLine{File: filePath, Line: lineNum, Content: formatContentLine(line)}
				content.FilesWithRC = append(content.FilesWithRC, lineContent)
			}
			lineNum++
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	return &content, nil
}

func (r *RancherRCDeps) ToString() (string, error) {
	tmpl := template.New("rancher-release-rc-dev-deps")
	tmpl = template.Must(tmpl.Parse(checkRancherRCDepsTemplate))
	buff := bytes.NewBuffer(nil)
	err := tmpl.ExecuteTemplate(buff, "componentsFile", r)
	return buff.String(), err
}

func remoteGitContent(ctx context.Context, ghClient *github.Client, org, repo, gitRef, filePath string) (string, error) {
	content, _, _, err := ghClient.Repositories.GetContents(ctx, org, repo, filePath, &github.RepositoryContentGetOptions{Ref: gitRef})
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

func GenerateMissingImagesList(version string) ([]string, error) {
	if !semver.IsValid(version) {
		return nil, errors.New("version is not a valid semver: " + version)
	}
	const rancherWindowsImagesFile = "rancher-windows-images.txt"
	const rancherImagesFile = "rancher-images.txt"
	rancherWindowsImages, err := getRancherPrimeArtifact(version, rancherWindowsImagesFile)
	if err != nil {
		return nil, errors.New("failed to get rancher windows images: " + err.Error())
	}
	rancherImages, err := getRancherPrimeArtifact(version, rancherImagesFile)
	if err != nil {
		return nil, errors.New("failed to get rancher images: " + err.Error())
	}
	images := append(rancherWindowsImages, rancherImages...)

	// create an error group with a limit to prevent accidentaly doing a DOS attack against our registry
	ctx, cancel := context.WithCancel(context.Background())
	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.SetLimit(2)
	missingImagesChan := make(chan string, len(images))

	for _, imageAndVersion := range images {
		splitImage := strings.Split(imageAndVersion, ":")
		image := splitImage[0]
		imageVersion := splitImage[1]

		func(ctx context.Context, missingImagesChan chan string, image, imageVersion string) {
			errGroup.Go(func() error {
				// if any other check failed, stop running to prevent wasting resources
				// this doesn't include 404's since it is expected that this happens some times
				// it does include any other errors
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					exists, err := checkIfImageExists(image, imageVersion)
					if err != nil {
						cancel()
						return err
					}
					fullImage := image + ":" + imageVersion
					if !exists {
						missingImagesChan <- fullImage
						log.Println(fullImage + " does not exists")
					} else {
						log.Println(fullImage + " exists")
					}
					return nil
				}
			})
		}(ctx, missingImagesChan, image, imageVersion)

	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	close(missingImagesChan)
	missingImages := readStringChan(missingImagesChan)
	return missingImages, nil
}

func checkIfImageExists(img, imgVersion string) (bool, error) {
	log.Println("checking image: " + img + ":" + imgVersion)
	auth, err := getRegistryAuth(sccSUSEService, img)
	if err != nil {
		return false, err
	}
	httpClient := ecmHTTP.NewClient(time.Second * 5)
	req, err := http.NewRequest("GET", rancherRegistryBaseURL+"/v2/"+img+"/manifests/"+imgVersion, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+prettyjws")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")
	req.Header.Add("Docker-Distribution-Api-Version", "registry/2.0")
	req.Header.Add("Authorization", "Bearer "+auth)
	res, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if res.StatusCode != http.StatusOK {
		return false, errors.New("expected status code to be 200, got: " + strconv.Itoa(res.StatusCode))
	}
	return true, nil
}

func getRegistryAuth(service, image string) (string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 5)
	scope := "repository:" + image + ":pull"
	res, err := httpClient.Get(sccSUSEBaseURL + "/api/registry/authorize?scope=" + scope + "&service=" + service)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", errors.New("expected status code to be 200, got: " + strconv.Itoa(res.StatusCode))
	}
	decoder := json.NewDecoder(res.Body)
	var auth RegistryAuth
	if err := decoder.Decode(&auth); err != nil {
		return "", err
	}
	return auth.Token, nil
}

func getRancherPrimeArtifact(version, artifactName string) ([]string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	res, err := httpClient.Get(rancherArtifactsBaseURL + "/rancher/" + version + "/" + artifactName)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var file []string
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		file = append(file, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return file, nil
}

func readStringChan(ch <-chan string) []string {
	var data []string
	for s := range ch {
		data = append(data, s)
	}
	return data
}

const artifactsIndexTempalte = `{{ define "release-artifacts-index" }}
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Rancher Prime Artifacts</title>
    <link rel="icon" type="image/png" href="https://prime.ribs.rancher.io/assets/img/favicon.png">
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
      <img src="https://prime.ribs.rancher.io/assets/img/rancher-suse-logo-horizontal-color.svg" alt="rancher logo" id="rancher-logo" />
      <h1>Prime Artifacts</h1>
    </header>
    <main>
      <div class="project-rancher project">
        <h2>rancher</h2>
        {{ range $i, $version := .Versions }}
        <div class="release-{{ $version }} release">
          <div class="release-title">
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files hidden" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.VersionsFiles $version }}
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
const checkRancherRCDepsTemplate = `{{- define "componentsFile" -}}
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
