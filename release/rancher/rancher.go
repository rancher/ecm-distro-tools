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
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

const (
	rancherOrg                    = "rancher"
	rancherRepo                   = rancherOrg
	rancherImagesBaseURL          = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName         = "/rancher-images.txt"
	rancherHelmRepositoryURL      = "https://releases.rancher.com/server-charts/latest/index.yaml"
	rancherArtifactsListURL       = "https://prime-artifacts.s3.amazonaws.com"
	rancherArtifactsBaseURL       = "https://prime.ribs.rancher.io"
	rancherRegistryBaseURL        = "https://registry.rancher.com"
	stagingRancherRegistryBaseURL = "https://stgregistry.suse.com"
	sccSUSEURL                    = "https://scc.suse.com/api/registry/authorize"
	stagingSccSUSEURL             = "https://stgscc.suse.com/api/registry/authorize"
	dockerRegistryURL             = "https://registry-1.docker.io"
	dockerAuthURL                 = "https://auth.docker.io/token"
	sccSUSEService                = "SUSE+Linux+Docker+Registry"
	dockerService                 = "registry.docker.io"
)

var registriesInfo = map[string]registryInfo{
	"registry.rancher.com": {
		BaseURL: rancherRegistryBaseURL,
		AuthURL: sccSUSEURL,
		Service: sccSUSEService,
	},
	"stgregistry.suse.com": {
		BaseURL: stagingRancherRegistryBaseURL,
		AuthURL: stagingSccSUSEURL,
		Service: sccSUSEService,
	},
	"docker.io": {
		BaseURL: dockerRegistryURL,
		AuthURL: dockerAuthURL,
		Service: dockerService,
	},
}

type registryInfo struct {
	BaseURL string
	AuthURL string
	Service string
}

type imageDigest map[string]string

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

type registryAuthToken struct {
	Token string `json:"token"`
}

func GeneratePrimeArtifactsIndex(path string, ignoreVersions []string) error {
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

	ignore := make(map[string]bool, len(ignoreVersions))
	for _, v := range ignoreVersions {
		ignore[v] = true
	}

	content := generateArtifactsIndexContent(listBucket, ignore)
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

func generateArtifactsIndexContent(listBucket ListBucketResult, ignoreVersions map[string]bool) ArtifactsIndexContent {
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

		if _, ok := ignoreVersions[version]; ok {
			continue
		}

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

func GenerateMissingImagesList(version string, concurrencyLimit int, images []string) ([]string, error) {
	if !semver.IsValid(version) {
		return nil, errors.New("version is not a valid semver: " + version)
	}
	if len(images) == 0 {
		const rancherWindowsImagesFile = "rancher-windows-images.txt"
		const rancherImagesFile = "rancher-images.txt"
		rancherWindowsImages, err := rancherPrimeArtifact(version, rancherWindowsImagesFile)
		if err != nil {
			return nil, errors.New("failed to get rancher windows images: " + err.Error())
		}
		rancherImages, err := rancherPrimeArtifact(version, rancherImagesFile)
		if err != nil {
			return nil, errors.New("failed to get rancher images: " + err.Error())
		}
		images = append(rancherWindowsImages, rancherImages...)
	}

	// create an error group with a limit to prevent accidentaly doing a DOS attack against our registry
	ctx, cancel := context.WithCancel(context.Background())
	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.SetLimit(concurrencyLimit)
	missingImagesChan := make(chan string, len(images))
	// auth tokens can be reused, but maps need a lock for reading and writing in go routines
	repositoryAuths := make(map[string]string)
	mu := sync.RWMutex{}

	for _, imageAndVersion := range images {
		if !strings.Contains(imageAndVersion, ":") {
			cancel()
			return nil, errors.New("malformed image name: , missing ':'")
		}
		splitImage := strings.Split(imageAndVersion, ":")
		image := splitImage[0]
		imageVersion := splitImage[1]

		func(ctx context.Context, missingImagesChan chan string, image, imageVersion string, repositoryAuths map[string]string, mu *sync.RWMutex) {
			errGroup.Go(func() error {
				// if any other check failed, stop running to prevent wasting resources
				// this doesn't include 404's since it is expected it does include any other errors
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					mu.Lock()
					var ok bool
					var auth string
					var err error
					auth, ok = repositoryAuths[image]
					if !ok {
						auth, err = registryAuth(sccSUSEURL, sccSUSEService, image)
						if err != nil {
							cancel()
							return err
						}
						repositoryAuths[image] = auth
					}
					mu.Unlock()
					exists, err := checkIfImageExists(rancherRegistryBaseURL, image, imageVersion, auth)
					if err != nil {
						cancel()
						return err
					}
					fullImage := image + ":" + imageVersion
					if !exists {
						missingImagesChan <- fullImage
						log.Println(fullImage + " is missing")
					} else {
						log.Println(fullImage + " exists")
					}
					return nil
				}
			})
		}(ctx, missingImagesChan, image, imageVersion, repositoryAuths, &mu)

	}
	if err := errGroup.Wait(); err != nil {
		cancel()
		return nil, err
	}
	cancel()
	close(missingImagesChan)
	missingImages := readStringChan(missingImagesChan)
	return missingImages, nil
}

func GenerateDockerImageDigests(outputFile, imagesFileURL, registry string) error {
	imagesDigests, err := dockerImagesDigests(imagesFileURL, registry)
	if err != nil {
		return err
	}
	return createAssetFile(outputFile, imagesDigests)
}

func dockerImagesDigests(imagesFileURL, registry string) (imageDigest, error) {
	imagesList, err := artifactImageList(imagesFileURL, registry)
	if err != nil {
		return nil, err
	}

	rgInfo, ok := registriesInfo[registry]
	if !ok {
		return nil, errors.New("registry must be one of the following: 'docker.io', 'registry.rancher.com' or 'stgregistry.suse.com'")
	}

	imagesDigests := make(imageDigest)
	var repositoryAuths = make(map[string]string)

	for _, imageAndVersion := range imagesList {
		if imageAndVersion == "" || imageAndVersion == " " {
			continue
		}
		slog.Info("image: " + imageAndVersion)
		if !strings.Contains(imageAndVersion, ":") {
			return nil, errors.New("malformed image name: , missing ':'")
		}
		splitImage := strings.Split(imageAndVersion, ":")
		image := splitImage[0]
		imageVersion := splitImage[1]

		if _, ok := repositoryAuths[image]; !ok {
			auth, err := registryAuth(rgInfo.AuthURL, rgInfo.Service, image)
			if err != nil {
				return nil, err
			}
			repositoryAuths[image] = auth
		}
		digest, statusCode, err := dockerImageDigest(rgInfo.BaseURL, image, imageVersion, repositoryAuths[image])
		slog.Info("status code: " + strconv.Itoa(statusCode))
		if err != nil {
			return nil, err
		}
		imagesDigests[imageAndVersion] = digest
	}
	return imagesDigests, nil
}

func createAssetFile(outputFile string, contents fmt.Stringer) error {
	fo, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer fo.Close()
	_, err = fo.Write([]byte(contents.String()))
	return err
}

func artifactImageList(imagesFileURL, registry string) ([]string, error) {
	client := http.Client{Timeout: time.Second * 15}
	res, err := client.Get(imagesFileURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	list, err := getLinesFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return list, fmt.Errorf("no outputFile %s found or contents were empty, can not proceed", imagesFileURL)
	}

	for k, im := range list {
		if im == "" || im == " " {
			continue
		}
		image := cleanImage(im, registry)
		list[k] = image
	}

	return list, nil
}

func cleanImage(image, registry string) string {
	switch registry {
	case "docker.io":
		if len(strings.Split(image, "/")) == 1 {
			image = path.Join("library", image)
		}
	}

	return image
}

func (d imageDigest) String() string {
	var o strings.Builder
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&o, "%s %s\n", k, d[k])
	}
	return o.String()
}

func getLinesFromReader(body io.Reader) ([]string, error) {
	lines, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return []string{}, errors.New("file was empty")
	}

	return strings.Split(string(lines), "\n"), nil
}

func dockerImageDigest(registryBaseURL, img, imgVersion, auth string) (string, int, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 5)
	req, err := http.NewRequest("GET", registryBaseURL+"/v2/"+img+"/manifests/"+imgVersion, nil)
	if err != nil {
		return "", 0, err
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
		return "", 0, err
	}
	if res.StatusCode == http.StatusNotFound {
		return "", res.StatusCode, nil
	}
	dockerDigest := res.Header.Get("Docker-Content-Digest")
	if dockerDigest == "" {
		return "", res.StatusCode, errors.New("empty digest header 'Docker-Content-Digest'")
	}
	return dockerDigest, res.StatusCode, nil
}

func checkIfImageExists(registryBaseURL, img, imgVersion, auth string) (bool, error) {
	log.Println("checking image: " + img + ":" + imgVersion)
	_, statusCode, err := dockerImageDigest(registryBaseURL, img, imgVersion, auth)
	if err != nil {
		return false, err
	}
	if statusCode == http.StatusNotFound {
		return false, nil
	}
	if statusCode != http.StatusOK {
		return false, errors.New("expected status code to be 200, got: " + strconv.Itoa(statusCode))
	}
	return true, nil
}

func registryAuth(authURL, service, image string) (string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 5)
	scope := "repository:" + image + ":pull"
	url := authURL + "?scope=" + scope + "&service=" + service
	res, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", errors.New("expected status code to be 200, got: " + strconv.Itoa(res.StatusCode))
	}
	decoder := json.NewDecoder(res.Body)
	var auth registryAuthToken
	if err := decoder.Decode(&auth); err != nil {
		return "", err
	}
	return auth.Token, nil
}

func rancherPrimeArtifact(version, artifactName string) ([]string, error) {
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

func UploadRancherArtifacts(ctx context.Context, ghClient *github.Client, s3Uploader *manager.Uploader, rancherRelease *config.RancherRelease, releaseTag string) error {
	fmt.Println("validating release tag: " + releaseTag)
	if !semver.IsValid(releaseTag) {
		return errors.New("the tag isn't a valid semver: " + releaseTag)
	}
	fmt.Println("getting release by tag: " + releaseTag)
	release, _, err := ghClient.Repositories.GetReleaseByTag(ctx, rancherRelease.RancherRepoOwner, rancherRepo, releaseTag)
	if err != nil {
		return errors.New("failed to get release by tag: " + err.Error())
	}
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	releaseAssets := make(map[string][]byte)
	fmt.Println("downloading release assets")
	for _, asset := range release.Assets {
		fmt.Println("downloading asset: " + *asset.Name)
		rc, _, err := ghClient.Repositories.DownloadReleaseAsset(ctx, rancherRelease.RancherRepoOwner, rancherRepo, *asset.ID, &httpClient)
		if err != nil {
			return errors.New("failed to download release asset: " + err.Error())
		}
		fmt.Println("reading asset content")
		releaseAssets[*asset.Name], err = io.ReadAll(rc)
		if err != nil {
			return errors.New("failed to read release asset content: " + err.Error())
		}
		if err := rc.Close(); err != nil {
			return errors.New("failed to close reader body: " + err.Error())
		}
		// Only artifacts with "digests" in the name contain registry information
		if strings.Contains(*asset.Name, "digests") {
			fmt.Println("digests artifact, replacing registry from '" + rancherRelease.BaseRegistry + "' to '" + rancherRelease.Registry + "'")
			stringContent := string(releaseAssets[*asset.Name])
			releaseAssets[*asset.Name] = []byte(strings.ReplaceAll(stringContent, rancherRelease.BaseRegistry, rancherRelease.Registry))
		}
	}
	fmt.Println("uploading artifacts")
	for name, content := range releaseAssets {
		fmt.Println("uploading: " + name)
		if rancherRelease.DryRun {
			fmt.Println("dry run, skipping upload")
			continue
		}
		_, err := s3Uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: &rancherRelease.PrimeArtifactsBucket,
			Key:    aws.String(rancherRepo + "/" + name),
			Body:   bytes.NewReader(content),
		})
		if err != nil {
			return err
		}
	}
	return nil
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
    body { font-family: 'Courier New', monospace, Verdana, Geneneva; }
    header { display: flex; flex-direction: row; justify-items: center; }
    #rancher-logo { width: 200px; }
    .project { margin-left: 20px; }
    .release { margin-left: 40px; margin-bottom: 20px; }
    .release h3 { margin-bottom: 0px; }
    .files { margin-left: 60px; display: flex; flex-direction: column; }
    .release-title { display: flex; flex-direction: row; }
    .release-title-tag { margin-right: 20px; min-width: 70px; }
    .release-title-expand { background-color: #2453ff; color: white; border-radius: 5px; border: none; }
    .release-title-expand:hover, .expand-active{ background-color: white; color: #2453ff; border: 1px solid #2453ff; }
    .hidden { display: none; overflow: hidden; }
    </style>
  </head>
  <body>
    <header>
      <img src="https://prime.ribs.rancher.io/assets/img/rancher-suse-logo-horizontal-color.svg" alt="rancher logo" id="rancher-logo" />
      <h1>PRIME ARTIFACTS</h1>
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
          <div class="files" id="release-{{ $version }}-files">
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
    hideFiles()
    function expand(tag) {
      const filesId = "release-" + tag + "-files"
      const expandButtonId = "release-" + tag + "-expand"
      document.getElementById(filesId).classList.toggle("hidden")
      document.getElementById(expandButtonId).classList.toggle("expand-active")
    }
    function hideFiles() {
        const fileDivs = document.querySelectorAll(".files")
        fileDivs.forEach(f => f.classList.add("hidden"))
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
