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
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"
)

const (
	rancherOrg                    = "rancher"
	rancherRepo                   = rancherOrg
	rancherImagesBaseURL          = "https://github.com/rancher/rancher/releases/download/"
	rancherImagesFileName         = "/rancher-images.txt"
	rancherHelmRepositoryURL      = "https://releases.rancher.com/server-charts/latest/index.yaml"
	rancherArtifactsBucket        = "prime-artifacts"
	rancherArtifactsPrefix        = "rancher/v"
	rke2ArtifactsPrefix           = "rke2/v"
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

type ArtifactsIndexVersions struct {
	Versions      []string            `json:"versions"`
	VersionsFiles map[string][]string `json:"versionsFiles"`
}

type ArtifactsIndexContentGroup struct {
	Rancher ArtifactsIndexVersions
	RKE2    ArtifactsIndexVersions
	BaseURL string `json:"baseUrl"`
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
	rancherKeys, err := listS3Objects(ctx, s3Client, rancherArtifactsBucket, rancherArtifactsPrefix)
	if err != nil {
		return err
	}
	rke2Keys, err := listS3Objects(ctx, s3Client, rancherArtifactsBucket, rke2ArtifactsPrefix)
	if err != nil {
		return err
	}
	content := generateArtifactsIndexContent(rancherKeys, rke2Keys, ignore)
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

func UpdateDashboardReferences(ctx context.Context, cfg *ecmConfig.Dashboard, ghClient *github.Client, r *ecmConfig.DashboardRelease, u *ecmConfig.User) error {
	r.RancherUpstreamURL = cfg.RancherUpstreamURL

	if err := updateDashboardReferencesAndPush(r, u); err != nil {
		return err
	}

	return createDashboardReferencesPR(ctx, cfg, ghClient, r, u)
}

func updateDashboardReferencesAndPush(r *ecmConfig.DashboardRelease, _ *ecmConfig.User) error {
	fmt.Println("verifying if workspace dir exists")
	funcMap := template.FuncMap{"replaceAll": strings.ReplaceAll}
	fmt.Println("creating update dashboard references script template")
	updateScriptOut, err := ecmExec.RunTemplatedScript("./", "replase_dash_ref.sh", updateDashboardReferencesScript, funcMap, r)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createDashboardReferencesPR(ctx context.Context, cfg *ecmConfig.Dashboard, ghClient *github.Client, r *ecmConfig.DashboardRelease, u *ecmConfig.User) error {
	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Bump Dashboard to `%s`", r.Tag)),
		Base:                github.String(r.RancherReleaseBranch),
		Head:                github.String(u.GithubUsername + ":update-build-refs-" + r.Tag),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	pr, _, err := ghClient.PullRequests.Create(ctx, cfg.RancherRepoOwner, cfg.RancherRepoName, pull)
	if err != nil {
		return err
	}

	fmt.Println("Pull Request created successfully:", pr.GetHTMLURL())

	return nil
}

func UpdateCLIReferences(ctx context.Context, cfg *ecmConfig.CLI, ghClient *github.Client, r *ecmConfig.CLIRelease, u *ecmConfig.User) error {
	r.RancherUpstreamURL = cfg.RancherUpstreamURL

	if err := updateCLIReferencesAndPush(r, u); err != nil {
		return err
	}

	return createCLIReferencesPR(ctx, cfg, ghClient, r, u)
}

func updateCLIReferencesAndPush(r *ecmConfig.CLIRelease, _ *ecmConfig.User) error {
	funcMap := template.FuncMap{"replaceAll": strings.ReplaceAll}
	fmt.Println("creating update cli references script template")
	updateScriptOut, err := ecmExec.RunTemplatedScript("./", "replace_cli_ref.sh", updateCLIReferencesScript, funcMap, r)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createCLIReferencesPR(ctx context.Context, cfg *ecmConfig.CLI, ghClient *github.Client, r *ecmConfig.CLIRelease, u *ecmConfig.User) error {
	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Bump Rancher CLI version to `%s`", r.Tag)),
		Base:                github.String(r.RancherReleaseBranch),
		Head:                github.String(u.GithubUsername + ":update-build-refs-" + r.Tag),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	pr, _, err := ghClient.PullRequests.Create(ctx, cfg.RancherRepoOwner, cfg.RancherRepoName, pull)
	if err != nil {
		return err
	}

	fmt.Println("Pull Request created successfully:", pr.GetHTMLURL())

	return nil
}

func generateArtifactsIndexContent(rancherKeys, rke2Keys []string, ignoreVersions map[string]bool) ArtifactsIndexContent {
	indexContent := ArtifactsIndexContent{
		GA: ArtifactsIndexContentGroup{
			Rancher: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			RKE2: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			BaseURL: rancherArtifactsBaseURL,
		},
		PreRelease: ArtifactsIndexContentGroup{
			Rancher: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			RKE2: ArtifactsIndexVersions{
				Versions:      []string{},
				VersionsFiles: map[string][]string{},
			},
			BaseURL: rancherArtifactsBaseURL,
		},
	}

	indexContent.GA.Rancher, indexContent.PreRelease.Rancher = parseVersionsFromKeys(rancherKeys, "rancher/", ignoreVersions)
	indexContent.GA.RKE2, indexContent.PreRelease.RKE2 = parseVersionsFromKeys(rke2Keys, "rke2/", ignoreVersions)

	return indexContent
}

// parseVersionsFromKeys extracts versions and files from keys and returns GA and pre-release version structs
func parseVersionsFromKeys(keys []string, prefix string, ignoreVersions map[string]bool) (ArtifactsIndexVersions, ArtifactsIndexVersions) {
	var versions []string
	versionsFiles := make(map[string][]string)

	gaVersions := ArtifactsIndexVersions{
		Versions:      []string{},
		VersionsFiles: map[string][]string{},
	}

	preReleaseVersions := ArtifactsIndexVersions{
		Versions:      []string{},
		VersionsFiles: map[string][]string{},
	}

	for _, key := range keys {
		if !strings.Contains(key, prefix) {
			continue
		}
		keyFile := strings.Split(strings.TrimPrefix(key, prefix), "/")
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
		// only non ga releases contains '-' e.g: -rc, -hotfix
		if strings.Contains(version, "-") {
			preReleaseVersions.Versions = append(preReleaseVersions.Versions, version)
			preReleaseVersions.VersionsFiles[version] = versionsFiles[version]
		} else {
			gaVersions.Versions = append(gaVersions.Versions, version)
			gaVersions.VersionsFiles[version] = versionsFiles[version]
		}
	}

	return gaVersions, preReleaseVersions
}

func generatePrimeArtifactsHTML(content ArtifactsIndexContentGroup) ([]byte, error) {
	tmpl, err := htmlTemplate.New("release-artifacts-index").Parse(artifactsIndexTemplate)
	if err != nil {
		return nil, err
	}
	buff := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buff, "release-artifacts-index", content); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// CreateRelease gets the latest commit in a release branch, checks if CI is passing and creates a github release, returning the created release HTML URL or an error
func CreateRelease(ctx context.Context, ghClient *github.Client, r *ecmConfig.RancherRelease, opts *repository.CreateReleaseOpts, preRelease bool, releaseType string) (string, error) {
	if !semver.IsValid(opts.Tag) {
		return "", errors.New("the tag isn't a valid semver: " + opts.Tag)
	}

	releaseName := opts.Tag
	if preRelease {
		latestVersionNumber := 1
		latestVersion, err := release.LatestPreRelease(ctx, ghClient, opts.Owner, opts.Repo, opts.Tag, releaseType)
		if err != nil {
			return "", err
		}

		if latestVersion != nil {
			trimmedVersionNumber := strings.TrimPrefix(*latestVersion, opts.Tag+"-"+releaseType)
			currentVersionNumber, err := strconv.Atoi(trimmedVersionNumber)
			if err != nil {
				return "", errors.New("failed to parse trimmed latest version number: " + err.Error())
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

	createdRelease, err := repository.CreateRelease(ctx, ghClient, opts)

	// GetHTMLURL will return an empty value if it isn't present
	return createdRelease.GetHTMLURL(), err
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

// ImagesLocationsMap searches for missing images in a registry and creates a map with the locations of the images, or if they are missing
// this map can be used to identify where which image should be synced from
func ImagesLocationsMap(username, password string, concurrencyLimit int, checkImages, ignoreImages []string, targetRegistry string, imagesRegiestries []string) (map[string][]string, error) {
	imagesLocations := make(map[string][]string)

	var err error
	missingFromTarget, err := MissingImagesFromRegistry(username, password, targetRegistry, concurrencyLimit, checkImages, ignoreImages)
	if err != nil {
		return nil, err
	}

	lastMissingImages := missingFromTarget
	for _, registry := range imagesRegiestries {
		missingFromRegistry, err := MissingImagesFromRegistry(username, password, registry, concurrencyLimit, lastMissingImages, ignoreImages)
		if err != nil {
			return nil, err
		}

		imagesLocations[registry], err = imagesDiff(lastMissingImages, missingFromRegistry)
		if err != nil {
			return nil, err
		}

		lastMissingImages = missingFromRegistry
	}

	imagesLocations["missing"] = lastMissingImages

	return imagesLocations, nil
}

// imagesDiff compares two images slices and returns a slice with all images that are in the source slice, but not in the compare slice
func imagesDiff(source, compare []string) ([]string, error) {
	cm, err := imageSliceToMap(compare, false)
	if err != nil {
		return nil, err
	}

	diff := make([]string, 0)

	for _, s := range source {
		if _, ok := cm[s]; !ok {
			diff = append(diff, s)
		}
	}

	return diff, nil
}

// missingImagesFromRegistry receives registry information and a list of images and checks which images are missing from that registry
// it uses the docker http api v2 to check images concurrently
func MissingImagesFromRegistry(username, password, registry string, concurrencyLimit int, checkImages, ignoreImages []string) ([]string, error) {
	ignore, err := imageSliceToMap(ignoreImages, true)
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
		cancel()
		return nil, errors.New("registry must be one of the following: 'docker.io', 'registry.rancher.com' or 'stgregistry.suse.com'")
	}

	for _, imageAndVersion := range checkImages {
		image, imageVersion, err := splitImageAndVersion(imageAndVersion)
		if err != nil {
			cancel()
			return nil, err
		}

		if _, ok := ignore[image]; ok {
			continue
		}

		func(ctx context.Context, missingImagesChan chan string, image, imageVersion, username, password string, repositoryAuths map[string]string, mu *sync.RWMutex) {
			errGroup.Go(func() error {
				// if any other check failed, stop running to prevent wasting resources
				// this doesn't include 404's since it is expected. Any other errors are included
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
						auth, err = registryAuth(rgInfo.AuthURL, rgInfo.Service, image, username, password)
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
					}
					return nil
				}
			})
		}(ctx, missingImagesChan, image, imageVersion, username, password, repositoryAuths, &mu)

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

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, config)
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

func imageSliceToMap(images []string, validate bool) (map[string]bool, error) {
	imagesMap := make(map[string]bool, len(images))
	for _, image := range images {
		if validate {
			if err := validateRepoImage(image); err != nil {
				return nil, err
			}
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

func GenerateDockerImageDigests(outputFile, imagesFileURL, registry, username, password string, verbose bool) error {
	imagesDigests, err := dockerImagesDigests(imagesFileURL, registry, username, password)
	if err != nil {
		return err
	}
	return createAssetFile(outputFile, imagesDigests)
}

func dockerImagesDigests(imagesFileURL, registry, username, password string) (imageDigest, error) {
	imagesList, err := artifactImageList(imagesFileURL, registry)
	if err != nil {
		return nil, err
	}

	rgInfo, ok := registriesInfo[registry]
	if !ok {
		return nil, errors.New("registry must be one of the following: 'docker.io', 'registry.rancher.com' or 'stgregistry.suse.com'")
	}

	imagesDigests := make(imageDigest)
	repositoryAuths := make(map[string]string)

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
			auth, err := registryAuth(rgInfo.AuthURL, rgInfo.Service, image, username, password)
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
	lines, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return []string{}, errors.New("file was empty")
	}

	return strings.Split(string(lines), "\n"), nil
}

func dockerImageDigest(registryBaseURL, img, imgVersion, auth string) (string, int, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
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

func registryAuth(authURL, service, image, username, password string) (string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	scope := "repository:" + image + ":pull"
	url := authURL + "?scope=" + scope + "&service=" + service
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if len(username) > 1 && len(password) > 1 {
		req.SetBasicAuth(username, password)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("expected status code to be 200, got: " + strconv.Itoa(res.StatusCode))
	}
	defer res.Body.Close()

	var auth registryAuthToken
	if err := json.NewDecoder(res.Body).Decode(&auth); err != nil {
		return "", err
	}

	return auth.Token, nil
}

func ImagesListFromArtifact(url string) ([]string, error) {
	httpClient := ecmHTTP.NewClient(time.Second * 15)
	res, err := httpClient.Get(url)
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

const artifactsIndexTemplate = `{{ define "release-artifacts-index" }}
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
        {{ range $i, $version := .Rancher.Versions }}
        <div class="release-{{ $version }} release">
          <div class="release-title">
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.Rancher.VersionsFiles $version }}
              <li><a href="{{ $.BaseURL }}/rancher/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rancher/{{ $version }}/{{ . }}</a></li>
              {{ end }}
            </ul>
          </div>
        </div>
				{{ end }}
      </div>
	  <div class="project-rke2 project">
        <h2>rke2</h2>
        {{ range $i, $version := .RKE2.Versions }}
        <div class="release-{{ $version }} release">
          <div class="release-title">
						<b class="release-title-tag">{{ $version }}</b>
            <button onclick="expand('{{ $version }}')" id="release-{{ $version }}-expand" class="release-title-expand">expand</button>
          </div>
          <div class="files" id="release-{{ $version }}-files">
            <ul>
              {{ range index $.RKE2.VersionsFiles $version }}
              <li><a href="{{ $.BaseURL }}/rke2/{{ $version | urlquery }}/{{ . }}">{{ $.BaseURL }}/rke2/{{ $version }}/{{ . }}</a></li>
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

const updateDashboardReferencesScript = `#!/bin/sh
# Enable verbose mode and exit on any error
set -ex

# Determine the operating system
OS=$(uname -s)

# Set variables (these are populated by Go's template engine)
DRY_RUN={{ .DryRun }}
BRANCH_NAME=update-build-refs-{{ .Tag }}
VERSION={{ .Tag }}
FILENAME=package/Dockerfile

# Add upstream remote if it doesn't exist
# Note: Using ls | grep is not recommended for general use, but it's okay here
# since we're only checking for 'rancher'
git remote -v | grep -w upstream || git remote add upstream {{ .RancherUpstreamURL }}
git fetch upstream
git stash

# Delete the branch if it exists, then create a new one based on upstream
git branch -D "${BRANCH_NAME}" > /dev/null 2>&1 || true
git checkout -B "${BRANCH_NAME}" upstream/{{.RancherReleaseBranch}}
# git clean -xfd

# Function to update the file
update_file() {
    _update_file_sed_cmd=""

    # Set the appropriate sed command based on the OS
    case "${OS}" in
        Darwin)
            _update_file_sed_cmd="sed -i ''"
            ;;
        Linux)
            _update_file_sed_cmd="sed -i"
            ;;
        *)
            echo "$(OS) not supported yet" >&2
            exit 1
            ;;
    esac

    # Update CATTLE_UI_VERSION, removing leading 'v' if present (${VERSION#v} the '#v' removes the leading 'v')
    ${_update_file_sed_cmd} "s/ENV CATTLE_UI_VERSION .*/ENV CATTLE_UI_VERSION ${VERSION#v}/" "${FILENAME}"

    # Update CATTLE_DASHBOARD_UI_VERSION
    ${_update_file_sed_cmd} "s/ENV CATTLE_DASHBOARD_UI_VERSION .*/ENV CATTLE_DASHBOARD_UI_VERSION ${VERSION}/" "${FILENAME}"
}

# Run the update function
update_file

git add $FILENAME
git commit --signoff -m "Update to Dashboard refs to ${VERSION}"

# Push the changes if not a dry run
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi

# Cleaning temp files/scripts
git clean -f`

const updateCLIReferencesScript = `#!/bin/sh
# Enable verbose mode and exit on any error
set -ex

# Determine the operating system
OS=$(uname -s)

# Set variables (these are populated by Go's template engine)
DRY_RUN={{ .DryRun }}
BRANCH_NAME=update-cli-build-refs-{{ .Tag }}
VERSION={{ .Tag }}
FILENAME=package/Dockerfile

# Add upstream remote if it doesn't exist
# Note: Using ls | grep is not recommended for general use, but it's okay here
# since we're only checking for 'rancher'
git remote -v | grep -w upstream || git remote add upstream {{ .RancherUpstreamURL }}
git fetch upstream
git stash

# Delete the branch if it exists, then create a new one based on upstream
git branch -D "${BRANCH_NAME}" > /dev/null 2>&1 || true
git checkout -B "${BRANCH_NAME}" upstream/{{.RancherReleaseBranch}}
# git clean -xfd

# Function to update the file
update_file() {
    _update_file_sed_cmd=""

    # Set the appropriate sed command based on the OS
    case "${OS}" in
        Darwin)
            _update_file_sed_cmd="sed -i ''"
            ;;
        Linux)
            _update_file_sed_cmd="sed -i"
            ;;
        *)
            echo "$(OS) not supported yet" >&2
            exit 1
            ;;
    esac

    # Update CATTLE_CLI_VERSION
    ${_update_file_sed_cmd} "s/ENV CATTLE_CLI_VERSION .*/ENV CATTLE_CLI_VERSION ${VERSION}/" "${FILENAME}"
}

# Run the update function
update_file

git add $FILENAME
git commit --signoff -m "Update to Dashboard refs to ${VERSION}"

# Push the changes if not a dry run
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi

# Cleaning temp files/scripts
git clean -f`
