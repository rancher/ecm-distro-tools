package release

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/google/go-github/v39/github"
	httpecm "github.com/rancher/ecm-distro-tools/http"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/yaml"
)

const (
	k3sRepo                = "k3s"
	rke2Repo               = "rke2"
	uiRepo                 = "ui"
	dashboardRepo          = "dashboard"
	cliRepo                = "cli"
	alternateVersion       = "1.23"
	rke2ChartsVersionsFile = "chart_versions.yaml"
	defaultTimeout         = 30 * time.Second
)

type charts struct {
	Charts []chart `yaml:"charts"`
}

type chart struct {
	Version   string `yaml:"version"`
	Filename  string `yaml:"filename"`
	Bootstrap bool   `yaml:"bootstrap"`
}

type changeLogData struct {
	PrevMilestone string
	Content       []repository.ChangeLog
}

type releaseNoteData struct {
	Milestone        string
	MajorMinor       string
	ChangeLogVersion string
	ChangeLogData    changeLogData
}

type releaseNote interface {
	Fill(milestone string) error
	Template() string
	Repo() string
}

type rke2ReleaseNoteData struct {
	K8sVersion                            string
	EtcdVersion                           string
	ContainerdVersion                     string
	RuncVersion                           string
	MetricsServerVersion                  string
	CoreDNSVersion                        string
	IngressNginxVersion                   string
	HelmControllerVersion                 string
	FlannelVersion                        string
	CanalCalicoVersion                    string
	CanalCalicoURL                        string
	CalicoVersion                         string
	CalicoURL                             string
	CiliumVersion                         string
	MultusVersion                         string
	CiliumChartVersion                    string
	CanalChartVersion                     string
	CalicoChartVersion                    string
	CalicoCRDChartVersion                 string
	CoreDNSChartVersion                   string
	IngressNginxChartVersion              string
	MetricsServerChartVersion             string
	VsphereCSIChartVersion                string
	VsphereCPIChartVersion                string
	HarvesterCloudProviderChartVersion    string
	HarvesterCSIDriverChartVersion        string
	SnapshotControllerChartVersion        string
	SnapshotControllerCRDChartVersion     string
	SnapshotValidationWebhookChartVersion string
	TraefikImageVersion                   string
	TraefikVersion                        string
	TraefikCRDVersion                     string
	releaseNoteData
}

func (rd *rke2ReleaseNoteData) Fill(milestone string) error {
	var containerdVersion string

	if rd.MajorMinor == alternateVersion {
		containerdVersion = goModLibVersion(containerdV2ModLib, rke2Repo, milestone)
		if containerdVersion == "" {
			containerdVersion = goModLibVersion(containerdModLib, rke2Repo, milestone)
		}
	} else {
		containerdVersion = dockerfileVersion("hardened-containerd", rke2Repo, milestone)
	}

	rd.EtcdVersion = buildScriptVersion("ETCD_VERSION", rke2Repo, milestone)
	rd.RuncVersion = dockerfileVersion("hardened-runc", rke2Repo, milestone)
	rd.CanalCalicoVersion = imageTagVersion("hardened-calico", rke2Repo, milestone)
	rd.CanalCalicoURL = createCalicoURL(rd.CanalCalicoVersion)
	rd.CiliumVersion = imageTagVersion("cilium-cilium", rke2Repo, milestone)
	rd.ContainerdVersion = containerdVersion
	rd.MetricsServerVersion = imageTagVersion("metrics-server", rke2Repo, milestone)
	rd.IngressNginxVersion = imageTagVersion("nginx-ingress-controller", rke2Repo, milestone)
	rd.FlannelVersion = imageTagVersion("flannel", rke2Repo, milestone)
	rd.MultusVersion = imageTagVersion("multus-cni", rke2Repo, milestone)
	rd.CalicoVersion = imageTagVersion("calico-node", rke2Repo, milestone)
	rd.CalicoURL = createCalicoURL(rd.CalicoVersion)

	traefikImageVersion := imageTagVersion("hardened-traefik", rke2Repo, milestone)
	rd.TraefikImageVersion = strings.Split(traefikImageVersion, "-")[0] // removing the metadata to link to upstream traefik/traefik

	// get charts versions
	chartsData, err := rke2ChartsVersion(milestone)
	if err != nil {
		return err
	}

	rd.CiliumChartVersion = chartsData["rke2-cilium.yaml"].Version
	rd.CanalChartVersion = chartsData["rke2-canal.yaml"].Version
	rd.CalicoChartVersion = chartsData["rke2-calico.yaml"].Version
	rd.CalicoCRDChartVersion = chartsData["rke2-calico-crd.yaml"].Version
	rd.CoreDNSChartVersion = chartsData["rke2-coredns.yaml"].Version
	rd.IngressNginxChartVersion = chartsData["rke2-ingress-nginx.yaml"].Version
	rd.MetricsServerChartVersion = chartsData["rke2-metrics-server.yaml"].Version
	rd.VsphereCSIChartVersion = chartsData["rancher-vsphere-csi.yaml"].Version
	rd.VsphereCPIChartVersion = chartsData["rancher-vsphere-cpi.yaml"].Version
	rd.HarvesterCloudProviderChartVersion = chartsData["harvester-cloud-provider.yaml"].Version
	rd.HarvesterCSIDriverChartVersion = chartsData["harvester-csi-driver.yaml"].Version
	rd.SnapshotControllerChartVersion = chartsData["rke2-snapshot-controller.yaml"].Version
	rd.SnapshotControllerCRDChartVersion = chartsData["rke2-snapshot-controller-crd.yaml"].Version
	rd.SnapshotValidationWebhookChartVersion = chartsData["rke2-snapshot-validation-webhook.yaml"].Version
	rd.TraefikVersion = chartsData["rke2-traefik.yaml"].Version
	rd.TraefikCRDVersion = chartsData["rke2-traefik-crd.yaml"].Version

	return nil
}
func (_ *rke2ReleaseNoteData) Template() string { return rke2ReleaseNoteTemplate }
func (_ *rke2ReleaseNoteData) Repo() string     { return rke2Repo }

type k3sReleaseNoteData struct {
	K8sVersion                  string
	ChangeLogSince              string
	KineVersion                 string
	SQLiteVersion               string
	SQLiteVersionReplaced       string
	EtcdVersion                 string
	ContainerdVersion           string
	RuncVersion                 string
	FlannelVersion              string
	MetricsServerVersion        string
	TraefikVersion              string
	CoreDNSVersion              string
	HelmControllerVersion       string
	LocalPathProvisionerVersion string
	releaseNoteData
}

func (rd *k3sReleaseNoteData) Fill(milestone string) error {
	var runcVersion string
	var containerdVersion string

	if semver.Compare(rd.K8sVersion, "v1.24.0") == 1 && semver.Compare(rd.K8sVersion, "v1.26.5") == -1 {
		containerdVersion = buildScriptVersion("VERSION_CONTAINERD", k3sRepo, milestone)
	} else {
		containerdVersion = goModLibVersion(containerdV2ModLib, k3sRepo, milestone)
		if containerdVersion == "" {
			containerdVersion = goModLibVersion(containerdModLib, k3sRepo, milestone)
		}
	}

	if rd.MajorMinor == alternateVersion {
		runcVersion = buildScriptVersion("VERSION_RUNC", k3sRepo, milestone)
	} else {
		runcVersion = goModLibVersion("runc", k3sRepo, milestone)
	}

	rd.KineVersion = goModLibVersion("kine", k3sRepo, milestone)
	rd.EtcdVersion = goModLibVersion("etcd/api/v3", k3sRepo, milestone)
	rd.ContainerdVersion = containerdVersion
	rd.RuncVersion = runcVersion
	rd.FlannelVersion = goModLibVersion("flannel", k3sRepo, milestone)
	rd.MetricsServerVersion = imageTagVersion("metrics-server", k3sRepo, milestone)
	rd.TraefikVersion = imageTagVersion("traefik", k3sRepo, milestone)
	rd.LocalPathProvisionerVersion = imageTagVersion("local-path-provisioner", k3sRepo, milestone)

	return nil
}

func (_ *k3sReleaseNoteData) Template() string { return k3sReleaseNoteTemplate }
func (_ *k3sReleaseNoteData) Repo() string     { return k3sRepo }

type uiReleaseNoteData struct {
	releaseNoteData
}

func (_ *uiReleaseNoteData) Fill(_ string) error { return nil }
func (_ *uiReleaseNoteData) Template() string    { return fmt.Sprintf(defaultReleaseNoteTemplate, uiRepo) }
func (_ *uiReleaseNoteData) Repo() string        { return uiRepo }

type dashboardReleaseNoteData struct {
	releaseNoteData
}

func (_ *dashboardReleaseNoteData) Fill(_ string) error { return nil }
func (_ *dashboardReleaseNoteData) Template() string {
	return fmt.Sprintf(defaultReleaseNoteTemplate, dashboardRepo)
}
func (_ *dashboardReleaseNoteData) Repo() string { return dashboardRepo }

type cliReleaseNoteData struct {
	releaseNoteData
}

func (_ *cliReleaseNoteData) Fill(_ string) error { return nil }
func (_ *cliReleaseNoteData) Template() string {
	return fmt.Sprintf(defaultReleaseNoteTemplate, cliRepo)
}
func (_ *cliReleaseNoteData) Repo() string { return cliRepo }

func majMin(v string) (string, error) {
	majMin := semver.MajorMinor(v)
	if majMin == "" {
		return "", errors.New("version is not valid")
	}

	return majMin, nil
}

func trimPeriods(v string) string {
	return strings.ReplaceAll(v, ".", "")
}

// capitalize returns a new string whose first letter is capitalized.
func capitalize(s string) string {
	if runes := []rune(s); len(runes) > 0 {
		for i, r := range runes {
			if unicode.IsLetter(r) {
				runes[i] = unicode.ToUpper(r)
				s = string(runes)
				break
			}
		}
	}

	return s
}

// GenReleaseNotes genereates release notes based on the given milestone,
// previous milestone, and repository.
func GenReleaseNotes(ctx context.Context, owner, repo, milestone, prevMilestone string, client *github.Client) (*bytes.Buffer, error) {
	funcMap := template.FuncMap{
		"majMin":      majMin,
		"trimPeriods": trimPeriods,
		"split":       strings.Split,
		"capitalize":  capitalize,
	}
	const templateName = "release-notes"
	tmpl := template.New(templateName).Funcs(funcMap)
	tmpl = template.Must(tmpl.Parse(changelogTemplate))

	content, err := repository.RetrieveChangeLogContents(ctx, client, owner, repo, prevMilestone, milestone)
	if err != nil {
		return nil, err
	}

	// account for processing against an rc
	milestoneNoRC := milestone
	idx := strings.Index(milestone, "-rc")
	if idx != -1 {
		tmpMilestone := []rune(milestone)
		tmpMilestone = append(tmpMilestone[0:idx], tmpMilestone[idx+4:]...)
		milestoneNoRC = string(tmpMilestone)
	}

	k8sVersion := strings.Split(milestoneNoRC, "+")[0]
	markdownVersion := strings.ReplaceAll(k8sVersion, ".", "")
	tmp := strings.Split(strings.ReplaceAll(k8sVersion, "v", ""), ".")
	var majorMinor string
	if len(tmp) > 1 {
		majorMinor = tmp[0] + "." + tmp[1]
	} else {
		// for master branch
		majorMinor = tmp[0]
	}

	changeLogSince := strings.ReplaceAll(strings.Split(prevMilestone, "+")[0], ".", "")
	sqliteVersionK3S := goModLibVersion("go-sqlite3", repo, milestone)
	sqliteVersionBinding := sqliteVersionBinding(sqliteVersionK3S)
	helmControllerVersion := goModLibVersion("helm-controller", repo, milestone)
	coreDNSVersion := imageTagVersion("coredns", repo, milestone)
	cgData := changeLogData{
		PrevMilestone: prevMilestone,
		Content:       content,
	}

	var rd releaseNote
	commonRD := releaseNoteData{
		Milestone:        milestoneNoRC,
		MajorMinor:       majorMinor,
		ChangeLogVersion: markdownVersion,
		ChangeLogData:    cgData,
	}

	switch repo {
	case k3sRepo:
		rd = &k3sReleaseNoteData{
			releaseNoteData:       commonRD,
			K8sVersion:            k8sVersion,
			ChangeLogSince:        changeLogSince,
			SQLiteVersion:         sqliteVersionBinding,
			SQLiteVersionReplaced: strings.ReplaceAll(sqliteVersionBinding, ".", "_"),
			HelmControllerVersion: helmControllerVersion,
			CoreDNSVersion:        coreDNSVersion,
		}

	case rke2Repo:
		rd = &rke2ReleaseNoteData{
			releaseNoteData:       commonRD,
			K8sVersion:            k8sVersion,
			HelmControllerVersion: helmControllerVersion,
			CoreDNSVersion:        coreDNSVersion,
		}

	case uiRepo:
		rd = &uiReleaseNoteData{
			releaseNoteData: commonRD,
		}

	case dashboardRepo:
		rd = &dashboardReleaseNoteData{
			releaseNoteData: commonRD,
		}

	case cliRepo:
		rd = &cliReleaseNoteData{
			releaseNoteData: commonRD,
		}
	default:
		return nil, errors.New("invalid repo: it must be k3s, rke2, ui, dashboard or cli, received " + repo)
	}

	if err := rd.Fill(milestone); err != nil {
		return nil, err
	}

	tmpl = template.Must(tmpl.Parse(rd.Template()))

	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, rd.Repo(), rd); err != nil {
		return nil, err
	}

	return b, nil
}

const (
	containerdModLib   = "containerd/containerd"
	containerdV2ModLib = containerdModLib + "/v2"
)

// CheckUpstreamRelease takes the given org, repo, and tags and checks
// for the tags' existence.
func CheckUpstreamRelease(ctx context.Context, client *github.Client, org, repo string, tags []string) (map[string]bool, error) {
	releases := make(map[string]bool, len(tags))

	for _, tag := range tags {
		_, _, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
		if err != nil {
			switch err := err.(type) {
			case *github.ErrorResponse:
				if err.Response.StatusCode != http.StatusNotFound {
					return nil, err
				}
				releases[tag] = false
				continue
			default:
				return nil, err
			}
		}

		releases[tag] = true
	}

	return releases, nil
}

func KubernetesGoVersion(ctx context.Context, client *github.Client, version string) (string, error) {
	var githubError *github.ErrorResponse

	file, _, _, err := client.Repositories.GetContents(ctx, "kubernetes", "kubernetes", ".go-version", &github.RepositoryContentGetOptions{
		Ref: version,
	})
	if err != nil {
		if errors.As(err, &githubError) {
			if githubError.Response.StatusCode == http.StatusNotFound {
				return "", err
			}
		}
		return "", err
	}

	goVersion, err := file.GetContent()
	if err != nil {
		return "", err
	}

	return strings.Trim(goVersion, "\n"), nil
}

// VerifyAssets checks the number of assets for the
// given release and indicates if the expected number has
// been met.
func VerifyAssets(ctx context.Context, client *github.Client, owner, repo string, tags []string) (map[string]bool, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}

	releases := make(map[string]bool, len(tags))

	const (
		rke2Assets    = 50
		k3sAssets     = 23
		rke2Packaging = 23
	)

	for _, tag := range tags {
		if tag == "" {
			continue
		}

		release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
		if err != nil {
			switch err := err.(type) {
			case *github.ErrorResponse:
				if err.Response.StatusCode != http.StatusNotFound {
					return nil, err
				}
				releases[tag] = false
				continue
			default:
				return nil, err
			}
		}

		if repo == rke2Repo && len(release.Assets) == rke2Assets {
			releases[tag] = true
		}

		if repo == k3sRepo && len(release.Assets) == k3sAssets {
			releases[tag] = true
		}

		if repo == "rke2-packing" && len(release.Assets) == rke2Packaging {
			releases[tag] = true
		}
	}

	return releases, nil
}

// ListAssets gets all assets associated with the given release.
func ListAssets(ctx context.Context, client *github.Client, owner, repo, tag string) ([]*github.ReleaseAsset, error) {
	if tag == "" {
		return nil, errors.New("invalid tag provided")
	}

	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		switch err := err.(type) {
		case *github.ErrorResponse:
			if err.Response.StatusCode != http.StatusNotFound {
				return nil, err
			}
		default:
			return nil, err
		}
	}

	return release.Assets, nil
}

// DeleteAssetsByRelease deletes all release assets for the given release tag.
func DeleteAssetsByRelease(ctx context.Context, client *github.Client, owner, repo, tag string) error {
	if tag == "" {
		return errors.New("invalid tag provided")
	}

	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		switch err := err.(type) {
		case *github.ErrorResponse:
			if err.Response.StatusCode != http.StatusNotFound {
				return err
			}
		default:
			return err
		}
	}

	for _, asset := range release.Assets {
		if _, err := client.Repositories.DeleteReleaseAsset(ctx, owner, repo, asset.GetID()); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAssetByID deletes the release asset associated with the given ID.
func DeleteAssetByID(ctx context.Context, client *github.Client, owner, repo, tag string, id int64) error {
	if tag == "" {
		return errors.New("invalid tag provided")
	}

	if _, err := client.Repositories.DeleteReleaseAsset(ctx, owner, repo, id); err != nil {
		return err
	}

	return nil
}

func goModLibVersion(libraryName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"
	if repo == rke2Repo {
		repoName = "rancher/rke2"
	}

	goModURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/go.mod"

	resp, err := http.Get(goModURL)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", goModURL, err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, goModURL)
		return ""
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Debugf("read body error: %v", err)
		return ""
	}

	modFile, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		logrus.Debugf("failed to parse go.mod file: %v", err)
		return ""
	}

	// use replace section if found
	for _, replace := range modFile.Replace {
		if strings.Contains(replace.Old.Path, libraryName) {
			return replace.New.Version
		}
	}

	// if replace not found search in require
	for _, require := range modFile.Require {
		if strings.Contains(require.Mod.Path, libraryName) {
			return require.Mod.Version
		}
	}
	logrus.Debugf("library %s not found", libraryName)

	return ""
}

func buildScriptVersion(varName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"

	if repo == rke2Repo {
		repoName = "rancher/rke2"
	}

	buildScriptURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/version.sh"

	const regex = `(?P<version>v[\d\.]+(-k3s.\w*)?)`
	submatch := findInURL(buildScriptURL, regex, varName, true)

	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

func dockerfileVersion(chartName, repo, branchVersion string) string {
	if strings.Contains(repo, "k3s") {
		return ""
	}

	const (
		repoName = "rancher/rke2"
		regex    = `FROM\s+[\w-]+/[\w-]+:(.*?)(-build.*)?\s`
	)

	dockerfileURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/Dockerfile"

	submatch := findInURL(dockerfileURL, regex, chartName, true)
	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

func imageTagVersion(ImageName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"

	imageListURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/airgap/image-list.txt"
	if repo == rke2Repo {
		repoName = "rancher/rke2"
		imageListURL = "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/build-images"
	}

	const regex = `:(.*)(-build.*)?`
	submatch := findInURL(imageListURL, regex, ImageName, true)

	if len(submatch) > 1 {
		if strings.Contains(submatch[1], "-build") {
			versionSplit := strings.Split(submatch[1], "-")
			return versionSplit[0]
		}
		return submatch[1]
	}

	return ""
}

func sqliteVersionBinding(sqliteVersion string) string {
	sqliteBindingURL := "https://raw.githubusercontent.com/mattn/go-sqlite3/" + sqliteVersion + "/sqlite3-binding.h"
	const (
		regex = `\"(.*)\"`
		word  = "SQLITE_VERSION"
	)

	submatch := findInURL(sqliteBindingURL, regex, word, true)
	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

func createCalicoURL(calicoVersion string) string {
	const (
		regex    = `\"(.*)\"`
		notFound = "Page Not Found"
	)

	versionRegex := regexp.MustCompile(`^v(\d+\.\d+)(?:\.\d+)?$`)

	formattedVersion := calicoVersion

	// Check if the version matches the pattern
	if versionRegex.MatchString(calicoVersion) {
		matches := versionRegex.FindStringSubmatch(calicoVersion)
		if len(matches) == 2 {
			formattedVersion = "v" + matches[1]
		}
	}

	calicoArchiveURL := "https://projectcalico.docs.tigera.io/archive/" + formattedVersion + "/release-notes/#" + strings.Trim(calicoVersion, "")

	// check if doesn't exists content for archive url
	submatch := findInURL(calicoArchiveURL, regex, notFound, false)
	if len(submatch) > 1 {
		return "https://docs.tigera.io/calico/latest/release-notes/#" + formattedVersion
	}

	return calicoArchiveURL
}

// findInURL will get and scan a url to find a slice submatch for all the words that matches a regex
// if the regex is empty then it will return the lines in a file that matches the str
func findInURL(url, regex, str string, checkStatusCode bool) []string {
	var submatch []string

	client := httpecm.NewClient(defaultTimeout)
	resp, err := client.Get(url)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", url, err)
		return nil
	}
	defer resp.Body.Close()

	if checkStatusCode && resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, url)
		return nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Debugf("read body error: %v", err)
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, str) {
			if regex == "" {
				submatch = append(submatch, line)
			} else {
				re := regexp.MustCompile(regex)
				submatch = re.FindStringSubmatch(line)
				if len(submatch) > 1 {
					return submatch
				}
			}
		}
	}

	return submatch
}

// LatestRC will get the latest rc created for the k8s version in either rke2 or k3s
func LatestRC(ctx context.Context, owner, repo, k8sVersion, projectSuffix string, client *github.Client) (*string, error) {
	var rcs []*github.RepositoryRelease

	allReleases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{
		Page:    0,
		PerPage: 40,
	})
	if err != nil {
		return nil, err
	}

	for _, release := range allReleases {
		if strings.Contains(*release.TagName, k8sVersion+"-rc") && strings.Contains(*release.TagName, projectSuffix) {
			rcs = append(rcs, release)
		}
	}

	return latestRelease(rcs), nil
}

func LatestPreRelease(ctx context.Context, client *github.Client, owner, repo, version, preReleaseSuffix string) (*string, error) {
	var versions []*github.RepositoryRelease

	allReleases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{
		Page:    0,
		PerPage: 40,
	})
	if err != nil {
		return nil, err
	}

	for _, release := range allReleases {
		if strings.Contains(*release.TagName, version+"-"+preReleaseSuffix) {
			versions = append(versions, release)
		}
	}

	return latestRelease(versions), nil
}

// StatsMonthly
type StatsMonthly struct {
	Count    int
	Captains []string
	Tags     []string
}

// RelStats
type RelStats struct {
	Count   int
	Monthly map[time.Month]StatsMonthly
}

// StatsData
type StatsData struct {
	Total    int64            `json:"total"`
	Data     map[int]RelStats `json:"data"`
	Captains map[string]int   `json:"captains"`
}

// dedup creates and returns a new slices based on the given slice
// but with duplicate entries removed.
func dedup(slice []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

// Stats collects and processes information regarding a set of releases for the given repo
// over the given period of time.
func Stats(ctx context.Context, client *github.Client, startDate, endDate time.Time, owner, repo string) (*StatsData, error) {
	if endDate.Before(startDate) {
		return nil, errors.New("end date before start date")
	}

	sd := StatsData{
		Data:     make(map[int]RelStats),
		Captains: make(map[string]int),
	}

	lo := github.ListOptions{
		PerPage: 100,
	}
	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &lo)
		if err != nil {
			return nil, err
		}

		for _, release := range releases {
			releaseDate := release.GetCreatedAt().Time
			if releaseDate.After(startDate) && (releaseDate.Before(endDate) || releaseDate.Equal(endDate)) {
				sd.Total++

				if _, ok := sd.Data[int(release.CreatedAt.Year())]; !ok {
					sd.Data[int(release.CreatedAt.Year())] = RelStats{
						Count: 1,
						Monthly: map[time.Month]StatsMonthly{
							release.CreatedAt.Month(): {
								Count: 1,
								Captains: []string{
									*release.Author.Login,
								},
								Tags: []string{
									*release.Name,
								},
							},
						},
					}
					continue
				}

				rs := sd.Data[int(release.CreatedAt.Year())]
				rs.Count++

				mon := rs.Monthly[release.CreatedAt.Month()]
				mon.Count++
				mon.Captains = append(mon.Captains, *release.Author.Login)
				mon.Tags = append(mon.Tags, *release.Name)

				rs.Monthly[release.CreatedAt.Month()] = mon

				sd.Data[int(release.CreatedAt.Year())] = rs

				if release.Author.Login != nil {
					if _, ok := sd.Captains[*release.Author.Login]; !ok {
						sd.Captains[*release.Author.Login]++
						continue
					}
					sd.Captains[*release.Author.Login]++
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		lo.Page = resp.NextPage
	}

	for year := range sd.Data {
		months := make([]int, 0, len(sd.Data[year].Monthly))
		for k := range sd.Data[year].Monthly {
			months = append(months, int(k))
		}
		sort.Ints(months)

		for _, m := range months {
			mon := time.Month(m)
			tmp := sd.Data[year].Monthly[mon]
			tmp.Captains = dedup(tmp.Captains)
			sd.Data[year].Monthly[mon] = tmp
		}
	}

	return &sd, nil
}

func latestRelease(versions []*github.RepositoryRelease) *string {
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].PublishedAt.Before(versions[j].PublishedAt.Time)
	})

	if len(versions) == 0 {
		return nil
	}

	return versions[len(versions)-1].TagName
}

// rke2ChartVersion will return the version of the rke2 chart from the chart versions file
func rke2ChartsVersion(branchVersion string) (map[string]chart, error) {
	chartVersionsURL := "https://raw.githubusercontent.com/rancher/rke2/" + branchVersion + "/charts/" + rke2ChartsVersionsFile

	client := httpecm.NewClient(defaultTimeout)
	resp, err := client.Get(chartVersionsURL)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", chartVersionsURL, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, err)
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var c charts

	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	chartsData := make(map[string]chart, len(c.Charts))
	for _, chart := range c.Charts {
		chartsData[filepath.Base(chart.Filename)] = chart
	}

	return chartsData, nil
}

var changelogTemplate = `
{{- define "changelog" -}}
## Changes since {{.ChangeLogData.PrevMilestone}}:
{{range .ChangeLogData.Content}}
* {{ capitalize .Title }} [(#{{.Number}})]({{.URL}})
{{- $lines := split .Note "\n"}}
{{- range $i, $line := $lines}}
{{- if ne $line "" }}
  * {{ capitalize $line }}
{{- end}}
{{- end}}
{{- end}}
{{- end}}`

const rke2ReleaseNoteTemplate = `
{{- define "rke2" -}}
<!-- {{.Milestone}} -->

This release updates Kubernetes to {{.K8sVersion}}.

**Important Note**

If your server (control-plane) nodes were not started with the ` + "`--token`" + ` CLI flag or config file key, a randomized token was generated during initial cluster startup. This key is used both for joining new nodes to the cluster, and for encrypting cluster bootstrap data within the datastore. Ensure that you retain a copy of this token, as is required when restoring from backup.

You may retrieve the token value from any server already joined to the cluster:
` + "```bash" + `
cat /var/lib/rancher/rke2/server/token
` + "```" + `

{{ template "changelog" . }}


## Charts Versions
| Component | Version |
| --- | --- |
| rke2-cilium | [{{.CiliumChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-cilium/rke2-cilium-{{.CiliumChartVersion}}.tgz) |
| rke2-canal | [{{.CanalChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-canal/rke2-canal-{{.CanalChartVersion}}.tgz) |
| rke2-calico | [{{.CalicoChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-calico/rke2-calico-{{.CalicoChartVersion}}.tgz) |
| rke2-calico-crd | [{{.CalicoCRDChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-calico/rke2-calico-crd-{{.CalicoCRDChartVersion}}.tgz) |
| rke2-coredns | [{{.CoreDNSChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-coredns/rke2-coredns-{{.CoreDNSChartVersion}}.tgz) |
| rke2-ingress-nginx | [{{.IngressNginxChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-ingress-nginx/rke2-ingress-nginx-{{.IngressNginxChartVersion}}.tgz) |
| rke2-metrics-server | [{{.MetricsServerChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-metrics-server/rke2-metrics-server-{{.MetricsServerChartVersion}}.tgz) |
| rancher-vsphere-csi | [{{.VsphereCSIChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rancher-vsphere-csi/rancher-vsphere-csi-{{.VsphereCSIChartVersion}}.tgz) |
| rancher-vsphere-cpi | [{{.VsphereCPIChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rancher-vsphere-cpi/rancher-vsphere-cpi-{{.VsphereCPIChartVersion}}.tgz) |
| harvester-cloud-provider | [{{.HarvesterCloudProviderChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/harvester-cloud-provider/harvester-cloud-provider-{{.HarvesterCloudProviderChartVersion}}.tgz) |
| harvester-csi-driver | [{{.HarvesterCSIDriverChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/harvester-cloud-provider/harvester-csi-driver-{{.HarvesterCSIDriverChartVersion}}.tgz) |
| rke2-snapshot-controller | [{{.SnapshotControllerChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-snapshot-controller/rke2-snapshot-controller-{{.SnapshotControllerChartVersion}}.tgz) |
| rke2-snapshot-controller-crd | [{{.SnapshotControllerCRDChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-snapshot-controller/rke2-snapshot-controller-crd-{{.SnapshotControllerCRDChartVersion}}.tgz) |
| rke2-snapshot-validation-webhook | [{{.SnapshotValidationWebhookChartVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-snapshot-validation-webhook/rke2-snapshot-validation-webhook-{{.SnapshotValidationWebhookChartVersion}}.tgz) |
| rke2-traefik | [{{.TraefikVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-traefik/rke2-traefik-{{.TraefikVersion}}.tgz) |
| rke2-traefik-crd | [{{.TraefikCRDVersion}}](https://github.com/rancher/rke2-charts/raw/main/assets/rke2-traefik/rke2-traefik-crd-{{.TraefikCRDVersion}}.tgz) |


## Packaged Component Versions
| Component | Version |
| --- | --- |
| Kubernetes | [{{.K8sVersion}}](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.MajorMinor}}.md#{{.ChangeLogVersion}}) |
| Etcd | [{{.EtcdVersion}}](https://github.com/k3s-io/etcd/releases/tag/{{.EtcdVersion}}) |
| Containerd | [{{.ContainerdVersion}}](https://github.com/k3s-io/containerd/releases/tag/{{.ContainerdVersion}}) |
| Runc | [{{.RuncVersion}}](https://github.com/opencontainers/runc/releases/tag/{{.RuncVersion}}) |
| Metrics-server | [{{.MetricsServerVersion}}](https://github.com/kubernetes-sigs/metrics-server/releases/tag/{{.MetricsServerVersion}}) |
| CoreDNS | [{{.CoreDNSVersion}}](https://github.com/coredns/coredns/releases/tag/{{.CoreDNSVersion}}) |
| Ingress-Nginx | [{{.IngressNginxVersion}}](https://github.com/rancher/ingress-nginx/releases/tag/{{.IngressNginxVersion}}) |
| Helm-controller | [{{.HelmControllerVersion}}](https://github.com/k3s-io/helm-controller/releases/tag/{{.HelmControllerVersion}}) |
{{if .TraefikImageVersion}}| Traefik | [{{.TraefikImageVersion}}](https://github.com/traefik/traefik/releases/tag/{{.TraefikImageVersion}}) |{{end}}

### Available CNIs
| Component | Version | FIPS Compliant |
| --- | --- | --- |
| Canal (Default) | [Flannel {{.FlannelVersion}}](https://github.com/flannel-io/flannel/releases/tag/{{.FlannelVersion}})<br/>[Calico {{.CanalCalicoVersion}}]({{.CanalCalicoURL}}) | Yes |
| Calico | [{{.CalicoVersion}}]({{.CalicoURL}}) | No |
| Cilium | [{{.CiliumVersion}}](https://github.com/cilium/cilium/releases/tag/{{.CiliumVersion}}) | No |
| Multus | [{{.MultusVersion}}](https://github.com/k8snetworkplumbingwg/multus-cni/releases/tag/{{.MultusVersion}}) | No |

## Helpful Links

As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/rke2/issues/new)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://docs.rke2.io) for guidance on how to get started.
{{ end }}`

const k3sReleaseNoteTemplate = `
{{- define "k3s" -}}
<!-- {{.Milestone}} -->

This release updates Kubernetes to {{.K8sVersion}}, and fixes a number of issues.

For more details on what's new, see the [Kubernetes release notes](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.MajorMinor}}.md#changelog-since-{{.ChangeLogSince}}).

{{ template "changelog" . }}

## Embedded Component Versions
| Component | Version |
|---|---|
| Kubernetes | [{{.K8sVersion}}](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.MajorMinor}}.md#{{.ChangeLogVersion}}) |
| Kine | [{{.KineVersion}}](https://github.com/k3s-io/kine/releases/tag/{{.KineVersion}}) |
| SQLite | [{{.SQLiteVersion}}](https://sqlite.org/releaselog/{{.SQLiteVersionReplaced}}.html) |
| Etcd | [{{.EtcdVersion}}](https://github.com/k3s-io/etcd/releases/tag/{{.EtcdVersion}}) |
| Containerd | [{{.ContainerdVersion}}](https://github.com/k3s-io/containerd/releases/tag/{{.ContainerdVersion}}) |
| Runc | [{{.RuncVersion}}](https://github.com/opencontainers/runc/releases/tag/{{.RuncVersion}}) |
| Flannel | [{{.FlannelVersion}}](https://github.com/flannel-io/flannel/releases/tag/{{.FlannelVersion}}) | 
| Metrics-server | [{{.MetricsServerVersion}}](https://github.com/kubernetes-sigs/metrics-server/releases/tag/{{.MetricsServerVersion}}) |
| Traefik | [v{{.TraefikVersion}}](https://github.com/traefik/traefik/releases/tag/v{{.TraefikVersion}}) |
| CoreDNS | [v{{.CoreDNSVersion}}](https://github.com/coredns/coredns/releases/tag/v{{.CoreDNSVersion}}) | 
| Helm-controller | [{{.HelmControllerVersion}}](https://github.com/k3s-io/helm-controller/releases/tag/{{.HelmControllerVersion}}) |
| Local-path-provisioner | [{{.LocalPathProvisionerVersion}}](https://github.com/rancher/local-path-provisioner/releases/tag/{{.LocalPathProvisionerVersion}}) |

## Helpful Links
As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/k3s/issues/new/choose)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://rancher.com/docs/k3s/latest/en/) for guidance on how to get started or to dive deep into K3s.
- [Read how you can contribute here](https://github.com/rancher/k3s/blob/master/CONTRIBUTING.md)
{{ end }}`

// This is the default template for release notes, for
// releases that don't have a "specific" structure for
// their releases(e.g default generated release notes).
const defaultReleaseNoteTemplate = `
{{- define "%s" -}}
<!-- {{.Milestone}} -->

{{ template "changelog" . }}
{{ end }}`
