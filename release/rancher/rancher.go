package rancher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"io/ioutil"
	"log"
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

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-github/v39/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	ecmLog "github.com/rancher/ecm-distro-tools/log"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

const (
	rancherOrg                    = "rancher"
	rancherRepo                   = rancherOrg
	rancherImagesBaseURL          = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName         = "/rancher-images.txt"
	rancherHelmRepositoryURL      = "https://releases.rancher.com/server-charts/latest/index.yaml"
	rancherArtifactsBucket        = "prime-artifacts"
	rancherArtifactsPrefix        = "rancher/v"
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

var regsyncDefaultMediaTypes = []string{
	"application/vnd.docker.distribution.manifest.v2+json",
	"application/vnd.docker.distribution.manifest.list.v2+json",
	"application/vnd.oci.image.manifest.v1+json",
	"application/vnd.oci.image.index.v1+json",
}

var registriesInfo = map[string]registryInfo{
	"registry.rancher.com": {
		BaseURL:     rancherRegistryBaseURL,
		AuthURL:     sccSUSEURL,
		Service:     sccSUSEService,
		UserEnv:     `{{env "PRIME_REGISTRY_USERNAME"}}`,
		PasswordEnv: `{{env "PRIME_REGISTRY_PASSWORD"}}`,
	},
	"stgregistry.suse.com": {
		BaseURL:     stagingRancherRegistryBaseURL,
		AuthURL:     stagingSccSUSEURL,
		Service:     sccSUSEService,
		UserEnv:     `{{env "STAGING_REGISTRY_USERNAME"}}`,
		PasswordEnv: `{{env "STAGING_REGISTRY_PASSWORD"}}`,
	},
	"docker.io": {
		BaseURL:     dockerRegistryURL,
		AuthURL:     dockerAuthURL,
		Service:     dockerService,
		UserEnv:     `{{env "DOCKERIO_REGISTRY_USERNAME"}}`,
		PasswordEnv: `{{env "DOCKERIO_REGISTRY_PASSWORD"}}`,
	},
}

type registryInfo struct {
	BaseURL     string
	AuthURL     string
	Service     string
	UserEnv     string
	PasswordEnv string
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

type regsyncConfig struct {
	Version  int             `yaml:"version"`
	Creds    []regsyncCreds  `yaml:"creds"`
	Defaults regsyncDefaults `yaml:"defaults"`
	Sync     []regsyncSync   `yaml:"sync"`
}

type regsyncCreds struct {
	Registry string `yaml:"registry"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
}

type regsyncDefaults struct {
	Parallel   int      `yaml:"parallel"`
	MediaTypes []string `yaml:"mediaTypes"`
}

type regsyncTags struct {
	Allow []string `yaml:"allow"`
}

type regsyncSync struct {
	Source string      `yaml:"source"`
	Target string      `yaml:"target"`
	Type   string      `yaml:"type"`
	Tags   regsyncTags `yaml:"tags"`
}

type releaseAnnnouncement struct {
	Tag              string
	PreviousTag      string
	RancherRepoOwner string
	CommitSHA        string
	ActionRunID      string
	ImagesWithRC     []string
	ComponentsWithRC []string
	UIVersion        string
	CLIVersion       string
}

func listS3Objects(ctx context.Context, s3Client *s3.Client, bucketName string, prefix string) ([]string, error) {
	var keys []string
	var continuationToken *string
	isTruncated := true
	for isTruncated {
		objects, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            &bucketName,
			Prefix:            &prefix,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}
		for _, object := range objects.Contents {
			keys = append(keys, *object.Key)
		}
		// used for pagination
		continuationToken = objects.NextContinuationToken
		// if the bucket has more keys
		if objects.IsTruncated != nil && !*objects.IsTruncated {
			isTruncated = false
		}
	}
	return keys, nil
}

func GeneratePrimeArtifactsIndex(ctx context.Context, path string, ignoreVersions []string, s3Client *s3.Client) error {
	ignore := make(map[string]bool, len(ignoreVersions))
	for _, v := range ignoreVersions {
		ignore[v] = true
	}
	keys, err := listS3Objects(ctx, s3Client, rancherArtifactsBucket, rancherArtifactsPrefix)
	if err != nil {
		return err
	}
	content := generateArtifactsIndexContent(keys, ignore)
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

func generateArtifactsIndexContent(keys []string, ignoreVersions map[string]bool) ArtifactsIndexContent {
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

	for _, key := range keys {
		if !strings.Contains(key, "rancher/") {
			continue
		}
		keyFile := strings.Split(strings.TrimPrefix(key, "rancher/"), "/")
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

func GenerateMissingImagesList(imagesListURL, registry string, concurrencyLimit int, checkImages, ignoreImages []string, verbose bool) ([]string, error) {
	log := ecmLog.NewLogger(verbose)
	if len(checkImages) == 0 {
		if imagesListURL == "" {
			return nil, errors.New("if no images are provided, an images list URL must be provided")
		}
		rancherImages, err := remoteTextFileToSlice(imagesListURL)
		if err != nil {
			return nil, errors.New("failed to get rancher images: " + err.Error())
		}
		checkImages = append(checkImages, rancherImages...)
	}

	ignore, err := imageSliceToMap(ignoreImages)
	if err != nil {
		return nil, err
	}

	// create an error group with a limit to prevent accidentaly doing a DOS attack against our registry
	ctx, cancel := context.WithCancel(context.Background())
	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.SetLimit(concurrencyLimit)
	missingImagesChan := make(chan string, len(checkImages))

	// auth tokens can be reused, but maps need a lock for reading and writing in go routines
	repositoryAuths := make(map[string]string)
	mu := sync.RWMutex{}

	rgInfo, ok := registriesInfo[registry]
	if !ok {
		return nil, errors.New("registry must be one of the following: 'docker.io', 'registry.rancher.com' or 'stgregistry.suse.com'")
	}

	for _, imageAndVersion := range checkImages {
		image, imageVersion, err := splitImageAndVersion(imageAndVersion)
		if err != nil {
			cancel()
			return nil, err
		}

		if _, ok := ignore[image]; ok {
			log.Println("skipping ignored image: " + imageAndVersion)
			continue
		}

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
						auth, err = registryAuth(rgInfo.AuthURL, rgInfo.Service, image)
						if err != nil {
							cancel()
							return err
						}
						repositoryAuths[image] = auth
					}
					mu.Unlock()

					exists, err := checkIfImageExists(rgInfo.BaseURL, image, imageVersion, auth)
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

func GenerateImagesSyncConfig(images []string, sourceRegistry, targetRegistry, outputPath string) error {
	config, err := generateRegsyncConfig(images, sourceRegistry, targetRegistry)
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return yaml.NewEncoder(f).Encode(config)
}

func generateRegsyncConfig(images []string, sourceRegistry, targetRegistry string) (*regsyncConfig, error) {
	sourceRegistryInfo, ok := registriesInfo[sourceRegistry]
	if !ok {
		return nil, errors.New("invalid source registry")
	}
	targetRegistryInfo, ok := registriesInfo[targetRegistry]
	if !ok {
		return nil, errors.New("invalid target registry")
	}

	config := regsyncConfig{
		Version: 1,
		Creds: []regsyncCreds{
			{
				Registry: sourceRegistry,
				User:     sourceRegistryInfo.UserEnv,
				Pass:     sourceRegistryInfo.PasswordEnv,
			},
			{
				Registry: targetRegistry,
				User:     targetRegistryInfo.UserEnv,
				Pass:     targetRegistryInfo.PasswordEnv,
			},
		},
		Defaults: regsyncDefaults{
			Parallel:   1,
			MediaTypes: regsyncDefaultMediaTypes,
		},
		Sync: make([]regsyncSync, len(images)),
	}

	for i, imageAndVersion := range images {
		image, imageVersion, err := splitImageAndVersion(imageAndVersion)
		if err != nil {
			return nil, err
		}
		config.Sync[i] = regsyncSync{
			Source: sourceRegistry + "/" + image,
			Target: targetRegistry + "/" + image,
			Type:   "repository",
			Tags:   regsyncTags{Allow: []string{imageVersion}},
		}
	}
	return &config, nil
}

func imageSliceToMap(images []string) (map[string]bool, error) {
	imagesMap := make(map[string]bool, len(images))
	for _, image := range images {
		if err := validateRepoImage(image); err != nil {
			return nil, err
		}
		imagesMap[image] = true
	}
	return imagesMap, nil
}

// splitImageAndVersion will validate the image format and return
// repo/image, version and any validation errors
// e.g: rancher/rancher-agent:v2.9.0
func splitImageAndVersion(image string) (string, string, error) {
	if !strings.Contains(image, ":") {
		return "", "", errors.New("malformed image name, missing ':' " + image)
	}
	splitImage := strings.Split(image, ":")
	repoImage := splitImage[0]
	if err := validateRepoImage(repoImage); err != nil {
		return "", "", err
	}
	imageVersion := splitImage[1]
	return repoImage, imageVersion, nil
}

// validateRepoImage will validate that a given string only contains
// the repo and image names and not the version. e.g: rancher/rancher
func validateRepoImage(repoImage string) error {
	if !strings.Contains(repoImage, "/") {
		return errors.New("malformed image name, missing '/' " + repoImage)
	}
	if strings.Contains(repoImage, ":") {
		return errors.New("malformed image name, the repo and image name shouldn't contain versions: " + repoImage)
	}
	return nil
}

func GenerateDockerImageDigests(outputFile, imagesFileURL, registry string, verbose bool) error {
	imagesDigests, err := dockerImagesDigests(imagesFileURL, registry)
	if err != nil {
		return err
	}
	return createAssetFile(outputFile, imagesDigests)
}

func GenerateAnnounceReleaseMessage(ctx context.Context, ghClient *github.Client, tag, previousTag, rancherRepoOwner, actionRunID string, primeOnly bool) (string, error) {
	release, _, err := ghClient.Repositories.GetReleaseByTag(ctx, rancherRepoOwner, rancherRepo, tag)
	if err != nil {
		return "", err
	}
	if release.TargetCommitish == nil {
		return "", errors.New("release commit sha is nil")
	}

	r := releaseAnnnouncement{
		Tag:              tag,
		PreviousTag:      previousTag,
		RancherRepoOwner: rancherRepoOwner,
		CommitSHA:        *release.TargetCommitish,
		ActionRunID:      actionRunID,
	}

	announceTemplate := announceReleaseGATemplate

	// only prerelease images like alpha, rc or hotfix contain - (e.g: v2.9.2-alpha1)
	isPreRelease := strings.ContainsRune(tag, '-')
	if isPreRelease {
		announceTemplate = announceReleasePreReleaseTemplate

		componentsURL := "https://github.com/" + rancherRepoOwner + "/rancher/releases/download/" + tag + "/rancher-components.txt"
		if primeOnly {
			componentsURL = rancherArtifactsBaseURL + "/rancher/" + tag + "/rancher-components.txt"
		}
		rancherComponents, err := remoteTextFileToSlice(componentsURL)
		if err != nil {
			return "", err
		}

		imagesWithRC, componentsWithRC, err := rancherImagesComponentsWithRC(rancherComponents)
		if err != nil {
			return "", err
		}
		r.ImagesWithRC = imagesWithRC
		r.ComponentsWithRC = componentsWithRC
	} else { // GA Version
		dockerfileURL := "https://raw.githubusercontent.com/" + rancherRepoOwner + "/rancher/" + *release.TargetCommitish + "/package/Dockerfile"
		dockerfile, err := remoteTextFileToSlice(dockerfileURL)
		if err != nil {
			return "", err
		}
		uiVersion, cliVersion, err := rancherUICLIVersions(dockerfile)
		if err != nil {
			return "", err
		}
		r.UIVersion = uiVersion
		r.CLIVersion = cliVersion
	}

	tmpl := template.New("announce-release")
	tmpl, err = tmpl.Parse(announceTemplate)
	if err != nil {
		return "", errors.New("failed to parse announce template: " + err.Error())
	}
	buff := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buff, "announceRelease", r); err != nil {
		return "", err
	}
	return buff.String(), nil
}

// rancherUICLIVersions scans a dockerfile line by line and retruns the ui and cli versions, or an error if any of them are not found
func rancherUICLIVersions(dockerfile []string) (string, string, error) {
	var uiVersion string
	var cliVersion string
	for _, line := range dockerfile {
		if strings.Contains(line, "ENV CATTLE_UI_VERSION ") {
			uiVersion = strings.TrimPrefix(line, "ENV CATTLE_UI_VERSION ")
			continue
		}
		if strings.Contains(line, "ENV CATTLE_CLI_VERSION ") {
			cliVersion = strings.TrimPrefix(line, "ENV CATTLE_CLI_VERSION ")
			continue
		}
		if len(uiVersion) > 0 && len(cliVersion) > 0 {
			break
		}
	}
	if uiVersion == "" || cliVersion == "" {
		return "", "", errors.New("missing ui or cli version")
	}
	return uiVersion, cliVersion, nil
}

// rancherImagesComponentsWithRC scans the rancher-components.txt file content and returns images and components, or an error
func rancherImagesComponentsWithRC(rancherComponents []string) ([]string, []string, error) {
	if len(rancherComponents) < 2 {
		return nil, nil, errors.New("rancher-components.txt should have at least two lines (images and components headers)")
	}
	images := make([]string, 0)
	components := make([]string, 0)

	var isImage bool
	for _, line := range rancherComponents {
		// always skip empty lines
		if line == "" || line == " " {
			continue
		}

		// if a line contains # it is a header for a section
		isHeader := strings.Contains(line, "#")

		if isHeader {
			imagesHeader := strings.Contains(line, "Images")
			componentsHeader := strings.Contains(line, "Components")
			// if it's a header, but not for images or components, ignore it and everything else after it
			if !imagesHeader && !componentsHeader {
				break
			}
			// isImage's value will persist between iterations
			// if imagesHeader is true, it means that all following lines are images
			// if it's false, it means that all following images are components
			isImage = imagesHeader
			continue
		}

		if isImage {
			images = append(images, line)
		} else {
			components = append(components, line)
		}
	}
	return images, components, nil
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
		digest, _, err := dockerImageDigest(rgInfo.BaseURL, image, imageVersion, repositoryAuths[image])
		if err != nil {
			return nil, err
		}
		// e.g: registry.rancher.com/rancher/rancher:v2.9.0 = sha256:1234567890
		imagesDigests[registry+"/"+imageAndVersion] = digest
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
	defer res.Body.Close()

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
		return "", errors.New("expected status code to be 200, got: " + res.Status)
	}

	var auth registryAuthToken
	if err := json.NewDecoder(res.Body).Decode(&auth); err != nil {
		return "", err
	}

	return auth.Token, nil
}

func remoteTextFileToSlice(url string) ([]string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New("expected status code to be 200, got: " + res.Status)
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

const announceReleaseHeaderTemplate = "`{{ .Tag }}` is available based on this commit ([link](https://github.com/{{ .RancherRepoOwner }}/rancher/commit/{{ .CommitSHA }}))!\n" +
	"* Link of commits between last 2 RCs. ([link](https://github.com/{{ .RancherRepoOwner }}/rancher/compare/{{ .PreviousTag }}...{{ .Tag }}))\n" +
	"* Completed GHA build ([link](https://github.com/{{ .RancherRepoOwner }}/rancher/actions/runs/{{ .ActionRunID }})).\n"

const announceReleasePreReleaseTemplate = `{{ define "announceRelease" }}` + announceReleaseHeaderTemplate +
	"* Images with -rc:\n" +
	"{{ range .ImagesWithRC }}" +
	"    * {{ . }}\n{{ end }}" +
	"* Components with -rc:\n" +
	"{{ range .ComponentsWithRC }}" +
	"    * {{ . }}\n{{ end }}{{ end }}"

const announceReleaseGATemplate = `{{  define "announceRelease" }}` + announceReleaseHeaderTemplate +
	"* UI Version: `{{ .UIVersion }}`\n" +
	"* CLI Version: `{{ .CLIVersion }}`{{ end }}"
