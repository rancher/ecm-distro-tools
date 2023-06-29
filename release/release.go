package release

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

const (
	k3sRepo  = "k3s"
	rke2Repo = "rke2"
)

type changeLogData struct {
	PrevMilestone string
	Content       []repository.ChangeLog
}

type rke2ReleaseNoteData struct {
	Milestone             string
	K8sVersion            string
	MajorMinor            string
	EtcdVersion           string
	ContainerdVersion     string
	RuncVersion           string
	MetricsServerVersion  string
	CoreDNSVersion        string
	ChangeLogVersion      string
	IngressNginxVersion   string
	HelmControllerVersion string
	FlannelVersion        string
	CanalCalicoVersion    string
	CalicoVersion         string
	CiliumVersion         string
	MultusVersion         string
	ComponentsTable       string
	CNIsTable             string
	ChangeLogData         changeLogData
}

type k3sReleaseNoteData struct {
	Milestone                   string
	K8sVersion                  string
	MajorMinor                  string
	ChangeLogSince              string
	ChangeLogVersion            string
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
	ComponentsTable             string
	ChangeLogData               changeLogData
}

var componentMarkdownLink map[string]string = map[string]string{
	"k8s":                  "[%[1]s](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-%[2]s.md#%[3]s)",
	"kine":                 "[%[1]s](https://github.com/k3s-io/kine/releases/tag/%[1]s)",
	"sqlite":               "[%[1]s](https://sqlite.org/releaselog/%[1]s.html)",
	"etcd":                 "[%[1]s](https://github.com/k3s-io/etcd/releases/tag/%[1]s)",
	"containerd":           "[%[1]s](https://github.com/k3s-io/containerd/releases/tag/%[1]s)",
	"runc":                 "[%[1]s](https://github.com/opencontainers/runc/releases/tag/%[1]s)",
	"flannel":              "[%[1]s](https://github.com/flannel-io/flannel/releases/tag/%[1]s)",
	"metricsServer":        "[%[1]s](https://github.com/kubernetes-sigs/metrics-server/releases/tag/%[1]s)",
	"traefik":              "[v%[1]s](https://github.com/traefik/traefik/releases/tag/v%[1]s)",
	"coreDNS":              "[%[1]s](https://github.com/coredns/coredns/releases/tag/v%[1]s)",
	"helmController":       "[%[1]s](https://github.com/k3s-io/helm-controller/releases/tag/%[1]s)",
	"localPathProvisioner": "[%[1]s](https://github.com/rancher/local-path-provisioner/releases/tag/%[1]s)",
	"ingressNginx":         "[%[1]s](https://github.com/kubernetes/ingress-nginx/releases/tag/helm-chart-%[1]s)",
	"canalDefault":         "[Flannel %[1]s](https://github.com/k3s-io/flannel/releases/tag/%[1]s)<br/>[Calico %[2]s](https://projectcalico.docs.tigera.io/archive/%[3]s/release-notes/#%[4]s)",
	"calico":               "[%[1]s](https://projectcalico.docs.tigera.io/archive/%[2]s/release-notes/#%[3]s)",
	"cilium":               "[%[1]s](https://github.com/cilium/cilium/releases/tag/%[1]s)",
	"multus":               "[%[1]s](https://github.com/k8snetworkplumbingwg/multus-cni/releases/tag/%[1]s)",
}

func majMin(v string) (string, error) {
	majMin := semver.MajorMinor(v)
	if majMin == "" {
		return "", errors.New("version is not valid")
	}
	return majMin, nil
}

func trimPeriods(v string) string {
	return strings.Replace(v, ".", "", -1)
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
func GenReleaseNotes(ctx context.Context, repo, milestone, prevMilestone string, client *github.Client) (*bytes.Buffer, error) {
	funcMap := template.FuncMap{
		"majMin":      majMin,
		"trimPeriods": trimPeriods,
		"split":       strings.Split,
		"capitalize":  capitalize,
	}
	const templateName = "release-notes"
	tmpl := template.New(templateName).Funcs(funcMap)
	tmpl = template.Must(tmpl.Parse(changelogTemplate))

	content, err := repository.RetrieveChangeLogContents(ctx, client, repo, prevMilestone, milestone)
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
	markdownVersion := strings.Replace(k8sVersion, ".", "", -1)
	tmp := strings.Split(strings.Replace(k8sVersion, "v", "", -1), ".")
	majorMinor := tmp[0] + "." + tmp[1]
	changeLogSince := strings.Replace(strings.Split(prevMilestone, "+")[0], ".", "", -1)
	sqliteVersionK3S := goModLibVersion("go-sqlite3", repo, milestone)
	sqliteVersionBinding := sqliteVersionBinding(sqliteVersionK3S)
	helmControllerVersion := goModLibVersion("helm-controller", repo, milestone)
	coreDNSVersion := imageTagVersion("coredns", repo, milestone)
	cgData := changeLogData{
		PrevMilestone: prevMilestone,
		Content:       content,
	}

	if repo == k3sRepo {
		return genK3SReleaseNotes(
			tmpl,
			milestone,
			k3sReleaseNoteData{
				Milestone:             milestoneNoRC,
				MajorMinor:            majorMinor,
				K8sVersion:            k8sVersion,
				ChangeLogVersion:      markdownVersion,
				ChangeLogSince:        changeLogSince,
				SQLiteVersion:         sqliteVersionBinding,
				SQLiteVersionReplaced: strings.ReplaceAll(sqliteVersionBinding, ".", "_"),
				HelmControllerVersion: helmControllerVersion,
				ChangeLogData:         cgData,
				CoreDNSVersion:        coreDNSVersion,
			},
		)
	} else if repo == rke2Repo {
		return genRKE2ReleaseNotes(
			tmpl,
			milestone,
			rke2ReleaseNoteData{
				MajorMinor:            majorMinor,
				Milestone:             milestoneNoRC,
				ChangeLogVersion:      markdownVersion,
				K8sVersion:            k8sVersion,
				HelmControllerVersion: helmControllerVersion,
				CoreDNSVersion:        coreDNSVersion,
				ChangeLogData:         cgData,
			},
		)
	}
	return nil, errors.New("invalid repo, it must be either k3s or rke2")
}

func genK3SReleaseNotes(tmpl *template.Template, milestone string, rd k3sReleaseNoteData) (*bytes.Buffer, error) {
	tmpl = template.Must(tmpl.Parse(k3sReleaseNoteTemplate))
	var runcVersion string
	var containerdVersion string

	if semver.Compare(rd.K8sVersion, "v1.24.0") == 1 &&
		semver.Compare(rd.K8sVersion, "v1.26.5") == -1 {
		containerdVersion = buildScriptVersion("VERSION_CONTAINERD", k3sRepo, milestone)
	} else {
		containerdVersion = goModLibVersion("containerd/containerd", k3sRepo, milestone)
	}

	if rd.MajorMinor == "1.23" {
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

	componentsHeader := []string{"Component", "Version"}
	componentsValues := [][]string{
		{"Kubernetes", fmt.Sprintf(componentMarkdownLink["k8s"], rd.K8sVersion, rd.MajorMinor, rd.ChangeLogVersion)},
		{"Kine", fmt.Sprintf(componentMarkdownLink["kine"], rd.KineVersion)},
		{"SQLite", fmt.Sprintf(componentMarkdownLink["sqlite"], rd.SQLiteVersionReplaced)},
		{"Etcd", fmt.Sprintf(componentMarkdownLink["etcd"], rd.EtcdVersion)},
		{"Containerd", fmt.Sprintf(componentMarkdownLink["containerd"], rd.ContainerdVersion)},
		{"Runc", fmt.Sprintf(componentMarkdownLink["runc"], rd.RuncVersion)},
		{"Flannel", fmt.Sprintf(componentMarkdownLink["flannel"], rd.FlannelVersion)},
		{"Metrics-server", fmt.Sprintf(componentMarkdownLink["metricsServer"], rd.MetricsServerVersion)},
		{"Traefik", fmt.Sprintf(componentMarkdownLink["traefik"], rd.TraefikVersion)},
		{"CoreDNS", fmt.Sprintf(componentMarkdownLink["coreDNS"], rd.CoreDNSVersion)},
		{"Helm-controller", fmt.Sprintf(componentMarkdownLink["helmController"], rd.HelmControllerVersion)},
		{"Local-path-provisioner", fmt.Sprintf(componentMarkdownLink["localPathProvisioner"], rd.LocalPathProvisionerVersion)},
	}

	componentsTable, err := NewMarkdownTable(componentsHeader, componentsValues)
	if err != nil {
		return nil, err
	}
	rd.ComponentsTable = componentsTable.String()

	buf := bytes.NewBuffer(nil)
	err = tmpl.ExecuteTemplate(buf, k3sRepo, rd)
	return buf, err
}

func genRKE2ReleaseNotes(tmpl *template.Template, milestone string, rd rke2ReleaseNoteData) (*bytes.Buffer, error) {
	tmpl = template.Must(tmpl.Parse(rke2ReleaseNoteTemplate))
	var containerdVersion string

	if rd.MajorMinor == "1.23" {
		containerdVersion = goModLibVersion("containerd/containerd", rke2Repo, milestone)
	} else {
		containerdVersion = dockerfileVersion("hardened-containerd", rke2Repo, milestone)
	}

	rd.EtcdVersion = buildScriptVersion("ETCD_VERSION", rke2Repo, milestone)
	rd.RuncVersion = dockerfileVersion("hardened-runc", rke2Repo, milestone)
	rd.CanalCalicoVersion = imageTagVersion("hardened-calico", rke2Repo, milestone)
	rd.CiliumVersion = imageTagVersion("cilium-cilium", rke2Repo, milestone)
	rd.ContainerdVersion = containerdVersion
	rd.MetricsServerVersion = imageTagVersion("metrics-server", rke2Repo, milestone)
	rd.IngressNginxVersion = dockerfileVersion("rke2-ingress-nginx", rke2Repo, milestone)
	rd.FlannelVersion = imageTagVersion("flannel", rke2Repo, milestone)
	rd.CalicoVersion = imageTagVersion("calico-node", rke2Repo, milestone)
	rd.MultusVersion = imageTagVersion("multus-cni", rke2Repo, milestone)

	componentsHeader := []string{"Component", "Version"}
	componentsValues := [][]string{
		{"Kubernetes", fmt.Sprintf(componentMarkdownLink["k8s"], rd.K8sVersion, rd.MajorMinor, rd.ChangeLogVersion)},
		{"Etcd", fmt.Sprintf(componentMarkdownLink["etcd"], rd.EtcdVersion)},
		{"Containerd", fmt.Sprintf(componentMarkdownLink["containerd"], rd.ContainerdVersion)},
		{"Runc", fmt.Sprintf(componentMarkdownLink["runc"], rd.RuncVersion)},
		{"Metrics-server", fmt.Sprintf(componentMarkdownLink["metricsServer"], rd.MetricsServerVersion)},
		{"CoreDNS", fmt.Sprintf(componentMarkdownLink["coreDNS"], rd.CoreDNSVersion)},
		{"Ingress-Nginx", fmt.Sprintf(componentMarkdownLink["ingressNginx"], rd.IngressNginxVersion)},
		{"Helm-controller", fmt.Sprintf(componentMarkdownLink["helmController"], rd.HelmControllerVersion)},
	}

	componentsTable, err := NewMarkdownTable(componentsHeader, componentsValues)
	if err != nil {
		return nil, err
	}
	rd.ComponentsTable = componentsTable.String()

	majMinCanalCalicoVersion, err := majMin(rd.CanalCalicoVersion)
	if err != nil {
		return nil, err
	}

	cnisHeader := []string{"Component", "Version", "FIPS Compliant"}
	cnisValues := [][]string{
		{"Canal (Default)", fmt.Sprintf(componentMarkdownLink["canalDefault"], rd.FlannelVersion, rd.CanalCalicoVersion, majMinCanalCalicoVersion, trimPeriods(rd.CanalCalicoVersion)), "Yes"},
		{"Calico", fmt.Sprintf(componentMarkdownLink["calico"], rd.CalicoVersion, majMinCanalCalicoVersion, trimPeriods(rd.CalicoVersion)), "No"},
		{"Cilium", fmt.Sprintf(componentMarkdownLink["cilium"], rd.CiliumVersion), "No"},
		{"Multus", fmt.Sprintf(componentMarkdownLink["multus"], rd.MultusVersion), "No"},
	}

	cnisTable, err := NewMarkdownTable(cnisHeader, cnisValues)
	if err != nil {
		return nil, err
	}
	rd.CNIsTable = cnisTable.String()

	buf := bytes.NewBuffer(nil)
	err = tmpl.ExecuteTemplate(buf, rke2Repo, rd)
	return buf, err
}

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
func VerifyAssets(ctx context.Context, client *github.Client, repo string, tags []string) (map[string]bool, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}

	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return nil, err
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

		release, _, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
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
func ListAssets(ctx context.Context, client *github.Client, repo, tag string) ([]*github.ReleaseAsset, error) {
	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		return nil, errors.New("invalid tag provided")
	}

	release, _, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
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
func DeleteAssetsByRelease(ctx context.Context, client *github.Client, repo, tag string) error {
	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return err
	}

	if tag == "" {
		return errors.New("invalid tag provided")
	}

	release, _, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
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
		if _, err := client.Repositories.DeleteReleaseAsset(ctx, org, repo, asset.GetID()); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAssetByID deletes the release asset associated with the given ID.
func DeleteAssetByID(ctx context.Context, client *github.Client, repo, tag string, id int64) error {
	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return err
	}

	if tag == "" {
		return errors.New("invalid tag provided")
	}

	if _, err := client.Repositories.DeleteReleaseAsset(ctx, org, repo, id); err != nil {
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

	b, err := ioutil.ReadAll(resp.Body)
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
	submatch := findInURL(buildScriptURL, regex, varName)

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
		regex    = `(?:FROM|RUN)\s(?:CHART_VERSION=\"|[\w-]+/[\w-]+:)(?P<version>.*?)([0-9][0-9])?(-build.*)?\"?\s`
	)

	dockerfileURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/Dockerfile"

	submatch := findInURL(dockerfileURL, regex, chartName)
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
	submatch := findInURL(imageListURL, regex, ImageName)

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

	submatch := findInURL(sqliteBindingURL, regex, word)
	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

// findInURL will get and scan a url to find a slice submatch for all the words that matches a regex
// if the regex is empty then it will return the lines in a file that matches the str
func findInURL(url, regex, str string) []string {
	var submatch []string

	resp, err := http.Get(url)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", url, err)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, url)
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Debugf("read body error: %v", err)
		return nil
	}
	defer resp.Body.Close()

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
func LatestRC(ctx context.Context, repo, k8sVersion string, client *github.Client) (string, error) {
	var rcs []*github.RepositoryRelease
	org, err := repository.OrgFromRepo(repo)
	if err != nil {
		return "", err
	}
	allReleases, _, err := client.Repositories.ListReleases(ctx, org, repo, &github.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, release := range allReleases {
		if strings.Contains(*release.Name, k8sVersion+"-rc") {
			rcs = append(rcs, release)
		}
	}
	sort.Slice(rcs, func(i, j int) bool {
		return rcs[i].PublishedAt.Time.Before(rcs[j].PublishedAt.Time)
	})

	return *rcs[len(rcs)-1].Name, nil

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

This release ... <FILL ME OUT!>

**Important Note**

If your server (control-plane) nodes were not started with the ` + "`--token`" + ` CLI flag or config file key, a randomized token was generated during initial cluster startup. This key is used both for joining new nodes to the cluster, and for encrypting cluster bootstrap data within the datastore. Ensure that you retain a copy of this token, as is required when restoring from backup.

You may retrieve the token value from any server already joined to the cluster:
` + "```bash" + `
cat /var/lib/rancher/rke2/server/token
` + "```" + `

{{ template "changelog" . }}

## Packaged Component Versions
{{ .ComponentsTable }}

### Available CNIs
{{ .CNIsTable }}

## Known Issues

- [#1447](https://github.com/rancher/rke2/issues/1447) - When restoring RKE2 from backup to a new node, you should ensure that all pods are stopped following the initial restore:

` + "```" + `bash
curl -sfL https://get.rke2.io | sudo INSTALL_RKE2_VERSION={{.Milestone}}
rke2 server \
  --cluster-reset \
  --cluster-reset-restore-path=<PATH-TO-SNAPSHOT> --token <token used in the original cluster>
rke2-killall.sh
systemctl enable rke2-server
systemctl start rke2-server
` + "```" + `

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
{{ .ComponentsTable }}

## Helpful Links
As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/k3s/issues/new/choose)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://rancher.com/docs/k3s/latest/en/) for guidance on how to get started or to dive deep into K3s.
- [Read how you can contribute here](https://github.com/rancher/k3s/blob/master/CONTRIBUTING.md)
{{ end }}`
