package rancher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/go-github/v78/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/cli"
	"github.com/rancher/ecm-distro-tools/repository"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"
)

const (
	rancherOrg                    = "rancher"
	rancherRepo                   = rancherOrg
	rancherRegistryBaseURL        = "https://registry.rancher.com"
	stagingRancherRegistryBaseURL = "https://stgregistry.suse.com"
	sccSUSEURL                    = "https://scc.suse.com/api/registry/authorize"
	stagingSccSUSEURL             = "https://stgscc.suse.com/api/registry/authorize"
	dockerRegistryURL             = "https://registry-1.docker.io"
	dockerAuthURL                 = "https://auth.docker.io/token"
	sccSUSEService                = "SUSE+Linux+Docker+Registry"
	dockerService                 = "registry.docker.io"
	dashboardUpdateRefsBranchBase = "update-dashboard-refs"
)

type ReleaseType int

const (
	ReleaseTypePreRelease ReleaseType = iota
	ReleaseTypeGA
)

var ReleaseTypes = map[string]ReleaseType{
	"alpha": ReleaseTypePreRelease,
	"rc":    ReleaseTypePreRelease,
	"ga":    ReleaseTypeGA,
}

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

type registryAuthToken struct {
	Token string `json:"token"`
}

type regsyncConfig struct {
	Version  int             `json:"version"`
	Creds    []regsyncCreds  `json:"creds"`
	Defaults regsyncDefaults `json:"defaults"`
	Sync     []regsyncSync   `json:"sync"`
}

type regsyncCreds struct {
	Registry string `json:"registry"`
	User     string `json:"user"`
	Pass     string `json:"pass"`
}

type regsyncDefaults struct {
	Parallel   int      `json:"parallel"`
	MediaTypes []string `json:"mediaTypes"`
}

type regsyncTags struct {
	Allow []string `json:"allow"`
}

type regsyncSync struct {
	Source string      `json:"source"`
	Target string      `json:"target"`
	Type   string      `json:"type"`
	Tags   regsyncTags `json:"tags"`
}

func UpdateDashboardReferences(ctx context.Context, ghClient *github.Client, r *ecmConfig.DashboardRelease, u *ecmConfig.User, tag, rancherReleaseBranch, rancherRepoName, rancherRepoOwner, rancherRepoURL string, dryRun bool) error {
	if err := updateDashboardReferencesAndPush(tag, rancherReleaseBranch, rancherRepoURL, dryRun); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	return createDashboardReferencesPR(ctx, ghClient, u, tag, rancherReleaseBranch, rancherRepoName, rancherRepoOwner)
}

func UpdateDashboardRefsBranchName(tag string) string {
	return dashboardUpdateRefsBranchBase + "-" + tag
}

func updateDashboardReferencesAndPush(tag, rancherReleaseBranch, rancherUpstreamURL string, dryRun bool) error {
	updateScriptVars := map[string]string{
		"Tag":                  tag,
		"RancherReleaseBranch": rancherReleaseBranch,
		"RancherUpstreamURL":   rancherUpstreamURL,
		"DryRun":               strconv.FormatBool(dryRun),
		"BranchBaseName":       UpdateDashboardRefsBranchName(tag),
	}
	updateScriptOut, err := ecmExec.RunTemplatedScript("./", "update_dashboard_refs.sh", updateDashboardReferencesScript, nil, updateScriptVars)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createDashboardReferencesPR(ctx context.Context, ghClient *github.Client, u *ecmConfig.User, tag, rancherReleaseBranch, rancherRepoName, rancherRepoOwner string) error {
	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Bump Dashboard to `%s`", tag)),
		Base:                github.String(rancherReleaseBranch),
		Head:                github.String(u.GithubUsername + ":" + UpdateDashboardRefsBranchName(tag)),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	pr, _, err := ghClient.PullRequests.Create(ctx, rancherRepoOwner, rancherRepoName, pull)
	if err != nil {
		return err
	}

	fmt.Println("Pull Request created successfully:", pr.GetHTMLURL())

	return nil
}

func UpdateCLIReferences(ctx context.Context, ghClient *github.Client, tag, rancherReleaseBranch, githubUsername, rancherRepoName, rancherRepoOwner, rancherUpstreamURL string, dryRun bool) error {
	if err := updateCLIReferencesAndPush(tag, rancherUpstreamURL, rancherReleaseBranch, dryRun); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	return createCLIReferencesPR(ctx, ghClient, tag, rancherReleaseBranch, githubUsername, rancherRepoName, rancherRepoOwner)
}

func updateCLIReferencesAndPush(tag, rancherUpstreamURL, rancherReleaseBranch string, dryRun bool) error {
	updateScriptVars := map[string]string{
		"DryRun":               strconv.FormatBool(dryRun),
		"BranchName":           cli.UpdateCLIRefsBranchName(tag),
		"Tag":                  tag,
		"RancherUpstreamURL":   rancherUpstreamURL,
		"RancherReleaseBranch": rancherReleaseBranch,
	}
	updateScriptOut, err := ecmExec.RunTemplatedScript("./", "replace_cli_ref.sh", updateCLIReferencesScript, nil, updateScriptVars)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createCLIReferencesPR(ctx context.Context, ghClient *github.Client, tag, rancherReleaseBranch, githubUsername, rancherRepoName, rancherRepoOwner string) error {
	pull := &github.NewPullRequest{
		Title:               github.String("Bump Rancher CLI version to " + tag),
		Base:                github.String(rancherReleaseBranch),
		Head:                github.String(githubUsername + ":" + cli.UpdateCLIRefsBranchName(tag)),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	pr, _, err := ghClient.PullRequests.Create(ctx, rancherRepoOwner, rancherRepoName, pull)
	if err != nil {
		return err
	}

	fmt.Println("Pull Request created successfully:", pr.GetHTMLURL())

	return nil
}

// ReleaseBranchFromTag generates the rancher release branch for a release line with the format of 'release/v{major}.{minor}'. The generated release branch might not be valid depending on multiple factors that cannot be treated on this function such as it being 'main'.
// Please make sure that this is the expected format before using the generated release branch.
func ReleaseBranchFromTag(tag string) (string, error) {
	majorMinor := semver.MajorMinor(tag)

	if majorMinor == "" {
		return "", errors.New("the tag isn't a valid semver: " + tag)
	}

	releaseBranch := "release/" + majorMinor

	return releaseBranch, nil
}

// CreateRelease gets the latest commit in a release branch, checks if CI is passing and creates a github release, returning the created release HTML URL or an error
func CreateRelease(ctx context.Context, ghClient *github.Client, r *ecmConfig.RancherRelease, opts *repository.CreateReleaseOpts, preRelease bool, releaseType string) (string, error) {
	if !semver.IsValid(opts.Tag) {
		return "", errors.New("the tag isn't a valid semver: " + opts.Tag)
	}
	if _, ok := ReleaseTypes[releaseType]; !ok {
		return "", errors.New("invalid release type")
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

// ImagesLocations searches for missing images in a registry and creates a map with the locations of the images, or if they are missing
// this map can be used to identify where which image should be synced from
func ImagesLocations(username, password string, concurrencyLimit int, checkImages, ignoreImages []string, targetRegistry string, imagesRegiestries []string) (map[string][]string, error) {
	imagesLocations := make(map[string][]string)

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

	if lastMissingImages == nil {
		lastMissingImages = make([]string, 0)
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

// MissingImagesFromRegistry receives registry information and a list of images and checks which images are missing from that registry
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

	b, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, b, 0644)
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

func ImagesFromArtifact(url string) ([]string, error) {
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
set -ex
OS=$(uname -s)
# Set variables (these are populated by Go's template engine)
DRY_RUN="{{ .DryRun }}"
BRANCH_NAME="{{ .BranchBaseName }}"
VERSION="{{ .Tag }}"
RANCHER_BRANCH="{{.RancherReleaseBranch}}"
RANCHER_UPSTREAM_URL="{{ .RancherUpstreamURL }}"
FILENAME="package/Dockerfile"

# Add upstream remote if it doesn't exist
# Note: Using ls | grep is not recommended for general use, but it's okay here
# since we're only checking for 'rancher'
git remote -v | grep -w upstream || git remote add upstream "${RANCHER_UPSTREAM_URL}"
git fetch upstream
git stash

# Delete the branch if it exists, then create a new one based on upstream
git branch -D "${BRANCH_NAME}" > /dev/null 2>&1 || true
git checkout -B "${BRANCH_NAME}" "upstream/${RANCHER_BRANCH}"
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
    ${_update_file_sed_cmd} "s/ENV CATTLE_UI_VERSION=.*/ENV CATTLE_UI_VERSION=${VERSION#v}/" "${FILENAME}"

    # Update CATTLE_DASHBOARD_UI_VERSION
    ${_update_file_sed_cmd} "s/ENV CATTLE_DASHBOARD_UI_VERSION=.*/ENV CATTLE_DASHBOARD_UI_VERSION=${VERSION}/" "${FILENAME}"
}

# Run the update function
update_file

git add $FILENAME
git commit --signoff -m "Update Dashboard refs to ${VERSION}"

if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi

# Cleaning temp files/scripts
git clean -f`

const updateCLIReferencesScript = `#!/bin/sh
set -ex
OS=$(uname -s)

# Set variables (these are populated by Go's template engine)
DRY_RUN="{{ .DryRun }}"
BRANCH_NAME="{{ .BranchName }}"
VERSION="{{ .Tag }}"
FILENAME="package/Dockerfile"
RANCHER_UPSTREAM_URL="{{ .RancherUpstreamURL }}"
RANCHER_RELEASE_BRANCH="{{ .RancherReleaseBranch }}"

# Add upstream remote if it doesn't exist
# Note: Using ls | grep is not recommended for general use, but it's okay here
# since we're only checking for 'rancher'
git remote -v | grep -w upstream || git remote add upstream "$RANCHER_UPSTREAM_URL"
git fetch upstream
git stash

# Delete the branch if it exists, then create a new one based on upstream
git branch -D "${BRANCH_NAME}" > /dev/null 2>&1 || true
git checkout -B "${BRANCH_NAME}" "upstream/$RANCHER_RELEASE_BRANCH"
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
    ${_update_file_sed_cmd} "s/ENV CATTLE_CLI_VERSION=.*/ENV CATTLE_CLI_VERSION=${VERSION}/" "${FILENAME}"
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
